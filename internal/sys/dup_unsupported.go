//go:build !unix || windows
// +build !unix windows

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"runtime"
)

// ReplaceFD reports that descriptor replacement is unavailable on this target.
func ReplaceFD(oldfd int, newfd int) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
