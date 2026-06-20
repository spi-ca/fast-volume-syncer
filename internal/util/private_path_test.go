// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEnsurePrivatePathPrefixUsesWorkingDirectory checks relative paths are resolved from the current directory.
func TestEnsurePrivatePathPrefixUsesWorkingDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	privateRoot := t.TempDir()
	if err := os.Chdir(privateRoot); err != nil {
		t.Fatalf("chdir private root: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	if err := EnsurePrivatePathPrefix(filepath.Join("relative", "dst")); err != nil {
		t.Fatalf("relative private path rejected: %v", err)
	}
}

// TestEnsurePrivatePathPrefixRejectsMissingTailBelowPublicWritableDir checks races under public dirs are rejected.
func TestEnsurePrivatePathPrefixRejectsMissingTailBelowPublicWritableDir(t *testing.T) {
	publicRoot := filepath.Join(t.TempDir(), "public")
	if err := os.Mkdir(publicRoot, 0o777); err != nil {
		t.Fatalf("create public root: %v", err)
	}
	if err := os.Chmod(publicRoot, 0o777); err != nil {
		t.Fatalf("make public root writable: %v", err)
	}
	if err := EnsurePrivatePathPrefix(filepath.Join(publicRoot, "missing")); err == nil {
		t.Fatal("missing tail below public writable directory was accepted")
	}
}
