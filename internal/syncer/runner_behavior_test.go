// Package syncer prepares sandboxed mounts and runs the copier against resolved paths.
package syncer

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// TestRunnerLogOutputTrimsLinesAndAppliesPrefix checks log formatting for reporting helpers.
func TestRunnerLogOutputTrimsLinesAndAppliesPrefix(t *testing.T) {
	var buf bytes.Buffer
	oldWriter := util.InfoLog.Writer()
	oldFlags := util.InfoLog.Flags()
	oldPrefix := util.InfoLog.Prefix()
	util.InfoLog.SetOutput(&buf)
	util.InfoLog.SetFlags(0)
	util.InfoLog.SetPrefix("")
	defer func() {
		util.InfoLog.SetOutput(oldWriter)
		util.InfoLog.SetFlags(oldFlags)
		util.InfoLog.SetPrefix(oldPrefix)
	}()

	(&Runner{}).logOutput(strings.NewReader("first  \nsecond\t\n"), "header=>", "|")

	got := buf.String()
	want := "header=>|first|second\n"
	if got != want {
		t.Fatalf("logged output = %q, want %q", got, want)
	}
}

// TestSafeWorkspacePathRejectsEscapes verifies selector subpaths cannot leave a mounted root.
func TestSafeWorkspacePathRejectsEscapes(t *testing.T) {
	root := t.TempDir()

	accepted, err := safeWorkspacePath(root, filepath.Join("project", "file.txt"))
	if err != nil {
		t.Fatalf("safeWorkspacePath accepted relative path: %v", err)
	}
	if !strings.HasPrefix(accepted, root+string(filepath.Separator)) {
		t.Fatalf("joined path %q does not stay under %q", accepted, root)
	}

	for _, subpath := range []string{"/etc", "..", filepath.Join("..", "etc"), filepath.Join("project", "..", "..", "etc")} {
		if got, err := safeWorkspacePath(root, subpath); err == nil {
			t.Fatalf("safeWorkspacePath(%q) = %q, want error", subpath, got)
		}
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("make symlink component: %v", err)
	}
	if got, err := safeWorkspacePath(root, filepath.Join("linked", "file.txt")); err == nil {
		t.Fatalf("safeWorkspacePath followed symlink and returned %q", got)
	}
}

// TestEnsureNoSymlinkPathRejectsExistingLinks checks the final pre-copy path guard.
func TestEnsureNoSymlinkPathRejectsExistingLinks(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatalf("make symlink component: %v", err)
	}
	if err := ensureNoSymlinkPath(filepath.Join(root, "plain", "missing")); err != nil {
		t.Fatalf("ensureNoSymlinkPath rejected missing safe tail: %v", err)
	}
	if err := ensureNoSymlinkPath(filepath.Join(root, "linked", "file.txt")); err == nil {
		t.Fatal("ensureNoSymlinkPath accepted a symlink component")
	}
}

// BenchmarkRunnerLogOutput measures reporting-log formatting overhead.
func BenchmarkRunnerLogOutput(b *testing.B) {
	oldWriter := util.InfoLog.Writer()
	oldFlags := util.InfoLog.Flags()
	oldPrefix := util.InfoLog.Prefix()
	util.InfoLog.SetOutput(io.Discard)
	util.InfoLog.SetFlags(0)
	util.InfoLog.SetPrefix("")
	defer func() {
		util.InfoLog.SetOutput(oldWriter)
		util.InfoLog.SetFlags(oldFlags)
		util.InfoLog.SetPrefix(oldPrefix)
	}()

	input := strings.Repeat("file mode size path\n", 64)
	runner := &Runner{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runner.logOutput(strings.NewReader(input), "header=>", "\t")
	}
}
