package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"

	flags "github.com/spf13/pflag"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/entry"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

const (
	name                = "fast-volume-syncer"
	defaultNodeSelector = -1
	defaultCSVFilename  = "data/09_copy_entries.csv"
)

var (
	sandboxSupported = strings.Compare(runtime.GOOS, "linux") == 0
	flagNameReplacer = strings.NewReplacer("-", ".", "_", ".")
	envNameReplacer  = strings.NewReplacer(".", "_", "-", "_")
)

func init() {
	flags.Bool("monitoring-disabled", false, "(daemon only)without slack monitoring")
	flags.String("log-file", fmt.Sprintf("log/%s.log", name), "(daemon only)specify a log file")
	flags.String("pid-file", fmt.Sprintf("%s.pid", name), "(daemon only)specify a pid file")
	flags.Bool("sandbox-disabled", false, "(selector only)without namespace isolation")
	flags.IntP("worker-size", "w", 5, "(selector only)specifies the maximum number of syncer processes that can run concurrently")
	flags.Bool("report-disabled", false, "don't do list files")

	flags.String("sandbox-mount-option", "size=150M,mode=700,nosuid,noexec,nodev", "(selector only)sandbox mount option")
	flags.Bool("rsync-enabled", false, "use rsync method")
	flags.Bool("rsync-verbose", false, "make rsync verbosely")
	flags.Bool("rsync-delete", false, "wipe conflicted destination path")
	flags.Bool("rsync-perms", false, "preserve source file mode")
	flags.Bool("rsync-owner", false, "preserve source file ownership")
	flags.Bool("rsync-special", false, "copy special/device/fifo file")
	flags.Bool("rsync-compress", false, "send with compressed")
	flags.Bool("rsync-whole-file", false, "disable delta xfer of rsync")
	flags.Bool("rsync-inplace", false, "write file directly info destination path")
	flags.Bool("rsync-recursive", false, "disable chunk xfer")
	flags.String("rsync-bandwidth-limit", "", "specify bandwidth limitation")
	flags.String("src-storage-mount-host", "192.0.2.10", "source storage host")
	flags.String("src-storage-mount-option", "ro,nodiratime,noatime,vers=3,rsize=524288,wsize=524288,hard,intr,nolock,proto=tcp,timeo=600,retrans=2,sec=sys", "source mount option")
	flags.String("src-storage-mount-name", "src", "source mountpoint name. e.g. /tmp/rand_path/*src*")
	flags.String("dst-storage-mount-host", "192.0.2.11", "destination storage host")
	flags.String("dst-storage-mount-option", "rw,nodiratime,noatime,vers=3,rsize=524288,wsize=524288,hard,intr,nolock,proto=tcp,timeo=600,retrans=2,sec=sys", "destination mount option")
	flags.String("dst-storage-mount-name", "dst", "destination mountpoint name e.g. /tmp/rand_path/*dst*")
	flags.Duration("scan-deadline", 3*time.Second, "scanning output deadline")
	flags.String("scan-find-path", "find", "specify find binary path, or use golang implementation")
	flags.IntP("task-size", "t", 30, "specifies the maximum number of rsync processes that can run concurrently")
	flags.IntP("chunk-size", "c", 500, "specifies how many files rsync will write at once")
	flags.Int("retry-attempts", 3, "specifies the maximum number of retries. less than or equal to 0 means no retries.")
	flags.Duration("retry-delay", 5*time.Second, "specifies the amount of time to wait between attempts.")
	flags.Duration("retry-max-delay", 5*time.Minute, "specifies the maximum amount of time to wait between attempts. if less than or equal to 0, retries are performed at fixed time intervals rather than backoff policy.")
	flags.Duration("retry-max-jitter", 7*time.Second, "specifies the jitter between retries. less than or equal to 0 sets no jitter.")

	flags.Parse()
	viper.SetEnvKeyReplacer(envNameReplacer)
	viper.AutomaticEnv()
	_ = viper.BindFlagValues(util.PFlagViperReplacer{FlagSet: flags.CommandLine, Replacer: flagNameReplacer})
}

