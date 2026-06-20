//go:build darwin && !cgo
// +build darwin,!cgo

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"os"
	"runtime"
)

// PathFromFd reports that descriptor path resolution needs cgo on this target.
func PathFromFd(_ uintptr) (string, os.FileInfo, error) {
	return "", nil, fmt.Errorf("this os(%s) requires cgo for fd path resolution", runtime.GOOS)
}
