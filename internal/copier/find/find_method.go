// Package find scans source trees with either `find -ls` or an in-process walker.
package find

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

var (
	findFormat    = regexp.MustCompile(`^(\d+?)\s+(\d+?)\s+(\S+?)\s+(\d+?)\s+(.+?)\s+(.+?)\s+(\d+?)\s+([A-Za-z]+?\s+\d+?\s+\d+?(?::\d+?)?)\s+(.*)$`)
	symlinkFormat = regexp.MustCompile(`^(.*) -> (.*)$`)
)

// parseFindEntry converts one `find -ls` output line into Fileinfo data.
func (s *Scanner) parseFindEntry(line []byte) (*returns.Fileinfo, error) {

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
	//blocks, _ := strconv.Atoi(match(2))
	mode := sys.UnFilemode(match(3))
	//num_of_hardlink, _ := strconv.Atoi(match(4))
	//owner := match(5)
	//group := match(6)
	size := util.SimpleStrconv(match(7))
	//date := match(8)
	path := match(9)

	if mode.Type()&fs.ModeSymlink != 0 {
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
		dst := symlinkPathMatch(2)
		return &returns.Fileinfo{
			Path:        string(src),
			SymlinkPath: string(dst),
			Mode:        mode,
			Size:        size,
		}, nil
	} else {
		return &returns.Fileinfo{
			Path: string(path),
			Mode: mode,
			Size: size,
		}, nil
	}
}

// handleFindStderr logs stderr lines so scanner failures keep their original context.
func (s *Scanner) handleFindStderr(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d]&2> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}

// handleFindStdout parses `find -ls` rows, filters ignored paths, and emits relative entries.
func (s *Scanner) handleFindStdout(ctx context.Context, res *returns.ExecutionResult, reader io.Reader, closer func(), rowChan chan<- returns.Fileinfo, root string) {
	defer closer()
	scanner := bufio.NewScanner(reader)
	scanner.Split(util.ScanLineFeed)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		entry, err := s.parseFindEntry(line)
		if err != nil {
			util.ErrLog.Printf("[%d]failed to parse find line: %s, %v", res.PID, string(line), err)
			continue
		}

		relPath, err := filepath.Rel(root, entry.Path)
		if err != nil {
			util.ErrLog.Printf("[%d]failed to make relative file path info: %v", res.PID, err)
			continue
		} else if s.ignore(relPath, entry.Mode) {
			continue
		}

		entry.Path = relPath

		select {
		case <-ctx.Done():
			return
		case rowChan <- *entry:
		}
	}
}

// executeFind runs the external find command and streams parsed results back to Scan.
func (s *Scanner) executeFind(ctx context.Context, root string, rowChan chan<- returns.Fileinfo) error {
	invoke := exec.CommandContext(
		ctx,
		s.FinderBinaryPath,
		root,
		"-ls",
	)

	invoke.Env = util.TrustedChildEnvironment()
	invoke.Stdin = nil
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if err := sys.ApplySysProAttrPdeathsig(invoke.SysProcAttr, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to set pdeathsig(%s): %w", syscall.SIGTERM, err)
	}

	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	// On Linux, pdeathsig will kill the child process when the thread dies,
	// not when the process dies. runtime.LockOSThread ensures that as long
	// as this function is executing that OS thread will still be around
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(find): %w", err)
	}
	started := time.Now()

	res := &returns.ExecutionResult{PID: invoke.Process.Pid}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go s.handleFindStdout(ctx, res, stdout, wg.Done, rowChan, root)
	go s.handleFindStderr(res, stderr, wg.Done)
	util.InfoLog.Printf("find started(%d)", res.PID)
	res.Err = invoke.Wait()
	ended := time.Now()
	wg.Wait()
	util.InfoLog.Printf("find(%d) ended in %s", &res, ended.Sub(started))
	return res.HandleError()
}
