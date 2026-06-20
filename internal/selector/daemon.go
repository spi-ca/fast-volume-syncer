// Package selector parses copy-entry CSV rows and fans them out to sync workers.
package selector

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// Daemonizer launches a detached select process with inherited sync configuration.
type Daemonizer struct {
	// NodeSelector limits the detached selector to one node when non-negative.
	NodeSelector int
	// CopyInfoCSVPath names the CSV file the detached selector should read.
	CopyInfoCSVPath string
	// LogFilePath is the append-only log sink for the detached selector process.
	LogFilePath string
	// PidFilePath is exported so the detached selector can lock and publish its pid.
	PidFilePath string
	// WorkerSize caps how many sync children the detached selector may run at once.
	WorkerSize int
	// SandboxDisabled tells the detached selector to skip process isolation.
	SandboxDisabled bool

	// Common carries the syncer/copier environment shared with the detached selector.
	Common args.SyncerCommonArguments
}

// assembleEnvironment appends daemon-specific environment variables to the inherited syncer environment.
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

// openLogFile creates the daemon log directory and opens the append-only log file.
func (i *Daemonizer) openLogFile() (*os.File, error) {
	logFileDir := filepath.Dir(i.LogFilePath)
	if err := ensureDaemonDirectory(logFileDir); err != nil {
		return nil, fmt.Errorf("unsafe log directory: %w", err)
	}
	if err := validatePidDirectory(logFileDir); err != nil {
		return nil, fmt.Errorf("unsafe log directory: %w", err)
	}

	logFile, err := os.OpenFile(i.LogFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	if info, err := logFile.Stat(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("failed to stat log file: %w", err)
	} else if !info.Mode().IsRegular() {
		_ = logFile.Close()
		return nil, fmt.Errorf("log file must be a regular file: %s", i.LogFilePath)
	} else if err := validateOwnedPrivatePath(i.LogFilePath, info); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	return logFile, nil
}

// Execute starts a detached select child, redirects logs, and releases the parent handle.
func (i *Daemonizer) Execute() error {

	logFile, err := i.openLogFile()
	if err != nil {
		return err
	}
	defer logFile.Close()

	var invoke *exec.Cmd

	if i.NodeSelector < 0 {
		invoke = exec.Command(sys.Executable(), "select", "_", i.CopyInfoCSVPath)
	} else {
		invoke = exec.Command(sys.Executable(), "select", strconv.Itoa(i.NodeSelector), i.CopyInfoCSVPath)

	}
	invoke.Stdin = nil
	invoke.Stdout = logFile
	invoke.Stderr = logFile
	invoke.Env = i.assembleEnvironment(util.TrustedChildEnvironment())
	invoke.SysProcAttr = &syscall.SysProcAttr{}

	err = sys.ApplySysProAttrSid(invoke.SysProcAttr)
	if err != nil {
		return fmt.Errorf("failed to set session id: %w", err)
	}

	err = invoke.Start()
	if err != nil {
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
