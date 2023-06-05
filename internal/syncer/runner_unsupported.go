//go:build !linux
// +build !linux

package syncer

import (
	"fmt"
	"runtime"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

func (r *Runner) sandbox() error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func (r *Runner) mount(_ common.MountInfo, _ string) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func (r *Runner) umount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}

func (r *Runner) recursiveUmount(_ string) error {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
