package main

import (
	"github.com/spf13/viper"
	"os"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/model"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/selector"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

func daemonStopEntry() {
	pidFilePath := viper.GetString("pid.file")
	pid, err := util.ReadPidFile(pidFilePath)
	if err != nil {
		util.ErrLog.Fatal(err)
	} else if pid < 1 {
		util.ErrLog.Fatalf("invalid pid(%d)")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		util.ErrLog.Fatalf("failed to find process(%d) :%w", pid, err)
	}
	defer proc.Release()

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		util.ErrLog.Fatalf("failed to SIGTERM(%d) :%w", pid, err)
	}
	util.InfoLog.Printf("sending SIGTERM(%d)", pid)
}

func daemonStartEntry() {

	util.InfoLog.Print("args:")
	util.InfoLog.Print("	pid.file=", viper.GetString("pid.file"))
	util.InfoLog.Print("	log.file=", viper.GetString("log.file"))
	util.InfoLog.Print("	sandbox.disabled=", viper.GetString("sandbox.disabled"))
	util.InfoLog.Print("	sandbox.mount.option=", viper.GetString("sandbox.mount.option"))
	util.InfoLog.Print("	rsync.verbose=", viper.GetBool("rsync.verbose"))
	util.InfoLog.Print("	rsync.perms=", viper.GetBool("rsync.perms"))
	util.InfoLog.Print("	rsync.owner=", viper.GetBool("rsync.owner"))
	util.InfoLog.Print("	rsync.special=", viper.GetBool("rsync.special"))
	util.InfoLog.Print("	rsync.compress=", viper.GetBool("rsync.compress"))
	util.InfoLog.Print("	rsync.whole.file=", viper.GetBool("rsync.whole.file"))
	util.InfoLog.Print("	rsync.inplace=", viper.GetBool("rsync.inplace"))
	util.InfoLog.Print("	rsync.recursive=", viper.GetBool("rsync.recursive"))
	util.InfoLog.Print("	src.storage.mount.host=", viper.GetString("src.storage.mount.host"))
	util.InfoLog.Print("	src.storage.mount.option=", viper.GetString("src.storage.mount.option"))
	util.InfoLog.Print("	src.storage.mount.name=", viper.GetString("src.storage.mount.name"))
	util.InfoLog.Print("	dst.storage.mount.host=", viper.GetString("dst.storage.mount.host"))
	util.InfoLog.Print("	dst.storage.mount.option=", viper.GetString("dst.storage.mount.option"))
	util.InfoLog.Print("	dst.storage.mount.name=", viper.GetString("dst.storage.mount.name"))
	util.InfoLog.Print("	scan.deadline=", viper.GetDuration("scan.deadline"))
	util.InfoLog.Print("	scan.find.path=", viper.GetString("scan.find.path"))
	util.InfoLog.Print("	worker.size=", viper.GetString("worker.size"))
	util.InfoLog.Print("	task.size=", viper.GetInt("task.size"))
	util.InfoLog.Print("	chunk.size=", viper.GetInt("chunk.size"))
	util.InfoLog.Print("	retry.attempts=", viper.GetInt("retry.attempts"))
	util.InfoLog.Print("	retry.delay=", viper.GetDuration("retry.delay"))
	util.InfoLog.Print("	retry.max.delay=", viper.GetDuration("retry.max.delay"))
	util.InfoLog.Print("	retry.max.jiiter=", viper.GetDuration("retry.max.jiiter"))
	util.InfoLog.Print("	sandboxSupported=", sandboxSupported)
	util.InfoLog.Print("---")

	runner := selector.Daemonizer{
		NodeSelector:    argNodeSelector,
		CopyInfoCSVPath: argCopyInfoFilePath,
		PidFilePath:     viper.GetString("pid.file"),
		LogFilePath:     viper.GetString("log.file"),
		WorkerSize:      viper.GetInt("worker.size"),
		SandboxDisabled: viper.GetBool("sandbox.disabled") || !sandboxSupported,
		Common: model.SyncerCommonArguments{
			SandboxMountOption: viper.GetString("sandbox.mount.option"),
			Args: model.RsyncArgs{
				Verbose:            viper.GetBool("rsync.verbose"),
				PreservePermission: viper.GetBool("rsync.perms"),
				PreserveOwnership:  viper.GetBool("rsync.owner"),
				CopySpecial:        viper.GetBool("rsync.special"),
				Compress:           viper.GetBool("rsync.compress"),
				WholeFile:          viper.GetBool("rsync.whole.file"),
				Inplace:            viper.GetBool("rsync.inplace"),
				Recursive:          viper.GetBool("rsync.recursive"),
			},
			SourceMountHost:    viper.GetString("src.storage.mount.host"),
			SourceMountOptions: viper.GetString("src.storage.mount.option"),
			SourceMountName:    viper.GetString("src.storage.mount.name"),

			DestinationMountHost:    viper.GetString("dst.storage.mount.host"),
			DestinationMountOptions: viper.GetString("dst.storage.mount.option"),
			DestinationMountName:    viper.GetString("dst.storage.mount.name"),

			ScanDuration:     viper.GetDuration("scan.deadline"),
			FinderBinaryPath: viper.GetString("scan.find.path"),
			TaskSize:         viper.GetInt("task.size"),
			ChunkSize:        viper.GetInt("chunk.size"),
			RetryAttempts:    viper.GetInt("retry.attempts"),
			RetryDelay:       viper.GetDuration("retry.delay"),
			RetryMaxDelay:    viper.GetDuration("retry.max.delay"),
			RetryMaxJitter:   viper.GetDuration("retry.max.jiiter"),
		},
	}
	if err := runner.Execute(); err != nil {
		util.ErrLog.Fatal(err)
	}
	util.InfoLog.Println("daemon started")
}
