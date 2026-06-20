// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"os"
)

var (
	selfExecutablePath string
)

// init caches the current executable path once so re-exec callers can reuse an absolute binary path.
func init() {
	if exePath, err := os.Executable(); err != nil {
		panic(fmt.Errorf("failed to get self-path: %w", err))
	} else {
		selfExecutablePath = exePath
	}
}

// Executable returns the absolute path captured during package initialization.
func Executable() string {
	return selfExecutablePath
}
