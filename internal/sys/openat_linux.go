//go:build linux
// +build linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// ChmodFD applies permissions to a file descriptor, including O_PATH descriptors.
func ChmodFD(fd uintptr, mode os.FileMode) error {
	procPath := fmt.Sprintf("/proc/self/fd/%d", fd)
	if err := os.Chmod(procPath, mode.Perm()); err != nil {
		return fmt.Errorf("chmod fd %d: %w", fd, err)
	}
	return nil
}

// OpenDirBeneath opens a directory below root without following symlinks or escaping the root fd.
func OpenDirBeneath(root, subpath string, create bool) (*os.File, error) {
	if subpath == "" {
		subpath = "."
	}
	if !filepath.IsLocal(subpath) {
		return nil, fmt.Errorf("path %q must be local", subpath)
	}
	rootFD, err := unix.Open(root, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open root %q: %w", root, err)
	}
	currentFD := rootFD
	closeCurrent := true
	for _, part := range strings.Split(filepath.Clean(subpath), string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if closeCurrent {
				_ = unix.Close(currentFD)
			}
			return nil, fmt.Errorf("path %q must not contain parent traversal", subpath)
		}
		nextFD, err := openDirAtNoSymlink(currentFD, part)
		if err != nil && create && os.IsNotExist(err) {
			if mkdirErr := unix.Mkdirat(currentFD, part, 0o700); mkdirErr != nil && !os.IsExist(mkdirErr) {
				if closeCurrent {
					_ = unix.Close(currentFD)
				}
				return nil, fmt.Errorf("create %q below %q: %w", part, root, mkdirErr)
			}
			nextFD, err = openDirAtNoSymlink(currentFD, part)
		}
		if err != nil {
			if closeCurrent {
				_ = unix.Close(currentFD)
			}
			return nil, fmt.Errorf("open %q below %q: %w", part, root, err)
		}
		if closeCurrent {
			_ = unix.Close(currentFD)
		}
		currentFD = nextFD
		closeCurrent = true
	}
	return os.NewFile(uintptr(currentFD), filepath.Join(root, filepath.Clean(subpath))), nil
}

// openDirAtNoSymlink opens one child directory without following symlinks, using openat2 when available.
func openDirAtNoSymlink(dirfd int, name string) (int, error) {
	fd, err := unix.Openat2(dirfd, name, &unix.OpenHow{
		Flags:   unix.O_PATH | unix.O_DIRECTORY | unix.O_CLOEXEC,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS,
	})
	if err == nil || !(err == unix.ENOSYS || err == unix.EINVAL) {
		return fd, err
	}
	return unix.Openat(dirfd, name, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
}
