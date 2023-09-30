package internal

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/ch-boot/internal/returns"
	"amuz.es/src/spi-ca/ch-boot/internal/sys"
	"amuz.es/src/spi-ca/ch-boot/internal/util"
)

type VirtiofsdMonitor struct {
	BinaryPath       string
	EntryChannelSize int
}

func (s *VirtiofsdMonitor) Execute(ctx context.Context, root string) (<-chan returns.Fileinfo, <-chan error) {
	entryChan := make(chan returns.Fileinfo, s.EntryChannelSize)
	errorChan := make(chan error, 1)
	go s.execute(ctx, root, errorChan)
	return entryChan, errorChan
}

func (s *VirtiofsdMonitor) handleStdout(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d]&1> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		log.Print(prefix, line)
	}
}
func (s *VirtiofsdMonitor) handleStderr(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d]&2> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		log.Print(prefix, line)
	}
}

func (s *VirtiofsdMonitor) execute(ctx context.Context, cwd string, errorChan chan<- error) {
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on Scanner.Scan : %v", err)
		}
		close(errorChan)
	}()

	invoke := exec.CommandContext(
		ctx,
		s.BinaryPath,
		cwd,
		"-ls",
	)

	invoke.Env = os.Environ()
	invoke.Stdin = nil
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if err := sys.ApplySysProAttrPdeathsig(invoke.SysProcAttr, syscall.SIGTERM); err != nil {
		errorChan <- fmt.Errorf("failed to set pdeathsig(%s): %w", syscall.SIGTERM, err)
	}

	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	// On Linux, pdeathsig will kill the child process when the thread dies,
	// not when the process dies. runtime.LockOSThread ensures that as long
	// as this function is executing that OS thread will still be around
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := invoke.Start(); err != nil {
		errorChan <- fmt.Errorf("failed to start process(find): %w", err)
	}
	started := time.Now()

	res := &returns.ExecutionResult{PID: invoke.Process.Pid}
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go s.handleStdout(res, stdout, wg.Done)
	go s.handleStderr(res, stderr, wg.Done)

	log.Printf("virtiofsd started(%d)", res.PID)
	res.Err = invoke.Wait()
	ended := time.Now()
	wg.Wait()
	log.Printf("virtiofsd(%d) ended in %s", &res, ended.Sub(started))

	if err := res.HandleError(); err != nil {
		errorChan <- fmt.Errorf("failed to start process(find): %w", err)
	}
}
