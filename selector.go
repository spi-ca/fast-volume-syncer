package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/selector"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
)

func selectorEntry() {
	ctx, cancel := context.WithCancel(context.Background())

	// 시그널 처리
	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	defer signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	go func() {
		select {
		case <-ctx.Done():
			return
		case sysSignal := <-exitSignal:
			log.Println(sysSignal.String(), " received")
			cancel()
			return
		}
	}()

	daemonized, _ := strconv.ParseBool(os.Getenv("_FVS_DAEMONEZED"))

	if daemonized {
		_ = common.SetProcessName("selector")
	}

	runner := selector.Runner{
		NodeSelector:    argNodeSelector,
		CopyInfoCSVPath: argCopyInfoFilePath,

		WorkerSize: viper.GetInt("worker.size"),

		Template: selector.Invoker{
			SandboxDisabled:    viper.GetBool("sandbox.disabled") || !sandboxSupported,
			SandboxMountOption: viper.GetString("sandbox.mount.option"),

			Args: rsync.Args{
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
	started := time.Now()
	if err := runner.Execute(ctx); err != nil {
		log.Fatal(err)
	}
	ended := time.Now()
	log.Printf("completed: in %s", ended.Sub(started))
}
