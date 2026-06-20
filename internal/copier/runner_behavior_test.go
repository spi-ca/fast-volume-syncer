// Package copier batches scanned entries and sends them to the selected copy backend.
package copier

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRunnerPrepareDirectoryRejectsDestinationSymlink checks direct copy roots cannot be symlink aliases.
func TestRunnerPrepareDirectoryRejectsDestinationSymlink(t *testing.T) {
	src := t.TempDir()
	target := t.TempDir()
	dstLink := filepath.Join(t.TempDir(), "dst")
	if err := os.Symlink(target, dstLink); err != nil {
		t.Fatalf("create destination symlink: %v", err)
	}

	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatalf("set target mode: %v", err)
	}
	if _, _, err := (&Runner{FileMode: 0o600}).prepareDirectory(src, dstLink); err == nil {
		t.Fatal("prepareDirectory accepted a symlinked destination root")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target after rejected prepare: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o755); got != want {
		t.Fatalf("symlink target mode mutated to %v, want %v", got, want)
	}
}

// TestRunnerPrepareDirectoryRejectsSourceSymlink checks direct copy sources cannot be symlink aliases.
func TestRunnerPrepareDirectoryRejectsSourceSymlink(t *testing.T) {
	target := t.TempDir()
	srcLink := filepath.Join(t.TempDir(), "src")
	if err := os.Symlink(target, srcLink); err != nil {
		t.Fatalf("create source symlink: %v", err)
	}

	if _, _, err := (&Runner{FileMode: 0o600}).prepareDirectory(srcLink, t.TempDir()); err == nil {
		t.Fatal("prepareDirectory accepted a symlinked source root")
	}
}

// TestRunnerPrepareDirectoryAppliesPrivateMode checks newly prepared direct-copy roots use private permissions.
func TestRunnerPrepareDirectoryAppliesPrivateMode(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dst")

	if _, _, err := (&Runner{FileMode: 0o640}).prepareDirectory(src, dst); err != nil {
		t.Fatalf("prepareDirectory: %v", err)
	}
	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o740); got != want {
		t.Fatalf("destination mode = %v, want %v", got, want)
	}
}
