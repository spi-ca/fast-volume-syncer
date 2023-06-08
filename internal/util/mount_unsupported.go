//go:build !linux
// +build !linux

package util

import (
	"fmt"
	"runtime"
	"syscall"
)

func Sandbox(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func Mount(_ string, _ string, _ string, _ string) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func Umount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func RecursiveUmount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func IsolateMountNamespaceFlags(attr *syscall.SysProcAttr) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
