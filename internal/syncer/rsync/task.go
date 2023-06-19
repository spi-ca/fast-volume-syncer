package rsync

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"github.com/schollz/progressbar/v3"

	"github.com/avast/retry-go"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

var (
	rsyncUptodateFormat = regexp.MustCompile(`^(.+?)( is uptodate)?$`)
)

type Task struct {
	Arguments       []string
	SourcePath      string
	DestinationPath string
	Retry           args.RetryArgs
}

func (t *Task) Execute(ctx context.Context, fileList []returns.Fileinfo) error {

	if t.Retry.Attempts <= 0 {
		return t.execute(ctx, fileList)
	}

	retryOptionArgs := t.Retry.Assemble(ctx)
	retryFunc := func() error { return t.execute(ctx, fileList) }
	return retry.Do(retryFunc, retryOptionArgs...)
}

func (t *Task) handleRsyncStdin(writer io.WriteCloser, closer func(), fileList []returns.Fileinfo) {
	defer closer()
	if writer == nil {
		return
	}
	defer writer.Close()

	if len(fileList) == 0 {
		return
	}
	w := bufio.NewWriter(writer)
	addSep := false
	for _, entry := range fileList {
		mode := entry.Mode
		if mode.IsDir() {
			// ensure mode
			dirMode := mode.Perm() | 0o700

			dirPath := filepath.Join(t.DestinationPath, entry.Path)
			destExists := false
			if dirPath == t.DestinationPath {
				// 자기자신은 무시하자
				continue
			} else if destMode, err := os.Lstat(dirPath); err == nil {
				if destExists = destMode.IsDir(); !destExists {
					// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
					if err = os.RemoveAll(dirPath); err != nil {
						util.ErrLog.Printf("failed to cleanup path %s :%v", dirPath, err)
						continue
					}
				}
			} else if !os.IsNotExist(err) {
				util.ErrLog.Printf("failed to create directory %s(%s) :%v", dirPath, dirMode, err)
				continue
			}

			if destExists {
				// directory path 확보상태(이미 생성됨)
				err := os.Chmod(dirPath, dirMode)
				if err != nil {
					util.ErrLog.Printf("failed to change directory mode %s(%s) :%v", dirPath, dirMode, err)
				}
			} else {
				// directory path 확보상태(비어있음)
				err := os.MkdirAll(dirPath, dirMode)
				if err != nil {
					util.ErrLog.Printf("failed to create directory %s(%s) :%v", dirPath, dirMode, err)
				}
			}
		} else if mode.Type()&fs.ModeSymlink != 0 {
			linkPath := filepath.Join(t.DestinationPath, entry.Path)

			if destMode, err := os.Lstat(linkPath); err == nil {
				if destMode.Mode().Type()&fs.ModeSymlink != 0 {
					// 대상 path가 symlink mode가 아닌 경우 대상을 날린다.
					destLinkPath, readLinkErr := os.Readlink(linkPath)
					if readLinkErr == nil && destLinkPath == entry.SymlinkPath {
						continue
					}
				}

				if err = os.RemoveAll(linkPath); err != nil {
					util.ErrLog.Printf("failed to cleanup path %s :%v", linkPath, err)
					continue
				}
			} else if !os.IsNotExist(err) {
				util.ErrLog.Printf("failed to create symlink %s, cannot get filestat :%v", linkPath, err)
				continue
			}

			// directory보다 링크가 먼저 온 case
			dirPath := filepath.Dir(linkPath)
			if destMode, err := os.Lstat(dirPath); err == nil {
				if !destMode.IsDir() {
					// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
					if err = os.RemoveAll(dirPath); err != nil {
						util.ErrLog.Printf("failed to make a symbolic link(%s -> %s), failed to cleanup path %s :%v",
							linkPath, entry.SymlinkPath, dirPath, err)
						continue
					}
				}
			} else if !os.IsNotExist(err) {
				util.ErrLog.Printf("failed to make a symbolic link(%s -> %s), failed to get fileinfo %s :%v",
					linkPath, entry.SymlinkPath, dirPath, err)
				continue
			} else if err = os.MkdirAll(dirPath, 0o755); err != nil {
				util.ErrLog.Printf("failed to make a symbolic link(%s -> %s), failed to create directory %s :%v",
					linkPath, entry.SymlinkPath,
					dirPath, err,
				)
			}

			if err := os.Symlink(entry.SymlinkPath, linkPath); err == nil {
				continue
			} else {
				util.ErrLog.Printf("failed to make a symbolic link(%s -> %s) :%v", linkPath, entry.SymlinkPath, err)
				continue
			}

		} else if mode.IsRegular() {
			if addSep {
				_ = w.WriteByte('\n')
			} else {
				addSep = true
			}
			_, _ = w.WriteString(entry.Path)
			_ = w.Flush()
		} else {
			util.ErrLog.Printf("skip filepath %s, unexpected filemode(%s)", entry.Path, entry.Mode)
		}
	}

}

