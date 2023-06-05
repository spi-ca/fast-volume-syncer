package rsync

import (
	"bufio"
	"bytes"
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
	"syscall"
	"time"
	"unicode"

	"github.com/avast/retry-go"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

var (
	rsyncUptodateFormat = regexp.MustCompile(`^(.+?)( is uptodate)?`)
)

type Task struct {
	Arguments       []string
	DestinationPath string
	RetryAttempts   int
	RetryDelay      time.Duration
	RetryMaxDelay   time.Duration
	RetryMaxJitter  time.Duration
}

func (t *Task) Execute(ctx context.Context, fileList []common.Fileinfo) (err error) {
	log.Printf("argument is %s", t.Arguments)

	if t.RetryAttempts <= 0 {
		return t.execute(ctx, fileList)
	}

	optionArgs := []retry.Option{
		retry.Context(ctx),
		retry.Attempts(uint(t.RetryAttempts)),
	}

	if t.RetryDelay > 0 {
		optionArgs = append(optionArgs, retry.Delay(t.RetryDelay))
		if t.RetryMaxDelay > t.RetryDelay {
			optionArgs = append(optionArgs,
				retry.MaxJitter(t.RetryMaxJitter),
				retry.DelayType(retry.BackOffDelay),
			)
		} else {
			optionArgs = append(optionArgs, retry.DelayType(retry.FixedDelay))
		}
	}
	if t.RetryMaxJitter > 0 {
		optionArgs = append(optionArgs, retry.MaxJitter(t.RetryMaxJitter))
	}

	return retry.Do(
		func() error { return t.execute(ctx, fileList) },
		optionArgs...,
	)
}

func (t *Task) handleRsyncStdin(writer io.WriteCloser, closeChan chan<- struct{}, fileList []common.Fileinfo) {
	defer close(closeChan)
	if writer == nil {
		return
	}
	defer writer.Close()

	if len(fileList) == 0 {
		return
	}
	w := bufio.NewWriter(writer)
	defer func() {
		if err := w.Flush(); err != nil {
			log.Printf("failed to flush buffer :%v", err)
		}
	}()
	addSep := false
	for _, entry := range fileList {
		mode := entry.Mode
		if mode.IsDir() {
			// ensure mode
			dirMode := mode.Perm() | 0o700
			dirPath := filepath.Join(t.DestinationPath, entry.Path)
			log.Printf("make directory %s(%s)", dirPath, dirMode)
			if err := os.MkdirAll(dirPath, dirMode); err != nil {
				log.Printf("failed to create directory %s(%s) :%v", dirPath, dirMode, err)
			} else {
				log.Printf("directory %s(%s) created", dirPath, dirMode)
			}
		} else if mode.IsRegular() || (mode&fs.ModeSymlink != 0) {
			if addSep {
				if err := w.WriteByte('\n'); err != nil {
					log.Printf("failed to write buffer :%v", err)
				}
			} else {
				addSep = true
			}

			if _, err := w.WriteString(entry.Path); err != nil {
				log.Printf("failed to write buffer :%v", err)
			}
			if err := w.Flush(); err != nil {
				log.Printf("failed to flush buffer :%v", err)
			}
		}
	}

}

func (t *Task) handleRsyncStdout(pid int, reader io.Reader, totalFiles int, closeChan chan<- struct{}) {
	defer close(closeChan)

	prefix := fmt.Sprintf("[%d]&1> ", pid)
	scanner := bufio.NewScanner(reader)

	if totalFiles == 0 {
		for scanner.Scan() {
			line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
			log.Print(prefix, line)
		}
		return
	} else {
		result := result{}
		result.total = totalFiles
		defer func() {
			log.Print(result.String())
		}()

		for scanner.Scan() {
			line := bytes.TrimRightFunc(scanner.Bytes(), unicode.IsSpace)
			if len(line) == 0 {
				continue
			}

			matched := rsyncUptodateFormat.FindSubmatchIndex(line)
			groups := (len(matched) / 2) - 1
			if groups < 0 {
				log.Print(prefix, line)
				continue
			}

			match := func(i int) []byte {
				if len(matched) < (i+1)*2 {
					return nil
				}
				return line[matched[i*2]:matched[i*2+1]]
			}

			result.appendFilename(match(1))

			if groups < 2 {
				result.uptodate++
			} else {
				result.sent++
			}
		}
	}

}

func (t *Task) handleRsyncStderr(pid int, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&2> ", pid)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		log.Print(prefix, line)
	}
}

func (t *Task) execute(parentCtx context.Context, fileList []common.Fileinfo) (err error) {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	retryable := false
	defer func() {
		if err != nil && !retryable {
			err = retry.Unrecoverable(err)
		}
	}()
	invoke := exec.CommandContext(
		ctx,
		"rsync",
		t.Arguments...,
	)

	invoke.Env = append([]string(nil), os.Environ()...)
	stdin, _ := invoke.StdinPipe()
	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	err = invoke.Start()
	if err != nil {
		err = fmt.Errorf("failed to start process(rsync): %w", err)
		return
	}
	started := time.Now()
	pid := invoke.Process.Pid
	stdinClosed := make(chan struct{})
	go t.handleRsyncStdin(stdin, stdinClosed, fileList)

	stdoutClosed := make(chan struct{})
	go t.handleRsyncStdout(pid, stdout, len(fileList), stdoutClosed)

	stderrClosed := make(chan struct{})
	go t.handleRsyncStderr(pid, stderr, stderrClosed)

	log.Printf("rsync started(%d)", pid)

	<-stdinClosed

	select {
	case <-stdoutClosed:
		<-stderrClosed
	case <-stderrClosed:
		<-stdoutClosed
	}

	exitcode := 0
	err = invoke.Wait()
	ended := time.Now()
	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitcode = ws.ExitStatus()
		} else {
			exitcode = -1
		}
	}

	log.Printf("rsync(%d) exit code is %d in %2.2f ms", pid, exitcode, float32(ended.Sub(started).Microseconds())/1000)

	err, retryable = isExitedNormally(exitcode, err)
	return
}
