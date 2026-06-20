//go:build linux
// +build linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"os"
	"syscall"
)

// PathFromFd resolves /proc/self/fd/N, follows the current symlink target, and returns its file info.
func PathFromFd(fd uintptr) (string, os.FileInfo, error) {

	path := fmt.Sprintf("/proc/self/fd/%d", fd)

	var (
		lnkfi os.FileInfo
		fi    os.FileInfo
		n     int
		name  string
		err   error
	)
	if lnkfi, err = os.Lstat(path); err != nil {
		return "", nil, err
	}

	buf := make([]byte, lnkfi.Size()+1)
	if n, err = syscall.Readlink(path, buf); err == nil {
		name = string(buf[:n])
	} else {
		return "", nil, err
	}

	if fi, err = os.Lstat(name); err != nil {
		return "", nil, err
	}

	return name, fi, nil
}
