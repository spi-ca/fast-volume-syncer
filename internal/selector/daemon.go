package selector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Daemonizer struct {
	NodeSelector    int
	CopyInfoCSVPath string
	LogFilePath     string
	PidFilePath     string
	WorkerSize      int
	SandboxDisabled bool

	Common args.SyncerCommonArguments
}

func (i *Daemonizer) assembleEnvironment(inherited []string) []string {
	inherited = i.Common.AssembleEnvironment(inherited)
	envs := make([]string, 0, 1)
	envs = append(envs, "_FVS_DAEMONEZED", strconv.FormatBool(true))
	envs = append(envs, "_PID_FILEPATH", i.PidFilePath)
	envs = append(envs, "WORKER_SIZE", strconv.Itoa(i.WorkerSize))
	envs = append(envs, "SANDBOX_DISABLED", strconv.FormatBool(i.SandboxDisabled))
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
func (i *Daemonizer) openFiles() (*os.File, *os.File, error) {

	nullFile, err := os.Open(os.DevNull)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open null file: %w", err)
	}
	logFileDir := filepath.Dir(i.LogFilePath)
	err = os.MkdirAll(logFileDir, 0o755)
	if err != nil {
		_ = nullFile.Close()
		return nil, nil, fmt.Errorf("failed to make logdir(%s): %w", logFileDir, err)
	}
	logFile, err := os.OpenFile(i.LogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		_ = nullFile.Close()
		return nil, nil, fmt.Errorf("failed to open log file: %w", err)
	}
	return logFile, nullFile, nil
}

func (i *Daemonizer) Execute() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	logFile, nullFile, err := i.openFiles()
	if err != nil {
		return err
	}

	defer func() {
		_ = nullFile.Close()
		_ = logFile.Close()
	}()

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get self-path: %w", err)
	}

	invoke := exec.Command("nohup", self, "select", strconv.Itoa(i.NodeSelector), i.CopyInfoCSVPath)
	invoke.Stdin = nil
	invoke.Stdout = logFile
	invoke.Stderr = logFile
	invoke.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
	invoke.Env = i.assembleEnvironment(os.Environ())

	if err = invoke.Start(); err != nil {
		return fmt.Errorf("failed to start process(selector): %w", err)
	}

	pid := invoke.Process.Pid
	util.InfoLog.Printf("daemon process(%d) invoked! ", pid)

	err = invoke.Process.Release()
	if err != nil {
		return fmt.Errorf("daemon process(selector) release failed: %w", err)
	}

	return nil
}
