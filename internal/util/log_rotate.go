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
	logRotateLock sync.Mutex
)

func RotateLogs() {
	rotateLogsInternal(time.Now())
}

func StartRotateLogMidnight(ctx context.Context) {
	// Position the first execution
	first, duration := getNextMidnight()
	offset := first.Sub(time.Now())
	firstC := time.After(offset)

	// Receiving from a nil channel blocks forever
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

func getNextMidnight() (time.Time, time.Duration) {
	const Day = 24 * time.Hour
	now := time.Now()
	_, dif := now.Zone()
	return now.Truncate(Day).Add(-time.Duration(dif) * time.Second).Add(Day), Day
}

func rotateLogsInternal(at time.Time) {
	logRotateLock.Lock()
	defer logRotateLock.Unlock()

	// 로그파일명 충돌을 막기 위하여 1초를 기다린다.
	delay := time.After(time.Second)
	defer func() { <-delay }()

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stdoutFilePath, stdoutInfo, stdoutErr := sys.PathFromFd(os.Stdout.Fd())
	stderrFilePath, stderrInfo, stderrErr := sys.PathFromFd(os.Stderr.Fd())
	if stdoutErr != nil ||
		stderrErr != nil {
		ErrLog.Printf("failed to get stdout|stderr fileinfo :%w", errors.Join(stdoutErr, stderrErr))
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
			ErrLog.Printf("failed to rotate stdout :%w", stdoutFilePath, err)
			return
		}

		newStdErrFd, err := syscall.Dup(stdoutFd)
		if err != nil {
			ErrLog.Printf("failed to dup for stderr:%v", err)
			return
		}

		_ = os.Stderr.Sync()
		_ = sys.ReplaceFD(newStdErrFd, stderrFd)

		InfoLog.Printf("log file rotated! previous log saved to %s", savedFilename)
	} else {
		if pos, _ := syscall.Seek(stdoutFd, 0, io.SeekCurrent); pos == 0 {
			// do nothing
		} else if savedFilename, err := rotateFile(stdoutFilePath, at, stdoutFd); err != nil {
			InfoLog.Printf("failed to rotate stdout :%w", stdoutFilePath, err)
		} else {
			InfoLog.Printf("stdout rotated! previous log saved to %s", savedFilename)
		}

		if pos, _ := syscall.Seek(stderrFd, 0, io.SeekCurrent); pos == 0 {
			// do nothing
		} else if savedFilename, err := rotateFile(stderrFilePath, at, stderrFd); err != nil {
			ErrLog.Printf("failed to rotate stderr :%w", stderrFilePath, err)
		} else {
			ErrLog.Printf("stderr rotated! previous log saved to %s", savedFilename)
		}
	}
}

func checkLogFile() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	stdoutFilePath, stdoutInfo, stdoutErr := sys.PathFromFd(os.Stdout.Fd())
	stderrFilePath, stderrInfo, stderrErr := sys.PathFromFd(os.Stdout.Fd())
	if stdoutErr != nil ||
		stderrErr != nil {
		ErrLog.Printf("failed to get stdout|stderr fileinfo :%w", errors.Join(stdoutErr, stderrErr))
		return
	} else if !stdoutInfo.Mode().IsRegular() ||
		!stderrInfo.Mode().IsRegular() {
		return
	}

	if stdoutFilePath == stderrFilePath {
		InfoLog.Printf("logging is stored in a file %s", stdoutFilePath)
	} else {
		InfoLog.Printf("stdout logging is stored in a file %s", stdoutFilePath)
		InfoLog.Printf("stderr logging is stored in a file %s", stderrFilePath)
	}
}

func rotateFile(filename string, at time.Time, fd int) (string, error) {
	ext := filepath.Ext(filename)
	rotateFilename := fmt.Sprintf("%s_%s%s", filename[:len(filename)-len(ext)], at.Format("2006-01-02-15:04:05"), ext)
	err := os.Rename(filename, rotateFilename)
	if err != nil {
		return "", fmt.Errorf("failed to rename a file(%s) :%w", filename, err)
	}

	newFd, err := syscall.Open(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND|syscall.O_CLOEXEC, 0o644)
	if err != nil {
		_ = os.Rename(rotateFilename, filename)
		return "", fmt.Errorf("failed to rename a file(%s) :%w", filename, err)
	}

	err = syscall.Fsync(fd)
	if err != nil {
		_ = os.Rename(rotateFilename, filename)
		_ = syscall.Close(newFd)
		return "", fmt.Errorf("failed to sync for fd(%d) :%w", fd, err)
	}

	err = sys.ReplaceFD(newFd, fd)
	if err != nil {
		_ = os.Rename(rotateFilename, filename)
		_ = syscall.Close(newFd)
		return "", fmt.Errorf("failed to dup(2)(%d,%d) :%w", newFd, fd, err)
	}

	return rotateFilename, nil
}
