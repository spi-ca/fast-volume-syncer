// Package selector parses copy-entry CSV rows and fans them out to sync workers.
package selector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
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

// Invoker launches one sync child process for a selected copy entry.
type Invoker struct {
	// SandboxDisabled decides whether the sync child should skip namespace isolation.
	SandboxDisabled bool
	// Common carries the shared syncer/copier environment for every child process.
	Common args.SyncerCommonArguments
}

// Run starts the sync child, forwards its logs, and ties its lifetime to the selector context.
func (i *Invoker) Run(parentContext context.Context, entry copyEntry) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	invoke := exec.CommandContext(ctx, sys.Executable(), "sync", entry.SourceVolume, entry.SourcePath, entry.DestinationVolume, entry.DestinationPath)
	invoke.Env = i.assembleEnvironment(util.TrustedChildEnvironment())
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if i.SandboxDisabled {
		// The sync child will mount directly in the current namespace.
	} else if err := sys.ApplySysProAttrIsolation(invoke.SysProcAttr); err != nil {
		return fmt.Errorf("failed to set unshare flags id: %w", err)
	}

	if err := sys.ApplySysProAttrPGid(invoke.SysProcAttr); err != nil {
		return fmt.Errorf("failed to set process group id: %w", err)
	}

	if err := sys.ApplySysProAttrPdeathsig(invoke.SysProcAttr, syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to set pdeathsig(%s): %w", syscall.SIGTERM, err)
	}

	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	// pdeathsig is tied to the calling OS thread, so keep this goroutine pinned until Start returns.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

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

// assembleEnvironment marks the child as selector-invoked and records whether it should sandbox itself.
func (i *Invoker) assembleEnvironment(inherited []string) []string {
	inherited = i.Common.AssembleEnvironment(inherited)
	envs := make([]string, 0, 2*2)
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

// handleStdout prefixes child stdout lines with the child pid and writes them to the info log.
func (i *Invoker) handleStdout(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d] ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		util.InfoLog.Print(prefix, line)
	}
}

// handleStderr prefixes child stderr lines, records them on the result, and writes them to the error log.
func (i *Invoker) handleStderr(res *returns.ExecutionResult, reader io.Reader, closer func()) {
	defer closer()
	prefix := fmt.Sprintf("[%d] ", res.PID)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		res.AppendLogLine(line)
		util.ErrLog.Print(prefix, line)
	}
}
