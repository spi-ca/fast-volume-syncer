// Package selector parses copy-entry CSV rows and fans them out to sync workers.
package selector

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAcquirePidFileWritesCurrentPidAndCleanupRemovesFile checks pid publication and cleanup.
func TestAcquirePidFileWritesCurrentPidAndCleanupRemovesFile(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "run", "fast-volume-syncer.pid")
	cleanup, err := AcquirePidFile(pidPath)
	if err != nil {
		t.Fatalf("AcquirePidFile() error = %v", err)
	}

	pid, err := ReadPidFile(pidPath)
	if err != nil {
		t.Fatalf("ReadPidFile() error = %v", err)
	}
	if pid != os.Getpid() {
		t.Fatalf("ReadPidFile() = %d, want current pid %d", pid, os.Getpid())
	}

	cleanup()
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("expected pid file removed after cleanup, stat err=%v", err)
	}
}

// TestPidFileRejectsSymlink checks that daemon pid files cannot follow caller-controlled links.
func TestPidFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	pidPath := filepath.Join(dir, "fast-volume-syncer.pid")
	targetPath := filepath.Join(dir, "target.pid")
	if err := os.WriteFile(targetPath, []byte("1\n"), 0o600); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}
	if err := os.Symlink(targetPath, pidPath); err != nil {
		t.Fatalf("create pid symlink: %v", err)
	}
	if cleanup, err := AcquirePidFile(pidPath); err == nil {
		cleanup()
		t.Fatal("AcquirePidFile followed a symlink")
	}
	if pid, err := ReadPidFile(pidPath); err == nil {
		t.Fatalf("ReadPidFile followed a symlink and read pid %d", pid)
	}
}

// TestPidFileRejectsSymlinkedParent checks that pid files cannot be placed through linked directories.
func TestPidFileRejectsSymlinkedParent(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	if err := os.Mkdir(realDir, 0o700); err != nil {
		t.Fatalf("make real dir: %v", err)
	}
	linkDir := filepath.Join(dir, "linked")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatalf("make linked dir: %v", err)
	}
	pidPath := filepath.Join(linkDir, "fast-volume-syncer.pid")
	if cleanup, err := AcquirePidFile(pidPath); err == nil {
		cleanup()
		t.Fatal("AcquirePidFile accepted a symlinked parent directory")
	}
}

// TestAcquirePidFileRejectsSecondLock checks that the pid-file lock rejects a second daemon.
func TestAcquirePidFileRejectsSecondLock(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "fast-volume-syncer.pid")
	cleanup, err := AcquirePidFile(pidPath)
	if err != nil {
		t.Fatalf("AcquirePidFile() error = %v", err)
	}
	defer cleanup()

	secondCleanup, err := AcquirePidFile(pidPath)
	if err == nil {
		secondCleanup()
		t.Fatal("expected second AcquirePidFile to fail while first lock is held")
	}
}
