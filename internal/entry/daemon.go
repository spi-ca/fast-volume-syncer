// Package entry adapts CLI commands to signal-aware internal runners.
package entry

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/spf13/viper"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/selector"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// DaemonStop reads the selector pid file and asks the running daemonized selector to exit.
func DaemonStop() {
	pidFilePath := viper.GetString("pid.file")
	pid, err := selector.ReadPidFile(pidFilePath)
	if err != nil {
		util.ErrLog.Fatal(err)
	} else if pid < 1 {
		util.ErrLog.Fatalf("invalid pid(%d)", pid)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		util.ErrLog.Fatalf("failed to find process(%d) :%v", pid, err)
	}
	defer proc.Release()

	if err := verifyDaemonProcess(pid); err != nil {
		util.ErrLog.Fatalf("refusing to stop pid(%d): %v", pid, err)
	}

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		util.ErrLog.Fatalf("failed to SIGTERM(%d) :%v", pid, err)
	}
	util.InfoLog.Printf("sending SIGTERM(%d)", pid)
}

// verifyDaemonProcess confirms that a pid file still points to a fast-volume-syncer executable on Linux.
func verifyDaemonProcess(pid int) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	procExe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return fmt.Errorf("read process executable: %w", err)
	}
	currentExe := sys.Executable()
	if resolved, err := filepath.EvalSymlinks(currentExe); err == nil {
		currentExe = resolved
	}
	if resolved, err := filepath.EvalSymlinks(procExe); err == nil {
		procExe = resolved
	}
	procExe = strings.TrimSuffix(procExe, " (deleted)")
	if procExe != currentExe && filepath.Base(procExe) != filepath.Base(currentExe) {
		return fmt.Errorf("process executable %q does not match %q", procExe, currentExe)
	}
	environ, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return fmt.Errorf("read process environment: %w", err)
	}
	if !strings.Contains(string(environ), "_FVS_DAEMONEZED=true") {
		return fmt.Errorf("process is not a daemonized selector")
	}
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return fmt.Errorf("read process command line: %w", err)
	}
	if !strings.Contains(string(cmdline), "\x00select\x00") {
		return fmt.Errorf("process command line is not selector mode")
	}
	return nil
}

