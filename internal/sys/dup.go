//go:build (!linux || !arm64) && (!linux || !riscv64) && !windows
// +build !linux !arm64
// +build !linux !riscv64
// +build !windows

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import "golang.org/x/sys/unix"

// ReplaceFD remaps newfd onto oldfd with dup2 on platforms that expose it directly.
func ReplaceFD(oldfd int, newfd int) (err error) {
	return unix.Dup2(oldfd, newfd)
}
