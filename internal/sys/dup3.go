//go:build linux && (riscv64 || arm64)
// +build linux
// +build riscv64 arm64

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import "syscall"

// ReplaceFD remaps newfd onto oldfd with dup3 on Linux architectures that do not provide dup2.
func ReplaceFD(oldfd int, newfd int) (err error) {
	// linux_arm64 platform doesn't have syscall.Dup2
	// so use the nearly identical syscall.Dup3 instead.
	return syscall.Dup3(oldfd, newfd, 0)
}
