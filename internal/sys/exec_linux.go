//go:build linux
// +build linux

package sys

import (
	"syscall"
)

func ApplySysProc(attr *syscall.SysProcAttr, isolate bool, pgid bool, sid bool, pdeathsig syscall.Signal) error {
	if isolate {
		attr.Unshareflags |= syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_FS
	}
	if sid {
		attr.Setsid = true
	}
	if pgid {
		attr.Setpgid = true
	}
	if pdeathsig != 0 {
		attr.Pdeathsig = pdeathsig
	}

	return nil
}
