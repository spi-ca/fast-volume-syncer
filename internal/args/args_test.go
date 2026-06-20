// Package args assembles environment and command arguments for sync and copy workers.
package args

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// envMap converts KEY=VALUE environment entries into a map for focused assertions.
func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			m[key] = value
		}
	}
	return m
}

// TestRsyncArgsAssembleDefaults verifies the default rsync flags used when every option is disabled.
func TestRsyncArgsAssembleDefaults(t *testing.T) {
	got := RsyncArgs{}.Assemble("/src", "/dst")
	wantContains := []string{
		"--no-links",
		"--times",
		"--one-file-system",
		"--omit-dir-times",
		"--human-readable",
		"--protect-args",
		"--timeout=0",
		"--contimeout=0",
		"--info=NAME2",
		"--no-perms",
		"--no-owner",
		"--no-group",
		"--no-devices",
		"--no-specials",
		"--no-compress",
		"--no-whole-file",
		"--no-inplace",
		"--no-recursive",
		"--files-from=-",
		"/src",
		"/dst",
	}
	for _, want := range wantContains {
		if !contains(got, want) {
			t.Fatalf("assembled rsync args missing %q: %#v", want, got)
		}
	}
	if got[len(got)-2] != "/src" || got[len(got)-1] != "/dst" {
		t.Fatalf("expected src/dst to be final args, got %#v", got)
	}
}

// TestRsyncArgsAssembleEnabledOptions verifies that enabled rsync settings produce the matching flags.
func TestRsyncArgsAssembleEnabledOptions(t *testing.T) {
	got := RsyncArgs{
		Verbose:            true,
		Delete:             true,
		PreservePermission: true,
		PreserveOwnership:  true,
		CopySpecial:        true,
		Compress:           true,
		WholeFile:          true,
		Inplace:            true,
		Recursive:          true,
		Port:               873,
		BandwidthLimit:     "10M",
	}.Assemble("src", "dst")

	for _, want := range []string{"--stats", "--verbose", "--progress", "--delete", "--delete-during", "--perms", "--owner", "--group", "--devices", "--specials", "--compress", "--whole-file", "--inplace", "--recursive", "--port", "873", "--bwlimit=10M"} {
		if !contains(got, want) {
			t.Fatalf("assembled rsync args missing %q: %#v", want, got)
		}
	}
	for _, unexpected := range []string{"--files-from=-", "--no-recursive", "--info=NAME2"} {
		if contains(got, unexpected) {
			t.Fatalf("assembled rsync args unexpectedly contain %q: %#v", unexpected, got)
		}
	}
}

// TestCopierCommonArgumentsAssembleEnvironment verifies the full copier environment export, including rsync and retry keys.
func TestCopierCommonArgumentsAssembleEnvironment(t *testing.T) {
	args := CopierCommonArguments{
		FileMode:         os.FileMode(0o640),
		LogPrefix:        "worker-a",
		UseRsync:         true,
		ScanDuration:     3 * time.Second,
		FinderBinaryPath: "find",
		TaskSize:         4,
		ChunkSize:        99,
		Args: RsyncArgs{
			Verbose:            true,
			Delete:             true,
			PreservePermission: true,
			PreserveOwnership:  true,
			CopySpecial:        true,
			Compress:           true,
			WholeFile:          true,
			Inplace:            true,
			Recursive:          true,
			Port:               873,
			BandwidthLimit:     "10M",
		},
		Retry: RetryArgs{
			Attempts:  7,
			Delay:     time.Second,
			MaxDelay:  5 * time.Second,
			MaxJitter: 250 * time.Millisecond,
		},
	}

	got := envMap(args.AssembleEnvironment([]string{"KEEP=1"}))
	want := map[string]string{
		"KEEP":                  "1",
		"FILE_MODE":             "-rw-r-----",
		"LOG_PREFIX":            "worker-a",
		"RSYNC_ENABLED":         "true",
		"RSYNC_VERBOSE":         "true",
		"RSYNC_DELETE":          "true",
		"RSYNC_PERMS":           "true",
		"RSYNC_OWNER":           "true",
		"RSYNC_SPECIAL":         "true",
		"RSYNC_COMPRESS":        "true",
		"RSYNC_WHOLE_FILE":      "true",
		"RSYNC_INPLACE":         "true",
		"RSYNC_RECURSIVE":       "true",
		"RSYNC_PORT":            "873",
		"RSYNC_BANDWIDTH_LIMIT": "10M",
		"SCAN_DEADLINE":         "3s",
		"SCAN_FIND_PATH":        "find",
		"TASK_SIZE":             "4",
		"CHUNK_SIZE":            "99",
		"RETRY_ATTEMPTS":        "7",
		"RETRY_DELAY":           "1s",
		"RETRY_MAX_DELAY":       "5s",
		"RETRY_MAX_JITTER":      "250ms",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected environment\nwant: %#v\n got: %#v", want, got)
	}
}

// TestSyncerCommonArgumentsAssembleEnvironmentIncludesCopierAndMountConfig verifies sync-only mount keys are added after shared copier settings.
func TestSyncerCommonArgumentsAssembleEnvironmentIncludesCopierAndMountConfig(t *testing.T) {
	args := SyncerCommonArguments{
		ReportEnabled:           true,
		SandboxMountOption:      "size=1M",
		SourceMountHost:         "192.0.2.10",
		SourceMountOptions:      "ro",
		SourceMountName:         "src",
		DestinationMountHost:    "192.0.2.11",
		DestinationMountOptions: "rw",
		DestinationMountName:    "dst",
		Common: CopierCommonArguments{
			FileMode:         os.FileMode(0o600),
			FinderBinaryPath: "find",
		},
	}

	got := envMap(args.AssembleEnvironment(nil))
	for key, want := range map[string]string{
		"FILE_MODE":                "-rw-------",
		"REPORT_ENABLED":           "true",
		"SANDBOX_MOUNT_OPTION":     "size=1M",
		"SRC_STORAGE_MOUNT_HOST":   "192.0.2.10",
		"SRC_STORAGE_MOUNT_OPTION": "ro",
		"SRC_STORAGE_MOUNT_NAME":   "src",
		"DST_STORAGE_MOUNT_HOST":   "192.0.2.11",
		"DST_STORAGE_MOUNT_OPTION": "rw",
		"DST_STORAGE_MOUNT_NAME":   "dst",
	} {
		if got[key] != want {
			t.Fatalf("expected %s=%q, got %q in %#v", key, want, got[key], got)
		}
	}
}

// contains reports whether want appears in values.
func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
