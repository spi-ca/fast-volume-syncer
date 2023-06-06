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
	"time"
	"unicode"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
)

type Invoker struct {
	SandboxDisabled    bool
	SandboxMountOption string

	Args rsync.Args

	SourceMountHost    string
	SourceMountOptions string
	SourceMountName    string

	DestinationMountHost    string
	DestinationMountOptions string
	DestinationMountName    string

	ScanDuration     time.Duration
	FinderBinaryPath string

	TaskSize  int
	ChunkSize int

	RetryAttempts  int
	RetryDelay     time.Duration
	RetryMaxDelay  time.Duration
	RetryMaxJitter time.Duration
}

func (i *Invoker) Run(ctx context.Context, entry copyEntry) error {
	return i.execute(ctx, entry.SourceVolume, entry.SourcePath, entry.DestinationVolume, entry.DestinationPath)
}

func (i *Invoker) assembleEnvironment(inherited []string) []string {
	envs := make([]string, 0, 26)

	envs = append(envs, "SANDBOX_DISABLED", strconv.FormatBool(i.SandboxDisabled))
	envs = append(envs, "SANDBOX_MOUNT_OPTION", i.SandboxMountOption)

	envs = append(envs, "RSYNC_VERBOSE", strconv.FormatBool(i.Args.Verbose))
	envs = append(envs, "RSYNC_PERMS", strconv.FormatBool(i.Args.PreservePermission))
	envs = append(envs, "RSYNC_OWNER", strconv.FormatBool(i.Args.PreserveOwnership))
	envs = append(envs, "RSYNC_SPECIAL", strconv.FormatBool(i.Args.CopySpecial))
	envs = append(envs, "RSYNC_COMPRESS", strconv.FormatBool(i.Args.Compress))
	envs = append(envs, "RSYNC_WHOLE_FILE", strconv.FormatBool(i.Args.WholeFile))
	envs = append(envs, "RSYNC_INPLACE", strconv.FormatBool(i.Args.Inplace))
	envs = append(envs, "RSYNC_RECURSIVE", strconv.FormatBool(i.Args.Recursive))

	envs = append(envs, "SRC_STORAGE_MOUNT_HOST", i.SourceMountHost)
	envs = append(envs, "SRC_STORAGE_MOUNT_OPTION", i.SourceMountOptions)
	envs = append(envs, "SRC_STORAGE_MOUNT_NAME", i.SourceMountName)

	envs = append(envs, "DST_STORAGE_MOUNT_HOST", i.DestinationMountHost)
	envs = append(envs, "DST_STORAGE_MOUNT_OPTION", i.DestinationMountOptions)
	envs = append(envs, "DST_STORAGE_MOUNT_NAME", i.DestinationMountName)

	envs = append(envs, "SCAN_DEADLINE", i.ScanDuration.String())
	envs = append(envs, "SCAN_FIND_PATH", i.FinderBinaryPath)

	envs = append(envs, "TASK_SIZE", strconv.Itoa(i.TaskSize))
	envs = append(envs, "CHUNK_SIZE", strconv.Itoa(i.ChunkSize))

	envs = append(envs, "RETRY_ATTEMPTS", strconv.Itoa(i.RetryAttempts))
	envs = append(envs, "RETRY_DELAY", i.RetryDelay.String())
	envs = append(envs, "RETRY_MAX_DELAY", i.RetryMaxDelay.String())
	envs = append(envs, "RETRY_MAX_JITTER", i.RetryMaxJitter.String())

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
	path, procAttr := common.Self(!i.SandboxDisabled)
	invoke := exec.CommandContext(
		ctx,
		path,
		"sync", srcPath, srcSubpath, dstPath, dstSubpath,
	)

	invoke.Env = i.assembleEnvironment(os.Environ())
	//for _, row := range invoke.Env {
	//	log.Print("environ :", row)
	//}
	invoke.SysProcAttr = procAttr
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
