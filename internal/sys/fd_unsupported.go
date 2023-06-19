//go:build !linux && !darwin
// +build !linux,!darwin

package sys

import (
	"os"
	"runtime"
)

func PathFromFd(fd int) (string, os.FileInfo, error) {
	return "", nil, fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
