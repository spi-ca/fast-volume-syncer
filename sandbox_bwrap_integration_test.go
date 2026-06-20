//go:build integration && linux
// +build integration,linux

// Package main contains Linux integration tests for the fast-volume-syncer binary.
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestBwrapCopyE2E builds the CLI and verifies copy behavior inside a bwrap sandbox.
func TestBwrapCopyE2E(t *testing.T) {
	bwrapPath, err := exec.LookPath("bwrap")
	if err != nil {
		t.Skip("bwrap is not installed")
	}

	probe := exec.Command(bwrapPath, "--ro-bind", "/", "/", "true")
	if out, err := probe.CombinedOutput(); err != nil {
		t.Skipf("bwrap is not usable in this environment: %v\n%s", err, out)
	}

	tmp := t.TempDir()
	binPath := filepath.Join(tmp, "fast-volume-syncer")
	build := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fast-volume-syncer: %v\n%s", err, out)
	}

	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	if err := os.MkdirAll(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("make source fixture: %v", err)
	}
	payload := []byte("sandbox e2e payload\n")
	if err := os.WriteFile(filepath.Join(src, "nested", "file.txt"), payload, 0o644); err != nil {
		t.Fatalf("write source fixture: %v", err)
	}
	if err := os.Symlink("nested/file.txt", filepath.Join(src, "link.txt")); err != nil {
		t.Fatalf("write source symlink: %v", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("make destination fixture: %v", err)
	}

	cmd := exec.Command(
		bwrapPath,
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--bind", src, src,
		"--bind", dst, dst,
		"--chdir", "/",
		binPath,
		"--scan-find-path", "",
		"--task-size", "1",
		"--chunk-size", "2",
		"--retry-attempts", "0",
		"copy", src, dst,
	)
	cmd.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bwrap copy e2e failed: %v\n%s", err, out)
	}

	copied, err := os.ReadFile(filepath.Join(dst, "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if !bytes.Equal(copied, payload) {
		t.Fatalf("copied payload = %q, want %q", copied, payload)
	}
	linkTarget, err := os.Readlink(filepath.Join(dst, "link.txt"))
	if err != nil {
		t.Fatalf("read copied symlink: %v", err)
	}
	if linkTarget != "nested/file.txt" {
		t.Fatalf("copied symlink target = %q, want nested/file.txt", linkTarget)
	}
}
