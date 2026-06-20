//go:build windows
// +build windows

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"runtime"
	"syscall"
)

// ApplySysProAttrIsolation reports that namespace-style sandboxing is unsupported on this target.
func ApplySysProAttrIsolation(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// ApplySysProAttrPGid reports that process groups are unsupported by this target shim.
func ApplySysProAttrPGid(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// ApplySysProAttrSid reports that sessions are unsupported by this target shim.
func ApplySysProAttrSid(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// ApplySysProAttrPdeathsig reports that parent-death signals are unsupported on this target.
func ApplySysProAttrPdeathsig(attr *syscall.SysProcAttr, pdeathsig syscall.Signal) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
