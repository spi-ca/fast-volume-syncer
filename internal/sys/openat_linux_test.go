//go:build linux
// +build linux

// Package sys wraps platform-specific process, mount, descriptor, and mode helpers.
package sys

import (
	"os"
	"path/filepath"
	"testing"
)

// TestOpenDirBeneathRejectsSymlinkEscape verifies fd-anchored subpath opening rejects symlink traversal.
func TestOpenDirBeneathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "safe"), 0o755); err != nil {
		t.Fatalf("make safe dir: %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("make symlink: %v", err)
	}

	file, err := OpenDirBeneath(root, "safe", false)
	if err != nil {
		t.Fatalf("OpenDirBeneath safe dir: %v", err)
	}
	file.Close()

	if file, err := OpenDirBeneath(root, filepath.Join("linked", "child"), false); err == nil {
		file.Close()
		t.Fatal("OpenDirBeneath followed a symlink component")
	}
	if file, err := OpenDirBeneath(root, filepath.Join("..", filepath.Base(outside)), false); err == nil {
		file.Close()
		t.Fatal("OpenDirBeneath allowed parent traversal")
	}
}

// TestOpenDirBeneathCreatesDestination verifies destination subpaths can be created below the root.
func TestOpenDirBeneathCreatesDestination(t *testing.T) {
	root := t.TempDir()
	file, err := OpenDirBeneath(root, filepath.Join("new", "nested"), true)
	if err != nil {
		t.Fatalf("OpenDirBeneath create: %v", err)
	}
	if err := ChmodFD(file.Fd(), 0o750); err != nil {
		file.Close()
		t.Fatalf("ChmodFD created destination: %v", err)
	}
	file.Close()
	if info, err := os.Stat(filepath.Join(root, "new", "nested")); err != nil || !info.IsDir() {
		t.Fatalf("created directory stat err=%v info=%v", err, info)
	} else if got, want := info.Mode().Perm(), os.FileMode(0o750); got != want {
		t.Fatalf("created directory mode = %v, want %v", got, want)
	}
}
