//go:build !linux && !darwin
// +build !linux,!darwin

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"os"
	"runtime"
)

// PathFromFd reports that fd-to-path resolution is unavailable on this target.
func PathFromFd(fd uintptr) (string, os.FileInfo, error) {
	return "", nil, fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
