package entry

import (
	log "github.com/sirupsen/logrus"
	"os"
	"syscall"

	"github.com/spf13/viper"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/selector"
)

func DaemonStop() {
	pidFilePath := viper.GetString("pid.file")
	pid, err := selector.ReadPidFile(pidFilePath)
	if err != nil {
		log.Fatal(err)
	} else if pid < 1 {
		log.Fatalf("invalid pid(%d)", pid)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("failed to find process(%d) :%v", pid, err)
	}
	defer proc.Release()

	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		log.Fatalf("failed to SIGTERM(%d) :%v", pid, err)
	}
	log.Infof("sending SIGTERM(%d)", pid)
}

func DaemonStart(sandboxSupported bool, nodeSelector int, copyInfoFilePath string) {

	log.Info("args:")
	log.Info("	pid.file=", viper.GetString("pid.file"))
	log.Info("	log.file=", viper.GetString("log.file"))
	log.Info("	monitoring.disabled=", viper.GetBool("monitoring.disabled"))
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
	log.Info("	sandboxSupported=", sandboxSupported)
	log.Info("---")

	runner := selector.Daemonizer{
		SlackMonitoring: !viper.GetBool("monitoring.disabled"),
		NodeSelector:    nodeSelector,
		CopyInfoCSVPath: copyInfoFilePath,
		PidFilePath:     viper.GetString("pid.file"),
		LogFilePath:     viper.GetString("log.file"),
		WorkerSize:      viper.GetInt("worker.size"),
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
	}
	if err := runner.Execute(); err != nil {
		log.Fatal(err)
	}
	log.Info("daemon started")
}
