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
	"time"
	"unicode"

	"github.com/schollz/progressbar/v3"

	"github.com/avast/retry-go"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

var (
	rsyncUptodateFormat = regexp.MustCompile(`^(.+?)( is uptodate)?$`)
)

type Task struct {
	Arguments       []string
	DestinationPath string
	RetryAttempts   int
	RetryDelay      time.Duration
	RetryMaxDelay   time.Duration
	RetryMaxJitter  time.Duration
}

func (t *Task) Execute(ctx context.Context, fileList []common.Fileinfo) error {

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
	addSep := false
	for _, entry := range fileList {
		mode := entry.Mode
		if mode.IsDir() {
			// ensure mode
			dirMode := mode.Perm() | 0o700
			dirPath := filepath.Join(t.DestinationPath, entry.Path)
			if err := os.MkdirAll(dirPath, dirMode); err != nil {
				log.Printf("failed to create directory %s(%s) :%v", dirPath, dirMode, err)
			}
		} else if mode.IsRegular() || (mode&fs.ModeSymlink != 0) {
			if addSep {
				_ = w.WriteByte('\n')
			} else {
				addSep = true
			}

			_, _ = w.WriteString(entry.Path)
			_ = w.Flush()
		}
	}

}

func (t *Task) handleRsyncStdout(res *result, reader io.Reader, fileList []common.Fileinfo, closeChan chan struct{}) {
	defer close(closeChan)

	prefix := fmt.Sprintf("[%d]&1> ", res.pid)
	scanner := bufio.NewScanner(reader)

	if len(fileList) == 0 {
		for scanner.Scan() {
			line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
			log.Print(prefix, line)
		}
		return
	}

	filenameSet := make(map[string]int)
	for idx, info := range fileList {
		filenameSet[info.Path] = idx
	}

	bar := progressbar.NewOptions(res.total,
		progressbar.OptionSetWriter(common.LogWriter{}),
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
		path := string(bytes.TrimSpace(match(1)))
		if idx, contains := filenameSet[path]; !contains {
			res.processing++
			continue
		} else {
			res.appendFilename(path)
			if len(match(2)) == 0 {
				info := fileList[idx]
				res.sent++
				res.sentBytes += info.Size
			} else {
				res.uptodate++
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

func (t *Task) execute(ctx context.Context, fileList []common.Fileinfo) error {
	invoke := exec.CommandContext(
		ctx,
		"rsync",
		t.Arguments...,
	)

	invoke.Env = append([]string(nil), os.Environ()...)
	stdin, _ := invoke.StdinPipe()
	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(rsync): %w", err)
	}
	res := &result{total: len(fileList), started: time.Now(), pid: invoke.Process.Pid}

	stdinClosed := make(chan struct{})
	go t.handleRsyncStdin(stdin, stdinClosed, fileList)

	stdoutClosed := make(chan struct{})
	go t.handleRsyncStdout(res, stdout, fileList, stdoutClosed)

	stderrClosed := make(chan struct{})
	go t.handleRsyncStderr(res.pid, stderr, stderrClosed)

	<-stdinClosed

	select {
	case <-stdoutClosed:
		<-stderrClosed
	case <-stderrClosed:
		<-stdoutClosed
	}

	res.err = invoke.Wait()
	log.Print(res)
	return res.HandleError()
}
