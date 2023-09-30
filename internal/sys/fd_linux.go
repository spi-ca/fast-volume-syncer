//go:build linux
// +build linux

package sys

import (
	"fmt"
	"os"
	"syscall"
)

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
