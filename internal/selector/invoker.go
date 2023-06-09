package selector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Invoker struct {
	SandboxDisabled bool

	Common args.SyncerCommonArguments
}

func (i *Invoker) Run(ctx context.Context, entry copyEntry) error {
	return i.execute(ctx, entry.SourceVolume, entry.SourcePath, entry.DestinationVolume, entry.DestinationPath)
}

func (i *Invoker) assembleEnvironment(inherited []string) []string {
	inherited = i.Common.AssembleEnvironment(inherited)
	envs := make([]string, 0, 2)
	envs = append(envs, "_SYNCER_INVOKED", strconv.FormatBool(true))
	envs = append(envs, "_SYNCER_SANDBOXED", strconv.FormatBool(!i.SandboxDisabled))

	b := strings.Builder{}
	for i := 0; i < len(envs)/2; i++ {
		b.WriteString(envs[i*2])
		b.WriteByte('=')
		b.WriteString(envs[i*2+1])
		inherited = append(inherited, b.String())
		b.Reset()
	}
	return inherited
}

func (i *Invoker) handleStdout(res *returns.ExecutionResult, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&1> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		util.InfoLog.Print(prefix, line)
	}
}

func (i *Invoker) handleStderr(res *returns.ExecutionResult, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&2> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}

func (i *Invoker) execute(ctx context.Context, srcPath, srcSubpath, dstPath, dstSubpath string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get self-path: %w", err)
	}

	invoke := exec.CommandContext(ctx, self, "sync", srcPath, srcSubpath, dstPath, dstSubpath)

	invoke.Env = i.assembleEnvironment(os.Environ())
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if !i.SandboxDisabled {
		if err := sys.IsolateMountNamespaceFlags(invoke.SysProcAttr); err != nil {
			return fmt.Errorf("failed to sanxbox a process: %w", err)
		}
	}

	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(rsync): %w", err)
	}
	started := time.Now()
	res := &returns.ExecutionResult{PID: invoke.Process.Pid}

	stdoutClosed := make(chan struct{})
	go i.handleStdout(res, stdout, stdoutClosed)

	stderrClosed := make(chan struct{})
	go i.handleStderr(res, stderr, stderrClosed)

	select {
	case <-stdoutClosed:
		<-stderrClosed
	case <-stderrClosed:
		<-stdoutClosed
	}

	res.Err = invoke.Wait()
	ended := time.Now()
	if err := res.HandleError(); err != nil {
		return fmt.Errorf("selector(%d): %w", res.PID, err)
	} else {
		util.InfoLog.Printf("selector(%d) ended in %2.2f ms", res.PID, float32(ended.Sub(started).Microseconds())/1000)
		return nil
	}
}
