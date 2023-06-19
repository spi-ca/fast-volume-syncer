//go:build !linux && !darwin
// +build !linux,!darwin

package sys

import (
	"fmt"
	"runtime"
	"syscall"
)

func ApplySysProAttrIsolation(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func ApplySysProAttrPGid(attr *syscall.SysProcAttr) error {
	attr.Setpgid = true
	return nil
}

func ApplySysProAttrSid(attr *syscall.SysProcAttr) error {
	attr.Setsid = true
	return nil
}
func ApplySysProAttrPdeathsig(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
