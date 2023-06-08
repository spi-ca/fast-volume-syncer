package find

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/model"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

var (
	findFormat    = regexp.MustCompile(`^(\d+?)\s+(\d+?)\s+([^\s]+?)\s+(\d+?)\s+(.+?)\s+(.+?)\s+(\d+?)\s+([A-Za-z]+?\s+\d+?\s+\d+?(?::\d+?)?)\s+(.*)$`)
	symlinkFormat = regexp.MustCompile(`^(.*) -> (.*)$`)
)

func (s *Scanner) parseFindEntry(line []byte) (*model.Fileinfo, error) {

	matched := findFormat.FindSubmatchIndex(line)
	if groups := len(matched) / 2; groups < 1 {
		return nil, fmt.Errorf("scan: invalid find result %s", line)
	}

	match := func(i int) []byte {
		if len(matched) < (i+1)*2 {
			return nil
		} else if matched[i*2] < 0 || matched[i*2+1] < 0 {
			return nil
		} else {
			return line[matched[i*2]:matched[i*2+1]]
		}
	}

	//inode, _ := strconv.Atoi(match(1))
	size := util.SimpleStrconv(match(2))
	mode := util.UnFilemode(match(3))
	//num_of_hardlink, _ := strconv.Atoi(match(4))
	//owner := match(5)
	//group := match(6)
	//store_size, _ := strconv.Atoi(match(7))
	//date := match(8)
	path := match(9)

	if mode&fs.ModeSymlink != 0 {
		symlinkedMatched := symlinkFormat.FindSubmatchIndex(path)
		if groups := len(symlinkedMatched) / 2; groups < 1 {
			return nil, fmt.Errorf("scan: invalid symlink path %s", path)
		}

		symlinkPathMatch := func(i int) []byte {
			if len(matched) < (i+1)*2 {
				return nil
			} else if matched[i*2] < 0 || matched[i*2+1] < 0 {
				return nil
			} else {
				return path[symlinkedMatched[i*2]:symlinkedMatched[i*2+1]]
			}
		}
		src := symlinkPathMatch(1)
		//dst := symlinkPathMatch(2)
		path = src
	}

	return &model.Fileinfo{
		Path: string(path),
		Mode: mode,
		Size: size,
	}, nil
}

func (s *Scanner) handleFindStderr(res *model.ExecutionResult, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&2> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}

func (s *Scanner) handleFindStdout(res *model.ExecutionResult, reader io.Reader, closeChan chan<- struct{}, rowChan chan<- model.Fileinfo, root string) {
	defer func() {
		close(rowChan)
		close(closeChan)
	}()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := bytes.TrimRightFunc(scanner.Bytes(), unicode.IsSpace)
		if len(line) == 0 {
			continue
		}
		entry, err := s.parseFindEntry(line)
		if err != nil {
			util.ErrLog.Printf("[%d]failed to parse find line: %s, %v", res.PID, line, err)
			continue
		}

		relPath, err := filepath.Rel(root, entry.Path)
		if err != nil {
			util.ErrLog.Printf("[%d]failed to make relative file path info: %v", res.PID, err)
			continue
		}
		entry.Path = relPath

		rowChan <- *entry
	}
}

func (s *Scanner) executeFind(ctx context.Context, root string, rowChan chan<- model.Fileinfo) {
	defer func() {
		if err := recover(); err != nil {
			util.SendSlackMessage(fmt.Sprintf("panic on Scanner.executeFind : %v", err))
		}
	}()

	invoke := exec.CommandContext(
		ctx,
		s.FinderBinaryPath,
		root,
		"-ls",
	)

	invoke.Env = os.Environ()
	invoke.Stdin = nil
	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		util.ErrLog.Printf("failed to start process(find): %v", err)
		return
	}
	started := time.Now()
	res := &model.ExecutionResult{PID: invoke.Process.Pid}

	stdoutClosed := make(chan struct{})
	go s.handleFindStdout(res, stdout, stdoutClosed, rowChan, root)

	stderrClosed := make(chan struct{})
	go s.handleFindStderr(res, stderr, stderrClosed)

	util.InfoLog.Printf("find started(%d)", res.PID)

	select {
	case <-stdoutClosed:
		<-stderrClosed
	case <-stderrClosed:
		<-stdoutClosed
	}

	res.Err = invoke.Wait()
	ended := time.Now()

	if err := res.HandleError(); err != nil {
		util.ErrLog.Printf("find(%d) ended in %2.2f ms, %v", res.PID, float32(ended.Sub(started).Microseconds())/1000, err)
	} else {
		util.InfoLog.Printf("find(%d) ended in %2.2f ms", &res, float32(ended.Sub(started).Microseconds())/1000)
	}
}
