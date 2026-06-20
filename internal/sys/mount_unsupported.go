//go:build !linux
// +build !linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"runtime"
)

// Sandbox reports that the Linux mount-namespace sandbox is unavailable on this target.
func Sandbox(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// Mount reports that filesystem mounts are unsupported on this target.
func Mount(_ string, _ string, _ string, _ string) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// BindMountFD reports that fd-backed bind mounts are unsupported on this target.
func BindMountFD(_ uintptr, _ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// Umount reports that filesystem unmounts are unsupported on this target.
func Umount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

// RecursiveUmounts reports that recursive unmounts are unsupported on this target.
func RecursiveUmounts(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
