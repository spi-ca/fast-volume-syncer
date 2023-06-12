package entry

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/viper"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/selector"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

func Selector(sandboxSupported bool, nodeSelector int, copyInfoFilePath string) {
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
			log.Errorln(sysSignal.String(), " received")
			cancel()
			return
		}
	}()

	daemonized, _ := strconv.ParseBool(os.Getenv("_FVS_DAEMONEZED"))
	slackMonitoring, _ := strconv.ParseBool(os.Getenv("_SLACK_MONITORING"))

	log.Info("args:")
	log.Info("	sandbox.disabled=", viper.GetString("sandbox.disabled"))
	log.Info("	sandbox.mount.option=", viper.GetString("sandbox.mount.option"))
	log.Info("	rsync.verbose=", viper.GetBool("rsync.verbose"))
	log.Info("	rsync.delete=", viper.GetBool("rsync.delete"))
	log.Info("	rsync.perms=", viper.GetBool("rsync.perms"))
	log.Info("	rsync.owner=", viper.GetBool("rsync.owner"))
	log.Info("	rsync.special=", viper.GetBool("rsync.special"))
	log.Info("	rsync.compress=", viper.GetBool("rsync.compress"))
	log.Info("	rsync.whole.file=", viper.GetBool("rsync.whole.file"))
	log.Info("	rsync.inplace=", viper.GetBool("rsync.inplace"))
	log.Info("	rsync.recursive=", viper.GetBool("rsync.recursive"))
	log.Info("	rsync.bandwidth.limit=", viper.GetString("rsync.bandwidth.limit"))
	log.Info("	src.storage.mount.host=", viper.GetString("src.storage.mount.host"))
	log.Info("	src.storage.mount.option=", viper.GetString("src.storage.mount.option"))
	log.Info("	src.storage.mount.name=", viper.GetString("src.storage.mount.name"))
	log.Info("	dst.storage.mount.host=", viper.GetString("dst.storage.mount.host"))
	log.Info("	dst.storage.mount.option=", viper.GetString("dst.storage.mount.option"))
	log.Info("	dst.storage.mount.name=", viper.GetString("dst.storage.mount.name"))
	log.Info("	scan.deadline=", viper.GetDuration("scan.deadline"))
	log.Info("	scan.find.path=", viper.GetString("scan.find.path"))
	log.Info("	worker.size=", viper.GetString("worker.size"))
	log.Info("	task.size=", viper.GetInt("task.size"))
	log.Info("	chunk.size=", viper.GetInt("chunk.size"))
	log.Info("	retry.attempts=", viper.GetInt("retry.attempts"))
	log.Info("	retry.delay=", viper.GetDuration("retry.delay"))
	log.Info("	retry.max.delay=", viper.GetDuration("retry.max.delay"))
	log.Info("	retry.max.jitter=", viper.GetDuration("retry.max.jitter"))
	log.Info("	daemonized=", daemonized)
	log.Info("	sandboxSupported=", sandboxSupported)
	log.Info("	env['_FVS_DAEMONEZED']=", os.Getenv("_FVS_DAEMONEZED"))
	log.Info("	env['_SLACK_MONITORING']=", os.Getenv("_SLACK_MONITORING"))
	log.Info("---")

	if daemonized {
		if pidFilePath := os.Getenv("_PID_FILEPATH"); len(pidFilePath) > 0 {
			closer, err := selector.AcquirePidFile(pidFilePath)
			if err != nil {
				log.Errorln("selector already running : %v", err)
				return
			}
			defer closer()
		}
		if slackMonitoring {
			util.SlackSender.Start()
			defer util.SlackSender.Close()
			log.AddHook(util.SlackSender)
			log.Errorf(fmt.Sprintf("fast-volume-sync/selector@%d(daemonized:%t) had been initiated", nodeSelector, daemonized))
		}
	}

	runner := selector.Runner{
		NodeSelector:    nodeSelector,
		CopyInfoCSVPath: copyInfoFilePath,

		WorkerSize: viper.GetInt("worker.size"),

		Template: selector.Invoker{
			SandboxDisabled: viper.GetBool("sandbox.disabled") || !sandboxSupported,
			Common: args.SyncerCommonArguments{
				SandboxMountOption: viper.GetString("sandbox.mount.option"),
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
				Retry: args.RetryArgs{
					Attempts:  viper.GetInt("retry.attempts"),
					Delay:     viper.GetDuration("retry.delay"),
					MaxDelay:  viper.GetDuration("retry.max.delay"),
					MaxJitter: viper.GetDuration("retry.max.jitter"),
				},
			},
		},
	}
	started := time.Now()
	err := runner.Execute(ctx)
	ended := time.Now()

	if err != nil {
		log.Errorf("fast-volume-sync/selector@%d ended with error(s) in %s. errors: %v", nodeSelector, ended.Sub(started), err)
	} else {
		log.Errorf("fast-volume-sync/selector@%d processed all copy entries in %s.", nodeSelector, ended.Sub(started))
	}
}
