//go:build linux
// +build linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"syscall"
)

// ApplySysProAttrIsolation enables the mount, UTS, IPC, and FS namespace isolation used by sandboxed workers.
func ApplySysProAttrIsolation(attr *syscall.SysProcAttr) error {
	attr.Unshareflags |= syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_FS
	return nil
}

// ApplySysProAttrPGid places the child in a separate process group.
func ApplySysProAttrPGid(attr *syscall.SysProcAttr) error {
	attr.Setpgid = true
	return nil
}

// ApplySysProAttrSid starts the child in a new session.
func ApplySysProAttrSid(attr *syscall.SysProcAttr) error {
	attr.Setsid = true
	return nil
}

// ApplySysProAttrPdeathsig configures the signal delivered when the parent process exits.
func ApplySysProAttrPdeathsig(attr *syscall.SysProcAttr, pdeathsig syscall.Signal) error {
	attr.Pdeathsig = pdeathsig
	return nil
}
