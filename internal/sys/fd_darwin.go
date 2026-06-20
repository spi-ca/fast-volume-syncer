//go:build darwin && cgo
// +build darwin,cgo

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

/*
#define __DARWIN_UNIX03 0
#define KERNEL
#define _DARWIN_USE_64_BIT_INODE
#include <dirent.h>
#include <fcntl.h>
#include <sys/param.h>
*/
import "C"

import (
	"os"
	"syscall"
	"unsafe"
)

// PathFromFd resolves an open descriptor to its current path with F_GETPATH and returns fresh file info.
func PathFromFd(fd uintptr) (string, os.FileInfo, error) {

	var (
		fi   os.FileInfo
		name string
		err  error
	)

	buf := make([]C.char, int(C.MAXPATHLEN)+1)
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, fd, syscall.F_GETPATH, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return "", nil, errno
	}

	name = C.GoString(&buf[0])

	if fi, err = os.Lstat(name); err != nil {
		return "", nil, err
	}

	return name, fi, nil
}
