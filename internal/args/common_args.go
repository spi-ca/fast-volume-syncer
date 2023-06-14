package args

import (
	"strconv"
	"strings"
	"time"
)

type SyncerCommonArguments struct {
	ReportDisabled     bool
	SandboxMountOption string

	Args RsyncArgs

	UseRsync           bool
	SourceMountHost    string
	SourceMountOptions string
	SourceMountName    string

	DestinationMountHost    string
	DestinationMountOptions string
	DestinationMountName    string

	ScanDuration     time.Duration
	FinderBinaryPath string

	TaskSize  int
	ChunkSize int
	Retry     RetryArgs
}

func (i *SyncerCommonArguments) AssembleEnvironment(inherited []string) []string {
	envs := make([]string, 0, 23)

	envs = append(envs, "REPORT_DISABLED", strconv.FormatBool(i.ReportDisabled))
	envs = append(envs, "SANDBOX_MOUNT_OPTION", i.SandboxMountOption)

	envs = append(envs, "RSYNC_ENABLED", strconv.FormatBool(i.UseRsync))
	envs = append(envs, "RSYNC_VERBOSE", strconv.FormatBool(i.Args.Verbose))
	envs = append(envs, "RSYNC_DELETE", strconv.FormatBool(i.Args.Delete))
	envs = append(envs, "RSYNC_PERMS", strconv.FormatBool(i.Args.PreservePermission))
	envs = append(envs, "RSYNC_OWNER", strconv.FormatBool(i.Args.PreserveOwnership))
	envs = append(envs, "RSYNC_SPECIAL", strconv.FormatBool(i.Args.CopySpecial))
	envs = append(envs, "RSYNC_COMPRESS", strconv.FormatBool(i.Args.Compress))
	envs = append(envs, "RSYNC_WHOLE_FILE", strconv.FormatBool(i.Args.WholeFile))
	envs = append(envs, "RSYNC_INPLACE", strconv.FormatBool(i.Args.Inplace))
	envs = append(envs, "RSYNC_RECURSIVE", strconv.FormatBool(i.Args.Recursive))
	envs = append(envs, "RSYNC_BANDWIDTH_LIMIT", i.Args.BandwidthLimit)

	envs = append(envs, "SRC_STORAGE_MOUNT_HOST", i.SourceMountHost)
	envs = append(envs, "SRC_STORAGE_MOUNT_OPTION", i.SourceMountOptions)
	envs = append(envs, "SRC_STORAGE_MOUNT_NAME", i.SourceMountName)

	envs = append(envs, "DST_STORAGE_MOUNT_HOST", i.DestinationMountHost)
	envs = append(envs, "DST_STORAGE_MOUNT_OPTION", i.DestinationMountOptions)
	envs = append(envs, "DST_STORAGE_MOUNT_NAME", i.DestinationMountName)

	envs = append(envs, "SCAN_DEADLINE", i.ScanDuration.String())
	envs = append(envs, "SCAN_FIND_PATH", i.FinderBinaryPath)

	envs = append(envs, "TASK_SIZE", strconv.Itoa(i.TaskSize))
	envs = append(envs, "CHUNK_SIZE", strconv.Itoa(i.ChunkSize))

	envs = append(envs, "RETRY_ATTEMPTS", strconv.Itoa(i.Retry.Attempts))
	envs = append(envs, "RETRY_DELAY", i.Retry.Delay.String())
	envs = append(envs, "RETRY_MAX_DELAY", i.Retry.MaxDelay.String())
	envs = append(envs, "RETRY_MAX_JITTER", i.Retry.MaxJitter.String())

	b := strings.Builder{}
	for i := 0; i < len(envs)/2; i++ {
		b.WriteString(envs[i*2])
		b.WriteByte('=')
		b.WriteString(envs[i*2+1])
		inherited = append(inherited, b.String())
		b.Reset()
	}
	return inherited
}
