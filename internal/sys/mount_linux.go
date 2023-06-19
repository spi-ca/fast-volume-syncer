//go:build linux
// +build linux

package sys

import (
	"fmt"
	"log"
	"os"

	"github.com/moby/sys/mount"
)

func Sandbox(sandboxMountOption string) error {
	err := mount.MakeRPrivate("/")
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

	if len(sandboxMountOption) == 0 {
		err = mount.Mount("tmp", tmpDir, "tmpfs", "nosuid,noexec,nodev")
	} else {
		err = mount.Mount("tmp", tmpDir, "tmpfs", sandboxMountOption)
	}
	if err != nil {
		return fmt.Errorf("failed to mount %s : %w", tmpDir, err)
	}

	log.Print("the process is sandboxed")
	return nil
}

func Mount(source string, destinationPath string, mountType string, mountOptions string) (err error) {
	err = os.Mkdir(destinationPath, 0o755)
	if err != nil {
		return fmt.Errorf("failed to make a directory(%s): %w", destinationPath, err)
	}

	err = mount.Mount(source, destinationPath, mountType, mountOptions)
	if err != nil {
		return fmt.Errorf("failed to mount %s : %w", destinationPath, err)
	}
	return nil
}

func Umount(destinationPath string) error {
	return mount.Unmount(destinationPath)
}

func RecursiveUmounts(destinationPath string) error {
	return mount.RecursiveUnmount(destinationPath)
}
