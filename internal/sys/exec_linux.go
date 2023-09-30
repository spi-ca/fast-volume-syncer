//go:build linux
// +build linux

package sys

import (
	"syscall"
)

func ApplySysProAttrIsolation(attr *syscall.SysProcAttr) error {
	attr.Unshareflags |= syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_FS
	return nil
}

func ApplySysProAttrPGid(attr *syscall.SysProcAttr) error {
	attr.Setpgid = true
	return nil
}

func ApplySysProAttrSid(attr *syscall.SysProcAttr) error {
	attr.Setsid = true
	return nil
}

func ApplySysProAttrPdeathsig(attr *syscall.SysProcAttr, pdeathsig syscall.Signal) error {
	attr.Pdeathsig = pdeathsig
	return nil
}