// DaemonStart builds a detached selector launcher from CLI configuration and starts it.
func DaemonStart(sandboxSupported bool, nodeSelector int, copyInfoFilePath string) {
	if copyInfoFilePath == "-" {
		util.ErrLog.Fatal("daemon start does not support stdin CSV path '-' ; use select _ - in the foreground or pass a file path")
	}
	util.CheckLogFile()
	util.InfoLog.Print(
		"args:",
		"\n	pid.file=", viper.GetString("pid.file"),
		"\n	log.file=", viper.GetString("log.file"),
		"\n	log.prefix=", viper.GetString("log.prefix"),
		"\n	report.enabled=", viper.GetBool("report.enabled"),
		"\n	sandbox.disabled=", viper.GetString("sandbox.disabled"),
		"\n	sandbox.mount.option=", viper.GetString("sandbox.mount.option"),
		"\n	file.mode=", viper.GetString("file.mode"),
		"\n	rsync.enabled=", viper.GetBool("rsync.enabled"),
		"\n	rsync.verbose=", viper.GetBool("rsync.verbose"),
		"\n	rsync.delete=", viper.GetBool("rsync.delete"),
		"\n	rsync.perms=", viper.GetBool("rsync.perms"),
		"\n	rsync.owner=", viper.GetBool("rsync.owner"),
		"\n	rsync.special=", viper.GetBool("rsync.special"),
		"\n	rsync.compress=", viper.GetBool("rsync.compress"),
		"\n	rsync.whole.file=", viper.GetBool("rsync.whole.file"),
		"\n	rsync.inplace=", viper.GetBool("rsync.inplace"),
		"\n	rsync.recursive=", viper.GetBool("rsync.recursive"),
		"\n	rsync.port=", viper.GetInt("rsync.port"),
		"\n	rsync.bandwidth.limit=", viper.GetString("rsync.bandwidth.limit"),
		"\n	src.storage.mount.host=", viper.GetString("src.storage.mount.host"),
		"\n	src.storage.mount.option=", viper.GetString("src.storage.mount.option"),
		"\n	src.storage.mount.name=", viper.GetString("src.storage.mount.name"),
		"\n	dst.storage.mount.host=", viper.GetString("dst.storage.mount.host"),
		"\n	dst.storage.mount.option=", viper.GetString("dst.storage.mount.option"),
		"\n	dst.storage.mount.name=", viper.GetString("dst.storage.mount.name"),
		"\n	scan.deadline=", viper.GetDuration("scan.deadline"),
		"\n	scan.find.path=", viper.GetString("scan.find.path"),
		"\n	worker.size=", viper.GetString("worker.size"),
		"\n	task.size=", viper.GetInt("task.size"),
		"\n	chunk.size=", viper.GetInt("chunk.size"),
		"\n	retry.attempts=", viper.GetInt("retry.attempts"),
		"\n	retry.delay=", viper.GetDuration("retry.delay"),
		"\n	retry.max.delay=", viper.GetDuration("retry.max.delay"),
		"\n	retry.max.jitter=", viper.GetDuration("retry.max.jitter"),
		"\n	sandboxSupported=", sandboxSupported,
		"\n---",
	)

	runner := selector.Daemonizer{
		NodeSelector:    nodeSelector,
		CopyInfoCSVPath: copyInfoFilePath,
		PidFilePath:     viper.GetString("pid.file"),
		LogFilePath:     viper.GetString("log.file"),
		WorkerSize:      viper.GetInt("worker.size"),
		SandboxDisabled: viper.GetBool("sandbox.disabled") || !sandboxSupported,
		Common: args.SyncerCommonArguments{
			ReportEnabled:      viper.GetBool("report.enabled"),
			SandboxMountOption: viper.GetString("sandbox.mount.option"),
			SourceMountHost:    viper.GetString("src.storage.mount.host"),
			SourceMountOptions: viper.GetString("src.storage.mount.option"),
			SourceMountName:    viper.GetString("src.storage.mount.name"),

			DestinationMountHost:    viper.GetString("dst.storage.mount.host"),
			DestinationMountOptions: viper.GetString("dst.storage.mount.option"),
			DestinationMountName:    viper.GetString("dst.storage.mount.name"),
			Common: args.CopierCommonArguments{
				LogPrefix: viper.GetString("log.prefix"),
				FileMode:  sys.UnFilemodeStr(viper.GetString("file.mode")),
				Args: args.RsyncArgs{
					Verbose:            viper.GetBool("rsync.verbose"),
					Delete:             viper.GetBool("rsync.delete"),
					PreservePermission: viper.GetBool("rsync.perms"),
					PreserveOwnership:  viper.GetBool("rsync.owner"),
					CopySpecial:        viper.GetBool("rsync.special"),
					Compress:           viper.GetBool("rsync.compress"),
					WholeFile:          viper.GetBool("rsync.whole.file"),
					Inplace:            viper.GetBool("rsync.inplace"),
					Recursive:          viper.GetBool("rsync.recursive"),
					Port:               viper.GetInt("rsync.port"),
					BandwidthLimit:     viper.GetString("rsync.bandwidth.limit"),
				},
				UseRsync:         viper.GetBool("rsync.enabled"),
				ScanDuration:     viper.GetDuration("scan.deadline"),
				FinderBinaryPath: viper.GetString("scan.find.path"),
				TaskSize:         viper.GetInt("task.size"),
				ChunkSize:        viper.GetInt("chunk.size"),
				Retry: args.RetryArgs{
					Attempts:  viper.GetInt("retry.attempts"),
					Delay:     viper.GetDuration("retry.delay"),
					MaxDelay:  viper.GetDuration("retry.max.delay"),
					MaxJitter: viper.GetDuration("retry.max.jitter"),
				},
			},
		},
	}
	if err := runner.Execute(); err != nil {
		util.ErrLog.Fatal(err)
	}
	util.InfoLog.Println("daemon started")
}
