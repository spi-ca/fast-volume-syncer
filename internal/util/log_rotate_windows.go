//go:build windows
// +build windows

// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import "context"

// RotateLogs is a no-op on Windows because descriptor rotation is unsupported.
func RotateLogs() {}

// StartRotateLogMidnight waits for cancellation without rotating logs on Windows.
func StartRotateLogMidnight(ctx context.Context) {
	<-ctx.Done()
}

// CheckLogFile is a no-op on Windows because descriptor path resolution is unsupported.
func CheckLogFile() {}
