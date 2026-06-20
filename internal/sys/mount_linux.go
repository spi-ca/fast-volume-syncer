//go:build linux
// +build linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"fmt"
	"log"
	"os"

	"github.com/moby/sys/mount"
)

// Sandbox makes the mount tree private, remounts /proc, and replaces the temp dir with tmpfs for sandboxed work.
func Sandbox(sandboxMountOption string) error {
	err := mount.MakeRPrivate("/")
	if err != nil {
		return fmt.Errorf("failed to make private mount point / : %w", err)
	}

	// Filesystem isolation starts after the root mount is made private.
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

// Mount creates destinationPath and mounts source there with the requested filesystem type and options.
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

// BindMountFD bind-mounts an already opened directory fd onto a private workspace mount point.
func BindMountFD(fd uintptr, destinationPath string) error {
	if err := os.Mkdir(destinationPath, 0o755); err != nil {
		return fmt.Errorf("failed to make a directory(%s): %w", destinationPath, err)
	}
	source := fmt.Sprintf("/proc/self/fd/%d", fd)
	if err := mount.Mount(source, destinationPath, "", "bind"); err != nil {
		return fmt.Errorf("failed to bind mount %s to %s: %w", source, destinationPath, err)
	}
	if err := mount.Mount("", destinationPath, "", "remount,bind,nosymfollow"); err != nil {
		_ = mount.Unmount(destinationPath)
		return fmt.Errorf("failed to enforce nosymfollow on bind mount %s: %w", destinationPath, err)
	}
	return nil
}

// Umount unmounts a single mount point created for sync work.
func Umount(destinationPath string) error {
	return mount.Unmount(destinationPath)
}

// RecursiveUmounts recursively tears down a mount point and any nested mounts below it.
func RecursiveUmounts(destinationPath string) error {
	return mount.RecursiveUnmount(destinationPath)
}
