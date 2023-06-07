package main

import (
	"github.com/spf13/viper"
	"log"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/selector"
)

func daemonStopEntry() {
}

func daemonStartEntry() {
	runner := selector.Daemonizer{
		NodeSelector:    argNodeSelector,
		CopyInfoCSVPath: argCopyInfoFilePath,
		LogFilePath:     viper.GetString("log.file"),
		WorkerSize:      viper.GetInt("worker.size"),
		SandboxDisabled: viper.GetBool("sandbox.disabled") || !sandboxSupported,
		Common: common.Template{
			SandboxMountOption: viper.GetString("sandbox.mount.option"),
			Args: common.RsyncArgs{
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
		log.Fatal(err)
	}
	log.Println("daemon started")
}
