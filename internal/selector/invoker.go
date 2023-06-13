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
	"sync"
	"syscall"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Invoker struct {
	SandboxDisabled bool
	Common          args.SyncerCommonArguments
}

func (i *Invoker) Run(parentContext context.Context, entry copyEntry) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invoke := exec.CommandContext(ctx, sys.Executable(), "sync", entry.SourceVolume, entry.SourcePath, entry.DestinationVolume, entry.DestinationPath)
	invoke.Env = i.assembleEnvironment(os.Environ())
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if err := sys.ApplySysProc(invoke.SysProcAttr, !i.SandboxDisabled, true, false, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to set SysProcAttr: %w", err)
	}

	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(rsync): %w", err)
	}

	res := &returns.ExecutionResult{PID: invoke.Process.Pid}

	go func() {
		select {
		case <-parentContext.Done():
			_ = invoke.Process.Signal(syscall.SIGTERM)
		case <-ctx.Done():
		}
	}()

	wg := &sync.WaitGroup{}
	wg.Add(2)
	go i.handleStdout(res, stdout, wg.Done)
	go i.handleStderr(res, stderr, wg.Done)
	res.Err = invoke.Wait()
	wg.Wait()

	return res.HandleError()
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

func (i *Invoker) handleStdout(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d]&1> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		util.InfoLog.Print(prefix, line)
	}
}

func (i *Invoker) handleStderr(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d]&2> ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}
