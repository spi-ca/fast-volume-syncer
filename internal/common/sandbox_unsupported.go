//go:build !linux
// +build !linux

package common

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
)

func Sandbox(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func Mount(_ MountInfo, _ string) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func Umount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func RecursiveUmount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func Self(_ bool) (string, *syscall.SysProcAttr) {
	name := os.Args[0]
	if filepath.Base(name) == name {
		if lp, err := exec.LookPath(name); err == nil {
			return lp, nil
		}
	}
	// handle conversion of relative paths to absolute
	if absName, err := filepath.Abs(name); err == nil {
		return absName, nil
	}
	// if we couldn't get absolute name, return original
	// (NOTE: Go only errors on Abs() if os.Getwd fails)
	return name, nil
}
func SetProcessName(name string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