func (t *Task) handleRsyncStdout(res *result, reader io.Reader, closer func(), fileList []returns.Fileinfo) {
	defer closer()

	prefix := fmt.Sprintf("[%d]&1> ", res.pid)
	scanner := bufio.NewScanner(reader)
	scanner.Split(util.ScanLineFeed)

	if len(fileList) == 0 {
		for scanner.Scan() {
			line := scanner.Text()
			util.InfoLog.Print(prefix, line)
		}
		return
	}

	filenameSet := make(map[string]int)
	for idx, info := range fileList {
		filenameSet[info.Path] = idx
	}

	bar := progressbar.NewOptions(res.total,
		progressbar.OptionSetWriter(util.LogWriter{}),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(time.Second),
		progressbar.OptionSetItsString("op"),
		progressbar.OptionSetDescription(fmt.Sprintf("[%d]", res.pid)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "-",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	defer bar.Close()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		matched := rsyncUptodateFormat.FindSubmatchIndex(line)
		groups := (len(matched) / 2) - 1
		if groups < 0 {
			log.Print(prefix, line)
			continue
		}
		bar.Add(1)

		match := func(i int) []byte {
			if len(matched) < (i+1)*2 {
				return nil
			} else if matched[i*2] < 0 || matched[i*2+1] < 0 {
				return nil
			} else {
				return line[matched[i*2]:matched[i*2+1]]
			}
		}
		path := string(match(1))
		if idx, contains := filenameSet[path]; !contains {
			res.processing++
			continue
		} else {
			res.appendFilename(path)
			if len(match(2)) == 0 {
				info := fileList[idx]
				res.sent++
				res.sentBytes += info.Size * 1024
			} else {
				res.uptodate++
			}
		}
	}
}

func (t *Task) handleRsyncStderr(res *result, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d]&2> ", res.pid)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.appendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}

func (t *Task) execute(parentContext context.Context, fileList []returns.Fileinfo) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invoke := exec.CommandContext(
		ctx,
		"rsync",
		t.Arguments...,
	)

	invoke.Env = os.Environ()
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if err := sys.ApplySysProc(invoke.SysProcAttr, false, false, false, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to set SysProcAttr: %w", err)
	}

	stdin, _ := invoke.StdinPipe()
	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(rsync): %w", err)
	}
	res := &result{total: len(fileList), started: time.Now(), pid: invoke.Process.Pid}

	go func() {
		select {
		case <-parentContext.Done():
			_ = invoke.Process.Signal(syscall.SIGTERM)
		case <-ctx.Done():
		}
	}()

	wg := &sync.WaitGroup{}
	wg.Add(3)
	go t.handleRsyncStdin(stdin, wg.Done, fileList)
	go t.handleRsyncStdout(res, stdout, wg.Done, fileList)
	go t.handleRsyncStderr(res, stderr, wg.Done)
	res.err = invoke.Wait()
	wg.Wait()
	util.InfoLog.Print(res)
	return res.HandleError()
}
