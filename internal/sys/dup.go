//go:build (!linux || !arm64) && (!linux || !riscv64) && !windows
// +build !linux !arm64
// +build !linux !riscv64
// +build !windows

package sys

import "golang.org/x/sys/unix"

func DupFD(oldfd int, newfd int) (err error) {
	return unix.Dup2(oldfd, newfd)
}
