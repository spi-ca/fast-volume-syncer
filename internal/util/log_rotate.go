//go:build !windows
// +build !windows

// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/sys"
)

var (
	// logRotateLock prevents concurrent descriptor rotation.
	logRotateLock sync.Mutex
)

// RotateLogs forces an immediate log-rotation pass using the current wall clock.
func RotateLogs() {
	rotateLogsInternal(time.Now())
}

// StartRotateLogMidnight waits for the next local midnight, then rotates once per day until ctx is canceled.
func StartRotateLogMidnight(ctx context.Context) {
	// Position the first execution.
	first, duration := getNextMidnight()
	offset := first.Sub(time.Now())
	firstC := time.After(offset)

	// Receiving from a nil channel blocks forever.
	t := &time.Ticker{C: nil}
	for {
		select {
		case v := <-firstC:
			t = time.NewTicker(duration)
			rotateLogsInternal(v)
		case v := <-t.C:
			rotateLogsInternal(v)
		case <-ctx.Done():
			t.Stop()
			return
		}
	}
}

// getNextMidnight returns the next local midnight and the fixed 24-hour interval used after the first rotation.
func getNextMidnight() (time.Time, time.Duration) {
	const Day = 24 * time.Hour
	now := time.Now()
	_, dif := now.Zone()
	return now.Truncate(Day).Add(-time.Duration(dif) * time.Second).Add(Day), Day
}

// rotateLogsInternal serializes rotation, swaps live stdio file descriptors, and handles shared stdout/stderr logs.
func rotateLogsInternal(at time.Time) {
	logRotateLock.Lock()
	defer logRotateLock.Unlock()

	// Wait one second before returning so the next rotated filename cannot collide on the same timestamp.
	delay := time.After(time.Second)
	defer func() { <-delay }()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stdoutFilePath, stdoutInfo, stdoutErr := sys.PathFromFd(os.Stdout.Fd())
	stderrFilePath, stderrInfo, stderrErr := sys.PathFromFd(os.Stderr.Fd())
	if stdoutErr != nil ||
		stderrErr != nil {
		ErrLog.Printf("failed to get stdout|stderr fileinfo :%v", errors.Join(stdoutErr, stderrErr))
		return
	} else if !stdoutInfo.Mode().IsRegular() ||
		!stderrInfo.Mode().IsRegular() {
		return
	}

	stdoutFd := int(os.Stdout.Fd())
	stderrFd := int(os.Stderr.Fd())

	if stdoutFilePath == stderrFilePath {
		if pos, _ := syscall.Seek(stdoutFd, 0, io.SeekCurrent); pos == 0 {
			// do nothing
			return
		}
		savedFilename, err := rotateFile(stdoutFilePath, at, stdoutFd)
		if err != nil {
			ErrLog.Printf("failed to rotate stdout %s: %v", stdoutFilePath, err)
			return
		}

		newStdErrFd, err := syscall.Dup(stdoutFd)
		if err != nil {
			ErrLog.Printf("failed to dup for stderr: %v", err)
			return
		}

		_ = os.Stderr.Sync()
		if err := sys.ReplaceFD(newStdErrFd, stderrFd); err != nil {
			_ = syscall.Close(newStdErrFd)
			ErrLog.Printf("failed to replace stderr fd: %v", err)
			return
		}
		_ = syscall.Close(newStdErrFd)

		InfoLog.Printf("log file rotated! previous log saved to %s", savedFilename)
	} else {
		if pos, _ := syscall.Seek(stdoutFd, 0, io.SeekCurrent); pos == 0 {
			// do nothing
		} else if savedFilename, err := rotateFile(stdoutFilePath, at, stdoutFd); err != nil {
			InfoLog.Printf("failed to rotate stdout: %v", err)
		} else {
			InfoLog.Printf("stdout rotated! previous log saved to %s", savedFilename)
		}

		if pos, _ := syscall.Seek(stderrFd, 0, io.SeekCurrent); pos == 0 {
			// do nothing
		} else if savedFilename, err := rotateFile(stderrFilePath, at, stderrFd); err != nil {
			ErrLog.Printf("failed to rotate stderr: %v", err)
		} else {
			ErrLog.Printf("stderr rotated! previous log saved to %s", savedFilename)
		}
	}
}

// CheckLogFile reports the regular-file destinations currently backing log output, if any.
func CheckLogFile() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stdoutFilePath, stdoutInfo, stdoutErr := sys.PathFromFd(os.Stdout.Fd())
	stderrFilePath, stderrInfo, stderrErr := sys.PathFromFd(os.Stderr.Fd())
	if stdoutErr != nil ||
		stderrErr != nil {
		return
	} else if !stdoutInfo.Mode().IsRegular() ||
		!stderrInfo.Mode().IsRegular() {
		return
	}

	if stdoutFilePath == stderrFilePath {
		InfoLog.Printf("logging is stored in a file %s", stdoutFilePath)
	} else {
		InfoLog.Printf("stdout logging is stored in a file %s", stdoutFilePath)
		ErrLog.Printf("stderr logging is stored in a file %s", stderrFilePath)
	}
}

// rotateFile renames the active log, opens a replacement, fsyncs the old fd, and dup-replaces it in place.
func rotateFile(filename string, at time.Time, fd int) (string, error) {
	ext := filepath.Ext(filename)
	rotateFilename := fmt.Sprintf("%s_%s%s", filename[:len(filename)-len(ext)], at.Format("2006-01-02-15:04:05"), ext)
	err := os.Rename(filename, rotateFilename)
	if err != nil {
		return "", fmt.Errorf("failed to rename a file(%s) :%w", filename, err)
	}

	newFd, err := syscall.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		_ = os.Rename(rotateFilename, filename)
		return "", fmt.Errorf("failed to reopen log file(%s): %w", filename, err)
	}
	var stat syscall.Stat_t
	if err := syscall.Fstat(newFd, &stat); err != nil || stat.Mode&syscall.S_IFMT != syscall.S_IFREG {
		_ = os.Rename(rotateFilename, filename)
		_ = syscall.Close(newFd)
		return "", fmt.Errorf("replacement log file must be regular(%s): %w", filename, err)
	}

	err = syscall.Fsync(fd)
	if err != nil {
		_ = os.Rename(rotateFilename, filename)
		_ = syscall.Close(newFd)
		return "", fmt.Errorf("failed to sync for fd(%d): %w", fd, err)
	}

	err = sys.ReplaceFD(newFd, fd)
	if err != nil {
		_ = os.Rename(rotateFilename, filename)
		_ = syscall.Close(newFd)
		return "", fmt.Errorf("failed to dup(2)(%d,%d): %w", newFd, fd, err)
	}
	_ = syscall.Close(newFd)

	return rotateFilename, nil
}
