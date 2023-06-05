//go:build linux
// +build linux

package syncer

import (
	"fmt"
	"github.com/moby/sys/mount"
	"log"
	"os"
	"runtime"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

func (r *Runner) sandbox() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := syscall.Unshare(syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_FS)
	if err != nil {
		return fmt.Errorf("failed to execute 'unshare' call / : %w", err)
	}
	// 여기서부터 namespace 격리.

	err = mount.MakeRPrivate("/")
	if err != nil {
		return fmt.Errorf("failed to make private mount point / : %w", err)
	}

	// 여기서부터 filesystem 격리.

	err = mount.Unmount("/proc")
	if err != nil {
		return fmt.Errorf("failed to umount /proc : %w", err)
	}

	err = mount.Mount("proc", "/proc", "proc", "nosuid,noexec,nodev")
	if err != nil {
		return fmt.Errorf("failed to mount /proc : %w", err)
	}

	tmpDir := os.TempDir()
	err = mount.Unmount(tmpDir)
	if err != nil {
		return fmt.Errorf("failed to umount %s : %w", tmpDir, err)
	}

	if len(r.SandboxMountOption) == 0 {
		err = mount.Mount("tmp", tmpDir, "tmpfs", "nosuid,noexec,nodev")
	} else {
		err = mount.Mount("tmp", tmpDir, "tmpfs", r.SandboxMountOption)
	}
	if err != nil {
		return fmt.Errorf("failed to mount %s : %w", tmpDir, err)
	}

	log.Printf("the process is sandboxed")
	return nil
}

func (r *Runner) mount(info common.MountInfo, destinationPath string) (err error) {
	err = os.Mkdir(destinationPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to make a directory(%s): %w", destinationPath, err)
	}

	err = mount.Mount(info.Source(), destinationPath, info.Type(), info.RefinedOptions())
	if err != nil {
		return fmt.Errorf("failed to mount %s : %w", destinationPath, err)
	}
	return nil
}

func (r *Runner) umount(destinationPath string) error {
	return mount.Unmount(destinationPath)
}

func (r *Runner) recursiveUmount(destinationPath string) error {
	return mount.RecursiveUnmount(destinationPath)
}