func main() {

	consumedArgs := 0
	if flags.NArg() == 0 {
		usage()
	}

	action := flags.Arg(0)
	consumedArgs++

	switch action {
	case "sync":
		var (
			srcStoragePath,
			srcStorageSubPath,
			dstStoragePath,
			dstStorageSubPath string
		)
		switch flags.NArg() {
		case consumedArgs + 2:
			srcStoragePath = flags.Arg(consumedArgs + 0)
			dstStoragePath = flags.Arg(consumedArgs + 1)
			consumedArgs += 2
		case consumedArgs + 4:
			srcStoragePath = flags.Arg(consumedArgs + 0)
			srcStorageSubPath = flags.Arg(consumedArgs + 1)
			dstStoragePath = flags.Arg(consumedArgs + 2)
			dstStorageSubPath = flags.Arg(consumedArgs + 3)
			consumedArgs += 4
		default:
			fmt.Println("required arguments missing")
			usage()
		}
		entry.Syncer(sandboxSupported, srcStoragePath, srcStorageSubPath, dstStoragePath, dstStorageSubPath)
	case "select":
		var (
			nodeSelector     = defaultNodeSelector
			copyInfoFilePath = defaultCSVFilename
		)
		switch flags.NArg() {
		case consumedArgs:
			consumedArgs += 0
		case consumedArgs + 1:
			if rawNodeSelector, err := strconv.Atoi(flags.Arg(consumedArgs + 0)); err == nil {
				nodeSelector = rawNodeSelector
			} else {
				fmt.Println("failed to parse nodeSelector:%w", err)
				usage()
			}
			consumedArgs += 1
		case consumedArgs + 2:
			if flag := flags.Arg(consumedArgs + 0); flag == "_" {
				// do nothing
			} else if rawNodeSelector, err := strconv.Atoi(flag); err == nil {
				nodeSelector = rawNodeSelector
			} else {
				fmt.Println("failed to parse nodeSelector:%w", err)
				usage()
			}
			copyInfoFilePath = flags.Arg(consumedArgs + 1)
			consumedArgs += 2
		default:
			fmt.Println("required arguments missing")
			usage()
		}
		entry.Selector(sandboxSupported, nodeSelector, copyInfoFilePath)
	case "start":
		var (
			nodeSelector     = defaultNodeSelector
			copyInfoFilePath = defaultCSVFilename
		)
		switch flags.NArg() {
		case consumedArgs:
			consumedArgs += 0
		case consumedArgs + 1:
			if flag := flags.Arg(consumedArgs + 0); flag == "_" {
				// do nothing
			} else if rawNodeSelector, err := strconv.Atoi(flag); err == nil {
				nodeSelector = rawNodeSelector
			} else {
				fmt.Println("failed to parse nodeSelector:%w", err)
				usage()
			}
			consumedArgs += 1
		case consumedArgs + 2:
			if flag := flags.Arg(consumedArgs + 0); flag == "_" {
				// do nothing
			} else if rawNodeSelector, err := strconv.Atoi(flag); err == nil {
				nodeSelector = rawNodeSelector
			} else {
				fmt.Println("failed to parse nodeSelector:%w", err)
				usage()
			}
			copyInfoFilePath = flags.Arg(consumedArgs + 1)
			consumedArgs += 2
		default:
			fmt.Println("required arguments missing")
			usage()
		}
		entry.DaemonStart(sandboxSupported, nodeSelector, copyInfoFilePath)
	case "stop":
		entry.DaemonStop()
	default:
		fmt.Printf("invalid action %s\n", action)
		usage()
	}
}

func usage() {
	fmt.Printf("usage: \n"+
		"\t%s sync SRC_PATH [SRC_SUBPATH] DST_PATH [DST_SUBPATH]\n"+
		"\t%s select [NODE_SELECTOR:%d] [COPY_INFO_CSV_PATH:%s]\n"+
		"\t%s start [NODE_SELECTOR:%d] [COPY_INFO_CSV_PATH:%s]\n"+
		"\t%s stop \nargs:\n",
		name,
		name, defaultNodeSelector, defaultCSVFilename,
		name, defaultNodeSelector, defaultCSVFilename,
		name,
	)
	flags.PrintDefaults()
	os.Exit(1)
}
