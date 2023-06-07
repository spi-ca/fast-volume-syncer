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

	log.Print("sandbox.disabled=", viper.GetString("sandbox.disabled"))
	log.Print("sandbox.mount.option=", viper.GetString("sandbox.mount.option"))
	log.Print("rsync.verbose=", viper.GetBool("rsync.verbose"))
	log.Print("rsync.perms=", viper.GetBool("rsync.perms"))
	log.Print("rsync.owner=", viper.GetBool("rsync.owner"))
	log.Print("rsync.special=", viper.GetBool("rsync.special"))
	log.Print("rsync.compress=", viper.GetBool("rsync.compress"))
	log.Print("rsync.whole.file=", viper.GetBool("rsync.whole.file"))
	log.Print("rsync.inplace=", viper.GetBool("rsync.inplace"))
	log.Print("rsync.recursive=", viper.GetBool("rsync.recursive"))
	log.Print("src.storage.mount.host=", viper.GetString("src.storage.mount.host"))
	log.Print("src.storage.mount.option=", viper.GetString("src.storage.mount.option"))
	log.Print("src.storage.mount.name=", viper.GetString("src.storage.mount.name"))
	log.Print("dst.storage.mount.host=", viper.GetString("dst.storage.mount.host"))
	log.Print("dst.storage.mount.option=", viper.GetString("dst.storage.mount.option"))
	log.Print("dst.storage.mount.name=", viper.GetString("dst.storage.mount.name"))
	log.Print("scan.deadline=", viper.GetDuration("scan.deadline"))
	log.Print("scan.find.path=", viper.GetString("scan.find.path"))
	log.Print("worker.size=", viper.GetString("worker.size"))
	log.Print("task.size=", viper.GetInt("task.size"))
	log.Print("chunk.size=", viper.GetInt("chunk.size"))
	log.Print("retry.attempts=", viper.GetInt("retry.attempts"))
	log.Print("retry.delay=", viper.GetDuration("retry.delay"))
	log.Print("retry.max.delay=", viper.GetDuration("retry.max.delay"))
	log.Print("retry.max.jiiter=", viper.GetDuration("retry.max.jiiter"))
	log.Print("daemonized=", daemonized)
	log.Print("sandboxSupported=", sandboxSupported)
	log.Print("---")

	runner := selector.Runner{
		NodeSelector:    argNodeSelector,
		CopyInfoCSVPath: argCopyInfoFilePath,

		WorkerSize: viper.GetInt("worker.size"),

		Template: selector.Invoker{
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
		},
	}
	started := time.Now()
	if err := runner.Execute(ctx); err != nil {
		log.Fatal(err)
	}
	ended := time.Now()
	log.Printf("completed: in %s", ended.Sub(started))
}
