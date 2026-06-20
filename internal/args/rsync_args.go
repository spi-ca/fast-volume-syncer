// Package args assembles environment and command arguments for sync and copy workers.
package args

import (
	"fmt"
	"strconv"
)

// RsyncArgs describes the optional flags appended to each rsync invocation.
type RsyncArgs struct {
	// Verbose enables rsync progress and transfer statistics.
	Verbose bool
	// Delete removes destination entries that no longer exist in the source.
	Delete bool
	// PreservePermission keeps source permission bits on copied entries.
	PreservePermission bool
	// PreserveOwnership keeps source owner and group metadata when possible.
	PreserveOwnership bool
	// CopySpecial allows device files and other special files to be copied.
	CopySpecial bool
	// Compress enables rsync stream compression.
	Compress bool
	// WholeFile disables delta transfer optimization.
	WholeFile bool
	// Inplace writes updates directly into the destination file.
	Inplace bool
	// Recursive traverses directory trees instead of reading file lists from stdin.
	Recursive bool
	// Port selects a non-default rsync daemon or remote-shell port.
	Port int
	// BandwidthLimit passes rsync's --bwlimit value through unchanged.
	BandwidthLimit string
}

// Assemble returns the full rsync argument vector, including src and dst as the final arguments.
func (a RsyncArgs) Assemble(src, dst string) []string {
	args := []string{
		//"--links",
		//"--safe-links",
		//"--omit-link-times",
		"--no-links",
		"--times",
		"--one-file-system",
		"--omit-dir-times",
		"--human-readable",
		"--protect-args",
		"--timeout=0",
		"--contimeout=0",
	}

	if a.Verbose {
		args = append(args, "--stats")
		args = append(args, "--verbose")
		args = append(args, "--progress")

	} else {
		args = append(args, "--info=NAME2")

	}

	if a.Delete {
		args = append(args, "--delete")
		args = append(args, "--delete-during")
	}

	if a.PreservePermission {
		args = append(args, "--perms")

	} else {
		args = append(args, "--no-perms")

	}

	if a.PreserveOwnership {
		args = append(args, "--owner")
		args = append(args, "--group")
	} else {
		args = append(args, "--no-owner")
		args = append(args, "--no-group")
	}

	if a.CopySpecial {
		args = append(args, "--devices")
		args = append(args, "--specials")
	} else {
		args = append(args, "--no-devices")
		args = append(args, "--no-specials")
	}

	if a.Compress {
		args = append(args, "--compress")

	} else {
		args = append(args, "--no-compress")
	}

	if a.WholeFile {
		args = append(args, "--whole-file")

	} else {
		args = append(args, "--no-whole-file")
	}

	if a.Inplace {
		args = append(args, "--inplace")

	} else {
		args = append(args, "--no-inplace")
	}

	if a.Recursive {
		args = append(args, "--recursive")
	} else {
		args = append(args, "--no-recursive")
		args = append(args, "--files-from=-")
	}

	if a.Port > 0 {
		args = append(args, "--port", strconv.Itoa(a.Port))
	}

	if len(a.BandwidthLimit) > 0 {
		args = append(args, fmt.Sprint("--bwlimit=", a.BandwidthLimit))
	}

	args = append(args, src)
	args = append(args, dst)
	return args
}
