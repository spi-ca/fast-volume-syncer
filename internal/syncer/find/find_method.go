package find

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
	"strconv"
	"strings"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/internal"
)

var (

	/*
	   e.g.

	   	51791395877894146  598 -rw-r--r--   1 root     root       612192 Feb 15 20:54 /fixture/root/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/rein20/dataset-host/image-prompt-fixture/input/image-set-2m/00000000-0000-4000-8000-000000000000.png
	   	35465847138781934 3571 -rw-r--r--   1 root     root      3655851 Nov 11  2021 /tmp/fixture/src/sandbox/fixture-data/units/unit-a/sandbox/session-a/org/deployments/model_fixture_low_loss/model_recommendation/output/model_fixture_long_train/account_map.json
	   	6192449664352658    0 drwxr-xr-x   2 root     root            0 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9
	   	7881299523698987    1 lrwxrwxrwx   1 root     root           52 May 23 20:04 ./c493ea33b1f86f09f0dd621d4e8c1d9d4a8453b9/vocab.json -> ../../blobs/0c9fccca89c9a8d2554dc00cc621c044aae04adb
	   	38035803        8 -rw-r--r--    1 example.user          staff                 362 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-os.cpython-311.pyc
	   	38035731        8 -rw-r--r--    1 example.user          staff                 562 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.Adw.cpython-311.pyc
	   	38035759        8 -rw-r--r--    1 example.user          staff                 570 May 29 19:50 ./venv_darwin/lib/python3.11/site-packages/Packager/hooks/__pycache__/hook-gui.repository.GstNet.cpython-311.pyc
	*/
	findFormat    = regexp.MustCompile(`^(\d+?)\s+(\d+?)\s+([^\s]+?)\s+(\d+?)\s+(.+?)\s+(.+?)\s+(\d+?)\s+([A-Za-z]+?\s+\d+?\s+\d+?(?::\d+?)?)\s+(.*)$`)
	symlinkFormat = regexp.MustCompile(`^(.*) -> (.*)$`)
)

func (s *Scanner) executeFind(ctx context.Context, root string, rowChan chan<- common.Fileinfo) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic on Scanner.executeFind : %v", err)
		}
	}()

	invoke := exec.CommandContext(
		ctx,
		s.FindBinaryPath,
		root,
		"-ls",
	)

	invoke.Env = append([]string(nil), os.Environ()...)
	invoke.Stdin = nil
	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		log.Printf("failed to start process(find): %v", err)
		return
	}
	started := time.Now()
	pid := invoke.Process.Pid
	stdoutClosed := make(chan struct{})
	go s.handleFindStdout(pid, stdout, stdoutClosed, rowChan, root)

	stderrClosed := make(chan struct{})
	go s.handleFindStderr(pid, stderr, stderrClosed)

	log.Printf("find started(%d)", pid)

	select {
	case <-stdoutClosed:
		<-stderrClosed
	case <-stderrClosed:
		<-stdoutClosed
	}

	err := invoke.Wait()
	ended := time.Now()
	if err != nil {
		log.Printf("find(%d) ended in %2.2f ms, %v", pid, float32(ended.Sub(started).Microseconds())/1000, err)
	} else {
		log.Printf("find(%d) ended in %2.2f ms", pid, float32(ended.Sub(started).Microseconds())/1000)
	}
}

func (s *Scanner) parseFindEntry(line []byte) (*common.Fileinfo, error) {

	matched := findFormat.FindSubmatchIndex(line)
	if groups := len(matched) / 2; groups < 1 {
		return nil, fmt.Errorf("scan: invalid find result %s", line)
	}

	match := func(i int) []byte {
		if len(matched) < (i+1)*2 {
			return nil
		}
		return line[matched[i*2]:matched[i*2+1]]
	}

	//inode, _ := strconv.Atoi(match(1))
	size, _ := strconv.ParseInt(string(match(2)), 10, 0)
	mode := internal.UnFilemode(match(3))
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
			}
			return path[symlinkedMatched[i*2]:symlinkedMatched[i*2+1]]
		}
		src := symlinkPathMatch(1)
		//dst := symlinkPathMatch(2)
		path = src
	}

	return &common.Fileinfo{
		Path: string(path),
		Mode: mode,
		Size: size,
	}, nil
}

func (s *Scanner) handleFindStderr(pid int, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&2> ", pid)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		log.Print(prefix, line)
	}
}

func (s *Scanner) handleFindStdout(pid int, reader io.Reader, closeChan chan<- struct{}, rowChan chan<- common.Fileinfo, root string) {
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
			log.Printf("[%d]failed to parse find line: %s, %v", pid, line, err)
			continue
		}

		relPath, err := filepath.Rel(root, entry.Path)
		if err != nil {
			log.Printf("[%d]failed to make relative file path info: %v", pid, err)
			continue
		}
		entry.Path = relPath

		rowChan <- *entry
	}
}
