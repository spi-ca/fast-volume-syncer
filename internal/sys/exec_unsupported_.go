//go:build !linux
// +build !linux

package sys

import (
	"fmt"
	"runtime"
	"syscall"
)

func ApplySysProc(attr *syscall.SysProcAttr, isolate bool, pgid bool, sid bool, pdeathsig syscall.Signal) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
