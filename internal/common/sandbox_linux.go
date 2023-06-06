//go:build linux
// +build linux

package common

import (
	"fmt"
	"github.com/moby/sys/mount"
	"golang.org/x/sys/unix"
	"log"
	"os"
	"reflect"
	"syscall"
	"unsafe"
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

	log.Printf("the process is sandboxed")
	return nil
}

func Mount(info MountInfo, destinationPath string) (err error) {
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

func Umount(destinationPath string) error {
	return mount.Unmount(destinationPath)
}

func RecursiveUmounts(destinationPath string) error {
	return mount.RecursiveUnmount(destinationPath)
}

func Self(sanboxed bool) (string, *syscall.SysProcAttr) {
	path := "/proc/self/exe"
	attr := &syscall.SysProcAttr{
		Pdeathsig: unix.SIGTERM,
	}

	if sanboxed {
		attr.Unshareflags |= syscall.CLONE_NEWNS | syscall.CLONE_NEWUTS | syscall.CLONE_NEWIPC | syscall.CLONE_FS
	}
	return path, attr
}

func SetProcessName(name string) error {
	bytes := append([]byte(name), 0)

	argv0str := (*reflect.StringHeader)(unsafe.Pointer(&os.Args[0]))
	argv0 := (*[1 << 30]byte)(unsafe.Pointer(argv0str.Data))[:argv0str.Len]

	n := copy(argv0, bytes)
	if n < len(argv0) {
		argv0[n] = 0
	}

	ptr := unsafe.Pointer(&bytes[0])
	if _, _, errno := syscall.RawSyscall6(syscall.SYS_PRCTL, syscall.PR_SET_NAME, uintptr(ptr), 0, 0, 0, 0); errno != 0 {
		return errno
	}
	return nil
}
