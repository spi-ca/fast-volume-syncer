//go:build darwin
// +build darwin

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"runtime"
	"syscall"
)

// ApplySysProAttrIsolation reports that the Linux namespace sandbox path is unavailable on Darwin.
func ApplySysProAttrIsolation(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
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

// ApplySysProAttrPdeathsig is a no-op because Darwin exposes no parent-death signal knob here.
func ApplySysProAttrPdeathsig(attr *syscall.SysProcAttr, pdeathsig syscall.Signal) error {
	// fmt.Errorf("this os(%s) not supported", runtime.GOOS)
	return nil
}
