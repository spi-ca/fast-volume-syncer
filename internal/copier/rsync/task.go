// Package rsync copies file chunks by driving the rsync CLI.
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

// Task drives one rsync process for a chunk of scanned entries.
type Task struct {
	// Arguments holds the rsync CLI flags assembled for Execute.
	Arguments args.RsyncArgs
	// FileMode is applied when directories or symlinks are materialized locally.
	FileMode os.FileMode
	// SourcePath is the rsync source root passed to the process.
	SourcePath string
	// DestinationPath is the rsync destination root passed to the process.
	DestinationPath string
	// Retry configures chunk-level retries around execute.
	Retry args.RetryArgs
	// chunkIdx numbers chunks for logs and result messages.
	chunkIdx uint64
}

// Execute runs rsync for one chunk and optionally retries the whole chunk.
func (t *Task) Execute(ctx context.Context, fileList []returns.Fileinfo) (result returns.IOResult, err error) {

	chunkIdx := atomic.AddUint64(&t.chunkIdx, 1)
	if t.Retry.Attempts <= 0 {
		return t.execute(ctx, chunkIdx, fileList)
	}

	retryOptionArgs := t.Retry.Assemble(ctx)
	return result, retry.Do(func() (retryErr error) {
		result, retryErr = t.execute(ctx, chunkIdx, fileList)
		return
	}, retryOptionArgs...)
}

// execute starts rsync, streams chunk paths through stdin, and collects stdout/stderr accounting.
func (t *Task) execute(parentContext context.Context, chunkIdx uint64, fileList []returns.Fileinfo) (returns.IOResult, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res := &result{chunkIdx: chunkIdx, total: len(fileList), started: time.Now()}
	defer res.markEnd()
	if err := t.ensureDestinationPaths(fileList); err != nil {
		return res, err
	}

	rsyncPath := util.LookupBinary("rsync")
	if rsyncPath == "" {
		return res, fmt.Errorf("rsync binary not found in trusted PATH")
	}
	invoke := exec.CommandContext(
		ctx,
		rsyncPath,
		t.Arguments.Assemble(t.SourcePath, t.DestinationPath)...,
	)

	invoke.Env = util.TrustedChildEnvironment()
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if err := sys.ApplySysProAttrPdeathsig(invoke.SysProcAttr, syscall.SIGTERM); err != nil {
		return res, fmt.Errorf("failed to set pdeathsig(%s): %w", syscall.SIGTERM, err)
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
		return res, fmt.Errorf("failed to start process(rsync): %w", err)
	}

	res.pid = invoke.Process.Pid

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
	return res, res.HandleError()
}

// ensureDestinationPaths rejects destination symlink ancestors before rsync receives the file list.
func (t *Task) ensureDestinationPaths(fileList []returns.Fileinfo) error {
	for _, entry := range fileList {
		dstPath := filepath.Join(t.DestinationPath, entry.Path)
		if entry.Mode.IsDir() {
			if err := util.EnsureNoSymlinkPath(dstPath); err != nil {
				return err
			}
		} else if err := util.EnsureNoSymlinkAncestors(dstPath); err != nil {
			return err
		}
	}
	return nil
}

// handleRsyncStdin writes the chunk file list to rsync and materializes local dirs/symlinks.
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
		if t.Arguments.Port > 0 {
			if addSep {
				_ = w.WriteByte('\n')
			} else {
				addSep = true
			}
			_, _ = w.WriteString(entry.Path)
			_ = w.Flush()
		} else {
			mode := entry.Mode
			if mode.IsDir() {
				// ensure private destination directory mode
				dirMode := t.FileMode.Perm() | 0o700
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
				if err := util.EnsurePrivatePath(filepath.Dir(filepath.Join(t.DestinationPath, entry.Path))); err != nil {
					util.ErrLog.Print(err)
					continue
				}
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

}

// handleRsyncStdout parses rsync output lines into per-entry progress counters.
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
			res.processed++
			continue
		} else {
			info := fileList[idx]
			res.appendFilename(path)
			res.addTypeCount(info.Mode)
			if len(match(2)) == 0 {
				res.sent++
				res.sentBytes += info.Size
			} else {
				res.uptodate++
			}
		}
	}
}

// handleRsyncStderr keeps the last stderr lines for exit-code based error reporting.
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

// processDirectory ensures a destination directory exists before rsync writes files into it.
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

// processSymbolicLink recreates the destination symlink so rsync can skip link management.
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
		} else if err = os.MkdirAll(dirPath, t.FileMode.Perm()|0o700); err != nil {
			return fmt.Errorf("failed to make a symbolic link(%s -> %s), failed to create directory %s :%w",
				dstPath, linkPath,
				dirPath, err,
			)
		}
	}
	if err := util.EnsurePrivatePath(filepath.Dir(dstPath)); err != nil {
		return err
	}

	err := os.Symlink(linkPath, dstPath)
	if err != nil {
		err = fmt.Errorf("failed to make a symbolic link(%s -> %s) :%w", dstPath, linkPath, err)
	}
	return err
}
