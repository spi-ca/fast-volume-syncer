package entry

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

func Syncer(
	sandboxSupported bool, srcStoragePath, srcStorageSubPath, dstStoragePath, dstStorageSubPath string,
) {
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
			util.ErrLog.Println(sysSignal.String(), " received")
			cancel()
			return
		}
	}()

	daemonized, _ := strconv.ParseBool(os.Getenv("_FVS_DAEMONEZED"))
	selectorInvoked, _ := strconv.ParseBool(os.Getenv("_SYNCER_INVOKED"))
	sandboxed, _ := strconv.ParseBool(os.Getenv("_SYNCER_SANDBOXED"))

	if daemonized || selectorInvoked {
		util.SetLogFlags(0)
	} else {
		util.InfoLog.SetPrefix(fmt.Sprintf("syncer[%d]&1> ", os.Getpid()))
		util.ErrLog.SetPrefix(fmt.Sprintf("syncer[%d]&2> ", os.Getpid()))
	}

	// debug
	util.InfoLog.Print("args:")
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
	util.InfoLog.Print("	task.size=", viper.GetInt("task.size"))
	util.InfoLog.Print("	chunk.size=", viper.GetInt("chunk.size"))
	util.InfoLog.Print("	retry.attempts=", viper.GetInt("retry.attempts"))
	util.InfoLog.Print("	retry.delay=", viper.GetDuration("retry.delay"))
	util.InfoLog.Print("	retry.max.delay=", viper.GetDuration("retry.max.delay"))
	util.InfoLog.Print("	retry.max.jitter=", viper.GetDuration("retry.max.jitter"))
	util.InfoLog.Print("	daemonized=", daemonized)
	util.InfoLog.Print("	selectorInvoked=", selectorInvoked)
	util.InfoLog.Print("	sandboxSupported=", sandboxSupported)
	util.InfoLog.Print("	sandboxed=", sandboxed)
	util.InfoLog.Print("	argSrcStoragePath=", srcStoragePath)
	util.InfoLog.Print("	argSrcStorageSubPath=", srcStorageSubPath)
	util.InfoLog.Print("	argDstStoragePath=", dstStoragePath)
	util.InfoLog.Print("	argDstStorageSubPath=", dstStorageSubPath)
	util.InfoLog.Print("	env['_FVS_DAEMONEZED']=", os.Getenv("_FVS_DAEMONEZED"))
	util.InfoLog.Print("	env['_SYNCER_INVOKED']=", os.Getenv("_SYNCER_INVOKED"))
	util.InfoLog.Print("	env['_SYNCER_SANDBOXED']=", os.Getenv("_SYNCER_SANDBOXED"))
	util.InfoLog.Print("---")
	//return

	runner := syncer.Runner{
		Sandboxed: selectorInvoked && sandboxed && sandboxSupported,
		Common: args.SyncerCommonArguments{
			SandboxMountOption: viper.GetString("sandbox.mount.option"),
			Args: args.RsyncArgs{
				Verbose:            viper.GetBool("rsync.verbose"),
				PreservePermission: viper.GetBool("rsync.perms"),
				PreserveOwnership:  viper.GetBool("rsync.owner"),
				CopySpecial:        viper.GetBool("rsync.special"),
				Compress:           viper.GetBool("rsync.compress"),
				WholeFile:          viper.GetBool("rsync.whole.file"),
				Inplace:            viper.GetBool("rsync.inplace"),
				Recursive:          viper.GetBool("rsync.recursive"),
			},

			SourceMountHost:         viper.GetString("src.storage.mount.host"),
			SourceMountOptions:      viper.GetString("src.storage.mount.option"),
			SourceMountName:         viper.GetString("src.storage.mount.name"),
			DestinationMountHost:    viper.GetString("dst.storage.mount.host"),
			DestinationMountOptions: viper.GetString("dst.storage.mount.option"),
			DestinationMountName:    viper.GetString("dst.storage.mount.name"),
			ScanDuration:            viper.GetDuration("scan.deadline"),
			FinderBinaryPath:        viper.GetString("scan.find.path"),
			TaskSize:                viper.GetInt("task.size"),
			ChunkSize:               viper.GetInt("chunk.size"),
			Retry: args.RetryArgs{
				Attempts:  viper.GetInt("retry.attempts"),
				Delay:     viper.GetDuration("retry.delay"),
				MaxDelay:  viper.GetDuration("retry.max.delay"),
				MaxJitter: viper.GetDuration("retry.max.jitter"),
			},
		},
		SourceMountPath:         srcStoragePath,
		SourceMountSubPath:      srcStorageSubPath,
		DestinationMountPath:    dstStoragePath,
		DestinationMountSubPath: dstStorageSubPath,
	}
	started := time.Now()
	if err := runner.Execute(ctx); err != nil {
		util.ErrLog.Fatal(err)
	}
	ended := time.Now()
	util.InfoLog.Printf("completed: in %s", ended.Sub(started))
}
