//go:build !linux
// +build !linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"os"
	"runtime"
)

// ChmodFD reports that fd-anchored chmod is unavailable on this target.
func ChmodFD(_ uintptr, _ os.FileMode) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// OpenDirBeneath reports that fd-anchored path opening is unavailable on this target.
func OpenDirBeneath(_, _ string, _ bool) (*os.File, error) {
	return nil, fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
