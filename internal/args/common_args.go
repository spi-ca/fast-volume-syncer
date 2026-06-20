// Package args assembles environment and command arguments for sync and copy workers.
package args

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// CopierCommonArguments holds copier settings that are exported as worker environment variables.
type CopierCommonArguments struct {
	// FileMode is serialized into FILE_MODE for created destination entries.
	FileMode os.FileMode
	// LogPrefix is serialized into LOG_PREFIX for child-process logging.
	LogPrefix string

	// Args provides the rsync-specific settings exported as RSYNC_* variables.
	Args RsyncArgs

	// UseRsync controls whether the child process enables rsync mode.
	UseRsync bool

	// ScanDuration is serialized into SCAN_DEADLINE for directory scans.
	ScanDuration time.Duration
	// FinderBinaryPath is serialized into SCAN_FIND_PATH for external find execution.
	FinderBinaryPath string

	// TaskSize is serialized into TASK_SIZE for worker fan-out.
	TaskSize int
	// ChunkSize is serialized into CHUNK_SIZE for per-batch file counts.
	ChunkSize int
	// Retry provides RETRY_* values for child retry behavior.
	Retry RetryArgs
}

// AssembleEnvironment appends KEY=VALUE pairs for copier settings onto inherited.
func (i *CopierCommonArguments) AssembleEnvironment(inherited []string) []string {
	envs := make([]string, 0, 20*2)

	envs = append(envs, "FILE_MODE", i.FileMode.String())
	envs = append(envs, "LOG_PREFIX", i.LogPrefix)
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
	envs = append(envs, "RSYNC_PORT", strconv.Itoa(i.Args.Port))
	envs = append(envs, "RSYNC_BANDWIDTH_LIMIT", i.Args.BandwidthLimit)

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

// SyncerCommonArguments holds sync-only environment variables on top of copier settings.
type SyncerCommonArguments struct {
	// ReportEnabled is serialized into REPORT_ENABLED for reporting hooks.
	ReportEnabled bool
	// SandboxMountOption is serialized into SANDBOX_MOUNT_OPTION for sandbox mounts.
	SandboxMountOption string

	// SourceMountHost is serialized into SRC_STORAGE_MOUNT_HOST.
	SourceMountHost string
	// SourceMountOptions is serialized into SRC_STORAGE_MOUNT_OPTION.
	SourceMountOptions string
	// SourceMountName is serialized into SRC_STORAGE_MOUNT_NAME.
	SourceMountName string

	// DestinationMountHost is serialized into DST_STORAGE_MOUNT_HOST.
	DestinationMountHost string
	// DestinationMountOptions is serialized into DST_STORAGE_MOUNT_OPTION.
	DestinationMountOptions string
	// DestinationMountName is serialized into DST_STORAGE_MOUNT_NAME.
	DestinationMountName string

	// Common contributes the shared copier environment variables first.
	Common CopierCommonArguments
}

// AssembleEnvironment appends sync-specific KEY=VALUE pairs after the shared copier environment.
func (i *SyncerCommonArguments) AssembleEnvironment(inherited []string) []string {
	inherited = i.Common.AssembleEnvironment(inherited)

	envs := make([]string, 0, 8*2)

	envs = append(envs, "REPORT_ENABLED", strconv.FormatBool(i.ReportEnabled))
	envs = append(envs, "SANDBOX_MOUNT_OPTION", i.SandboxMountOption)

	envs = append(envs, "SRC_STORAGE_MOUNT_HOST", i.SourceMountHost)
	envs = append(envs, "SRC_STORAGE_MOUNT_OPTION", i.SourceMountOptions)
	envs = append(envs, "SRC_STORAGE_MOUNT_NAME", i.SourceMountName)

	envs = append(envs, "DST_STORAGE_MOUNT_HOST", i.DestinationMountHost)
	envs = append(envs, "DST_STORAGE_MOUNT_OPTION", i.DestinationMountOptions)
	envs = append(envs, "DST_STORAGE_MOUNT_NAME", i.DestinationMountName)

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
