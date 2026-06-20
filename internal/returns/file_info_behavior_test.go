// Package returns defines result objects shared by worker, sync, and mount flows.
package returns

import (
	"io/fs"
	"testing"
)

// TestFileinfoStringFormatsRegularFile verifies regular files render without a symlink suffix.
func TestFileinfoStringFormatsRegularFile(t *testing.T) {
	entry := Fileinfo{Path: "nested/file.txt", Mode: 0o640, Size: 128}
	if got, want := entry.String(), "-rw-r-----\t128\tnested/file.txt"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

// TestFileinfoStringFormatsSymlink verifies symlink entries append the resolved target.
func TestFileinfoStringFormatsSymlink(t *testing.T) {
	entry := Fileinfo{Path: "nested/link", Mode: fs.ModeSymlink | 0o777, Size: 12, SymlinkPath: "target"}
	if got, want := entry.String(), "Lrwxrwxrwx\t12\tnested/link -> target"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

// BenchmarkFileinfoString measures allocation cost for the printable file listing format.
func BenchmarkFileinfoString(b *testing.B) {
	entry := Fileinfo{Path: "nested/link", Mode: fs.ModeSymlink | 0o777, Size: 12, SymlinkPath: "target"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if entry.String() == "" {
			b.Fatal("expected formatted entry")
		}
	}
}
