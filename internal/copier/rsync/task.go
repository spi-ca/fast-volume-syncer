package rsync

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
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
	Arguments       args.RsyncArgs
	FileMode        os.FileMode
	SourcePath      string
	DestinationPath string
	Retry           args.RetryArgs
	chunkIdx        uint64
}

func (t *Task) Execute(ctx context.Context, fileList []returns.Fileinfo) error {

	if t.Retry.Attempts <= 0 {
		return t.execute(ctx, fileList)
	}

	retryOptionArgs := t.Retry.Assemble(ctx)
	retryFunc := func() error { return t.execute(ctx, fileList) }
	return retry.Do(retryFunc, retryOptionArgs...)
}

func (t *Task) execute(parentContext context.Context, fileList []returns.Fileinfo) error {
	chunkIdx := atomic.AddUint64(&t.chunkIdx, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invoke := exec.CommandContext(
		ctx,
		"rsync",
		t.Arguments.Assemble(t.SourcePath, t.DestinationPath)...,
	)

	invoke.Env = os.Environ()
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if err := sys.ApplySysProAttrPdeathsig(invoke.SysProcAttr, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to set pdeathsig(%s): %w", syscall.SIGTERM, err)
	}

	stdin, _ := invoke.StdinPipe()
	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	// On Linux, pdeathsig will kill the child process when the thread dies,
	// not when the process dies. runtime.LockOSThread ensures that as long
	// as this function is executing that OS thread will still be around
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(rsync): %w", err)
	}
	res := &result{chunkIdx: chunkIdx, total: len(fileList), started: time.Now(), pid: invoke.Process.Pid}

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
			dirMode := mode.Perm() | t.FileMode.Perm()
			dirPath := filepath.Join(t.DestinationPath, entry.Path)
			err := t.processDirectory(dirPath, dirMode)
			if err != nil {
				util.ErrLog.Print(err)
				continue
			}
		} else if mode.Type()&fs.ModeSymlink != 0 {
			linkPath := filepath.Join(t.DestinationPath, entry.Path)
			err := t.processSymbolicLink(entry.SymlinkPath, linkPath)
			if err != nil {
				util.ErrLog.Print(err)
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

	prefix := fmt.Sprintf("[%d] ", res.pid)
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
		progressbar.OptionSetDescription(fmt.Sprintf("[chk:%d]\t", res.chunkIdx)),
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
			util.InfoLog.Print(prefix, line)
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
	prefix := fmt.Sprintf("[%d] ", res.pid)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.appendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}

func (t *Task) processDirectory(dstPath string, dstMode os.FileMode) error {
	destExists := false
	if dstPath == t.DestinationPath {
		// 자기자신은 무시하자
		return nil
	}

	existDstMode, err := os.Lstat(dstPath)

	if err == nil {
		if destExists = existDstMode.IsDir(); !destExists {
			// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
			err = os.RemoveAll(dstPath)
			if err != nil {
				return fmt.Errorf("failed to cleanup path %s :%w", dstPath, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to create directory %s(%s) :%w", dstPath, dstMode, err)
	}

	if destExists {
		// directory path 확보상태(이미 생성됨)
		if existDstMode.Mode() == dstMode {
			return nil
		}

		err = os.Chmod(dstPath, dstMode)
		if err != nil {
			return fmt.Errorf("failed to change directory(%s,%s): %w", dstPath, dstMode, err)

		}
	} else {
		// directory path 확보상태(비어있음)
		err = os.MkdirAll(dstPath, dstMode)
		if err != nil {
			return fmt.Errorf("failed to make a directory(%s,%s): %w", dstPath, dstMode, err)
		}
	}

	return nil
}

func (t *Task) processSymbolicLink(linkPath, dstPath string) error {

	if dstMode, err := os.Lstat(dstPath); err == nil {
		if dstMode.Mode().Type()&fs.ModeSymlink != 0 {
			existDstLinkPath, readLinkErr := os.Readlink(dstPath)
			if readLinkErr == nil && existDstLinkPath == linkPath {
				// 대상파일의 링크정보가 일치함
				return nil
			}
		}

		// 대상 path가 symlink mode가 아닌 경우 대상을 날린다.
		err = os.RemoveAll(dstPath)
		if err != nil {
			return fmt.Errorf("failed to cleanup path %s :%w", dstPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to create symlink %s, cannot get filestat :%w", dstPath, err)
	} else {
		// 자식이 없다는것은 부모도 없을 수 있다는 의미.  directory보다 링크가 먼저 온 case
		dirPath := filepath.Dir(dstPath)
		existDstDirMode, err := os.Lstat(dirPath)
		if err == nil {
			if !existDstDirMode.IsDir() {
				// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
				err = os.RemoveAll(dirPath)
				if err != nil {
					return fmt.Errorf("failed to make a symbolic link(%s -> %s), failed to cleanup path %s :%w",
						dstPath, linkPath, dirPath, err)
				}
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("failed to make a symbolic link(%s -> %s), failed to get fileinfo %s :%w",
				dstPath, linkPath, dirPath, err)
		} else if err = os.MkdirAll(dirPath, 0o755); err != nil {
			return fmt.Errorf("failed to make a symbolic link(%s -> %s), failed to create directory %s :%w",
				dstPath, linkPath,
				dirPath, err,
			)
		}
	}

	err := os.Symlink(linkPath, dstPath)
	if err != nil {
		err = fmt.Errorf("failed to make a symbolic link(%s -> %s) :%w", dstPath, linkPath, err)
	}
	return err
}
