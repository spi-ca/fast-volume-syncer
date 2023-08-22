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
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

func Copier(srcPath, dstPath string) {
	ctx, cancel := context.WithCancel(context.Background())

	// 시그널 처리
	exitSignal := make(chan os.Signal, 1)
	signal.Notify(exitSignal, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	defer signal.Ignore(syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
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
	if daemonized || selectorInvoked {
		util.SetLogFlags(0)
	} else {
		util.InfoLog.SetPrefix(fmt.Sprintf("%s[%d]&1>", viper.GetString("log.prefix"), os.Getpid()))
		util.ErrLog.SetPrefix(fmt.Sprintf("%s[%d]&2>", viper.GetString("log.prefix"), os.Getpid()))
	}

	// debug
	util.InfoLog.Print(
		"args:",
		"\n	log.prefix=", viper.GetString("log.prefix"),
		"\n	file.mode=", viper.GetString("file.mode"),
		"\n	rsync.enabled=", viper.GetBool("rsync.enabled"),
		"\n	rsync.delete=", viper.GetBool("rsync.delete"),
		"\n	rsync.perms=", viper.GetBool("rsync.perms"),
		"\n	rsync.owner=", viper.GetBool("rsync.owner"),
		"\n	rsync.special=", viper.GetBool("rsync.special"),
		"\n	rsync.compress=", viper.GetBool("rsync.compress"),
		"\n	rsync.whole.file=", viper.GetBool("rsync.whole.file"),
		"\n	rsync.inplace=", viper.GetBool("rsync.inplace"),
		"\n	rsync.recursive=", viper.GetBool("rsync.recursive"),
		"\n	rsync.bandwidth.limit=", viper.GetString("rsync.bandwidth.limit"),
		"\n	scan.deadline=", viper.GetDuration("scan.deadline"),
		"\n	scan.find.path=", viper.GetString("scan.find.path"),
		"\n	task.size=", viper.GetInt("task.size"),
		"\n	chunk.size=", viper.GetInt("chunk.size"),
		"\n	retry.attempts=", viper.GetInt("retry.attempts"),
		"\n	retry.delay=", viper.GetDuration("retry.delay"),
		"\n	retry.max.delay=", viper.GetDuration("retry.max.delay"),
		"\n	retry.max.jitter=", viper.GetDuration("retry.max.jitter"),
		"\n	daemonized=", daemonized,
		"\n	argSrcPath=", srcPath,
		"\n	argDstPath=", dstPath,
		"\n	env['_FVS_DAEMONEZED']=", os.Getenv("_FVS_DAEMONEZED"),
		"\n	env['_SYNCER_INVOKED']=", os.Getenv("_SYNCER_INVOKED"),
		"\n---",
	)

	util.InfoLog.Printf("fast-volume-sync/copier(%s -> %s) had been initiated", srcPath, dstPath)

	runner := copier.Runner{
		FileMode: sys.UnFilemodeStr(viper.GetString("file.mode")),
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
			BandwidthLimit:     viper.GetString("rsync.bandwidth.limit"),
		},
		UseRsync:         viper.GetBool("rsync.enabled"),
		ScanDuration:     viper.GetDuration("scan.deadline"),
		FinderBinaryPath: util.LookupBinary(viper.GetString("scan.find.path")),
		TaskSize:         viper.GetInt("task.size"),
		ChunkSize:        viper.GetInt("chunk.size"),
		Retry: args.RetryArgs{
			Attempts:  viper.GetInt("retry.attempts"),
			Delay:     viper.GetDuration("retry.delay"),
			MaxDelay:  viper.GetDuration("retry.max.delay"),
			MaxJitter: viper.GetDuration("retry.max.jitter"),
		},
	}
	started := time.Now()
	err := runner.Execute(ctx, srcPath, dstPath)
	ended := time.Now()
	if err == nil {
		util.InfoLog.Printf("fast-volume-sync/copier(%s -> %s) had been ended in %s", srcPath, dstPath, ended.Sub(started))
	} else {
		util.ErrLog.Fatal(err)
	}
}
