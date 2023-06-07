package selector

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

type Daemonizer struct {
	NodeSelector    int
	CopyInfoCSVPath string
	LogFilePath     string
	WorkerSize      int
	SandboxDisabled bool

	Common common.Template
}

func (i *Daemonizer) assembleEnvironment(inherited []string) []string {
	inherited = i.Common.AssembleEnvironment(inherited)
	envs := make([]string, 0, 1)
	envs = append(envs, "_FVS_DAEMONEZED", strconv.FormatBool(true))
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
	logFile, nullFile, err := i.openFiles()
	if err != nil {
		return err
	}

	defer func() {
		_ = nullFile.Close()
		_ = logFile.Close()
	}()

	attr := os.ProcAttr{
		Env: i.assembleEnvironment(os.Environ()),
		Files: []*os.File{
			nullFile, // (0) stdin
			logFile,  // (1) stdout
			logFile,  // (2) stderr
		},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}
	exe := common.Executables()
	argv := []string{exe, "select", strconv.Itoa(i.NodeSelector), i.CopyInfoCSVPath}

	child, err := os.StartProcess(exe, argv, &attr)
	if err != nil {
		return fmt.Errorf("failed to invoke a daemon process: %w", err)
	}
	_ = child.Release()
	log.Printf("daemon process invoked! ")
	return nil
	// todo pidfile
}
