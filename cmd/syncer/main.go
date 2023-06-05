package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	flag "github.com/spf13/pflag"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/rsync"
)

const (
	name = "syncer"
)

var (
	argSandboxDisabled    = flag.Bool("sandbox-disabled", false, "without namespace isolation")
	argSandboxMountOption = flag.String("sandbox-mount-option", "size=150M,mode=700,nosuid,noexec,nodev", "sandbox mount option")

	argRsyncVerbose            = flag.Bool("rsync-verbose", false, "make rsync verbosely")
	argRsyncPreservePermission = flag.Bool("rsync-perms", false, "preserve source file mode")
	argRsyncPreserveOwner      = flag.Bool("rsync-owner", false, "preserve source file ownership")
	argRsyncCopySpecialFile    = flag.Bool("rsync-special", false, "copy special/device/fifo file")
	argRsyncCompress           = flag.Bool("rsync-compress", false, "send with compressed")
	argRsyncWholeFile          = flag.Bool("rsync-whole-file", false, "disable delta xfer of rsync")
	argRsyncInplace            = flag.Bool("rsync-inplace", false, "write file directly info destination path")
	argRsyncRecursive          = flag.Bool("rsync-recursive", false, "disable chunk xfer")

	argSrcStorageHost        = flag.String("src-storage-host", "192.0.2.10", "source storage host")
	argSrcStorageMountOption = flag.String("src-storage-mount-option", "ro,nodiratime,noatime,vers=3,rsize=524288,wsize=524288,hard,nolock,proto=tcp,timeo=600,retrans=2,sec=sys", "source mount option")
	argSrcStorageMountName   = flag.String("src-storage-mount-name", "src", "source mountpoint name. e.g. /tmp/rand_path/*src*")

	argDstStorageHost        = flag.String("dst-storage-host", "192.0.2.11", "destination storage host")
	argDstStorageMountOption = flag.String("dst-storage-mount-option", "rw,nodiratime,noatime,vers=3,rsize=524288,wsize=524288,hard,nolock,proto=tcp,timeo=600,retrans=2,sec=sys", "destination mount option")
	argDstStorageMountName   = flag.String("dst-storage-mount-name", "dst", "destination mountpoint name e.g. /tmp/rand_path/*dst*")

	argScanDeadline = flag.Duration("scan-deadline", 3*time.Second, "scanning output deadline")
	argScanFindPath = flag.String("scan-find-path", "./find", "specify find binary path, or use golang implementation")

	argTaskSize  = flag.IntP("task-size", "t", 30, "specifies the maximum number of rsync processes that can run concurrently")
	argChunkSize = flag.IntP("chunk-size", "c", 4000, "specifies how many files rsync will write at once")

	argRetryAttempts  = flag.Int("retry-attempts", 7, "specifies the maximum number of retries. less than or equal to 0 means no retries.")
	argRetryDelay     = flag.Duration("retry-delay", 5*time.Second, "specifies the amount of time to wait between attempts.")
	argRetryMaxDelay  = flag.Duration("retry-max-delay", 5*time.Minute, "specifies the maximum amount of time to wait between attempts. if less than or equal to 0, retries are performed at fixed time intervals rather than backoff policy.")
	argRetryMaxJitter = flag.Duration("retry-max-jitter", 7*time.Second, "specifies the jitter between retries. less than or equal to 0 sets no jitter.")

	argSrcStoragePath    string
	argSrcStorageSubPath string
	argDstStoragePath    string
	argDstStorageSubPath string
)

func init() {
	flag.Parse()
	switch flag.NArg() {
	case 2:
		argSrcStoragePath = flag.Arg(0)
		argDstStoragePath = flag.Arg(1)
	case 4:
		argSrcStoragePath = flag.Arg(0)
		argSrcStorageSubPath = flag.Arg(1)
		argDstStoragePath = flag.Arg(2)
		argDstStorageSubPath = flag.Arg(3)
	default:
		fmt.Println("required arguments missing")
		usage()
		os.Exit(1)
	}
}
func usage() {
	fmt.Printf("usage: %s SRC_PATH [SRC_SUBPATH] DST_PATH [DST_SUBPATH]\n", name)
	flag.PrintDefaults()

}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	name := fmt.Sprintf("%s[%d] ", name, os.Getpid())
	log.SetPrefix(name)

	r := &syncer.Runner{
		SandboxDisabled:    *argSandboxDisabled,
		SandboxMountOption: *argSandboxMountOption,
		Args: rsync.Args{
			Verbose:            *argRsyncVerbose,
			PreservePermission: *argRsyncPreservePermission,
			PreserveOwnership:  *argRsyncPreserveOwner,
			CopySpecial:        *argRsyncCopySpecialFile,
			Compress:           *argRsyncCompress,
			WholeFile:          *argRsyncWholeFile,
			Inplace:            *argRsyncInplace,
			Recursive:          *argRsyncRecursive,
		},
		Source: common.RemoteInfo{
			MountInfo: common.MountInfo{
				Host:    *argSrcStorageHost,
				Path:    argSrcStoragePath,
				Options: *argSrcStorageMountOption,
			},
			SubPath: argSrcStorageSubPath,
		},
		SourceMountName: *argSrcStorageMountName,
		Destination: common.RemoteInfo{
			MountInfo: common.MountInfo{
				Host:    *argDstStorageHost,
				Path:    argDstStoragePath,
				Options: *argDstStorageMountOption,
			},
			SubPath: argDstStorageSubPath,
		},
		DestinationMountName: *argDstStorageMountName,
		ScanDuration:         *argScanDeadline,
		FindBinaryPath:       *argScanFindPath,
		TaskSize:             *argTaskSize,
		ChunkSize:            *argChunkSize,
		RetryAttempts:        *argRetryAttempts,
		RetryDelay:           *argRetryDelay,
		RetryMaxDelay:        *argRetryMaxDelay,
		RetryMaxJitter:       *argRetryMaxJitter,
	}
	log.Printf("%v", r)
	started := time.Now()
	if err := r.Execute(ctx); err != nil {
		log.Fatal(err)
	}

	ended := time.Now()
	log.Printf("completed: in %s", ended.Sub(started))
}
