package selector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

type Invoker struct {
	SandboxDisabled bool

	Common common.Template
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

func (i *Invoker) execute(ctx context.Context, srcPath, srcSubpath, dstPath, dstSubpath string) error {
	invoke := exec.CommandContext(ctx, common.Executables(), "sync", srcPath, srcSubpath, dstPath, dstSubpath)

	invoke.Env = i.assembleEnvironment(os.Environ())
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	if !i.SandboxDisabled {
		if err := common.ApplySandboxFlags(invoke.SysProcAttr); err != nil {
			return fmt.Errorf("failed to sanxbox a process: %w", err)
		}
	}

	stdout, _ := invoke.StdoutPipe()
	stderr, _ := invoke.StderrPipe()

	if err := invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(rsync): %w", err)
	}
	started := time.Now()
	pid := invoke.Process.Pid

	stdoutClosed := make(chan struct{})
	go i.handleStdout(pid, stdout, stdoutClosed)

	stderrClosed := make(chan struct{})
	go i.handleStderr(pid, stderr, stderrClosed)

	select {
	case <-stdoutClosed:
		<-stderrClosed
	case <-stderrClosed:
		<-stdoutClosed
	}

	err := invoke.Wait()
	ended := time.Now()
	if err != nil {
		return fmt.Errorf("selector(%d): %w", pid, err)
	} else {
		log.Printf("selector(%d) ended in %2.2f ms", pid, float32(ended.Sub(started).Microseconds())/1000)
		return nil
	}
}

func (i *Invoker) handleStdout(pid int, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&2> ", pid)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		log.Print(prefix, line)
	}
}

func (i *Invoker) handleStderr(pid int, reader io.Reader, closeChan chan<- struct{}) {
	defer close(closeChan)
	prefix := fmt.Sprintf("[%d]&2> ", pid)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimRightFunc(scanner.Text(), unicode.IsSpace)
		log.Print(prefix, line)
	}
}
