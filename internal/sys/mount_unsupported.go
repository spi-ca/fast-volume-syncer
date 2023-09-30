//go:build !linux
// +build !linux

package sys

import (
	"fmt"
	"runtime"
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
