// Package native copies scanned entries with direct filesystem operations.
package native

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// makeNativeCopierFixture creates a source tree and matching Fileinfo slice for native copier tests.
func makeNativeCopierFixture(t testing.TB, files int, fileSize int) (string, []returns.Fileinfo, int64) {
	t.Helper()

	src := t.TempDir()
	if err := os.Mkdir(filepath.Join(src, "nested"), 0o755); err != nil {
		t.Fatalf("make nested directory: %v", err)
	}

	entries := []returns.Fileinfo{{Path: "nested", Mode: fs.ModeDir | 0o755}}
	var totalBytes int64
	payload := bytes.Repeat([]byte("x"), fileSize)
	mtime := time.Unix(1_700_000_000, 0)
	for i := 0; i < files; i++ {
		rel := filepath.Join("nested", "file-"+strconv.Itoa(i)+".dat")
		path := filepath.Join(src, rel)
		if err := os.WriteFile(path, payload, 0o644); err != nil {
			t.Fatalf("write source file %s: %v", rel, err)
		}
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatalf("set source file time %s: %v", rel, err)
		}
		entries = append(entries, returns.Fileinfo{Path: rel, Mode: 0o644, Size: int64(fileSize)})
		totalBytes += int64(fileSize)
	}
	return src, entries, totalBytes
}

// TestCopierExecuteCopiesFilesAndReportsCounts checks native copy handling for files, directories, and symlinks.
func TestCopierExecuteCopiesFilesAndReportsCounts(t *testing.T) {
	src, entries, totalBytes := makeNativeCopierFixture(t, 2, 32)
	dst := t.TempDir()

	linkPath := filepath.Join(src, "nested", "link")
	if err := os.Symlink("file-0.dat", linkPath); err != nil {
		t.Fatalf("make source symlink: %v", err)
	}
	entries = append(entries, returns.Fileinfo{Path: filepath.Join("nested", "link"), Mode: fs.ModeSymlink | 0o777, SymlinkPath: "file-0.dat"})

	result, err := (&Copier{SourceRoot: src, DestinationRoot: dst, FileMode: 0o640}).Execute(context.Background(), entries)
	if err != nil {
		t.Fatalf("execute copier: %v", err)
	}
	if result.Total() != int64(len(entries)) || result.Directories() != 1 || result.Files() != 2 || result.Links() != 1 {
		t.Fatalf("unexpected result counts: total=%d dirs=%d files=%d links=%d", result.Total(), result.Directories(), result.Files(), result.Links())
	}
	if result.SentBytes() != totalBytes {
		t.Fatalf("expected sent bytes %d, got %d", totalBytes, result.SentBytes())
	}

	copied, err := os.ReadFile(filepath.Join(dst, "nested", "file-0.dat"))
	if err != nil {
		t.Fatalf("read copied file: %v", err)
	}
	if !bytes.Equal(copied, bytes.Repeat([]byte("x"), 32)) {
		t.Fatalf("copied file content mismatch: %q", copied)
	}
	linkTarget, err := os.Readlink(filepath.Join(dst, "nested", "link"))
	if err != nil {
		t.Fatalf("read copied symlink: %v", err)
	}
	if linkTarget != "file-0.dat" {
		t.Fatalf("expected symlink target file-0.dat, got %q", linkTarget)
	}
}

// TestCopierExecuteDoesNotRecopyUpToDateFiles checks result accounting when every entry is already current.
func TestCopierExecuteDoesNotRecopyUpToDateFiles(t *testing.T) {
	src, entries, _ := makeNativeCopierFixture(t, 1, 16)
	dst := t.TempDir()
	copier := &Copier{SourceRoot: src, DestinationRoot: dst, FileMode: 0o640}

	if _, err := copier.Execute(context.Background(), entries); err != nil {
		t.Fatalf("initial execute copier: %v", err)
	}
	ioResult, err := copier.Execute(context.Background(), entries)
	if err != nil {
		t.Fatalf("second execute copier: %v", err)
	}
	result, ok := ioResult.(*result)
	if !ok {
		t.Fatalf("expected native result type, got %T", ioResult)
	}
	if result.sent != 0 || result.uptodate != len(entries) || result.SentBytes() != 0 {
		t.Fatalf("expected all entries up-to-date without bytes sent, got sent=%d uptodate=%d sentBytes=%d", result.sent, result.uptodate, result.SentBytes())
	}
}

// TestCopierExecuteRejectsRegularEntryChangedToSymlink checks regular source entries are opened without following symlinks.
func TestCopierExecuteRejectsRegularEntryChangedToSymlink(t *testing.T) {
	src, entries, _ := makeNativeCopierFixture(t, 1, 16)
	dst := t.TempDir()
	rel := filepath.Join("nested", "file-0.dat")
	if err := os.Remove(filepath.Join(src, rel)); err != nil {
		t.Fatalf("remove source regular file: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside-secret")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(src, rel)); err != nil {
		t.Fatalf("replace source with symlink: %v", err)
	}

	if _, err := (&Copier{SourceRoot: src, DestinationRoot: dst, FileMode: 0o640}).Execute(context.Background(), entries); err == nil {
		t.Fatal("copier followed a source symlink for a regular entry")
	}
}

// TestCopierExecuteRejectsDestinationSymlinkLeaf checks regular files do not leave symlink leaves in place.
func TestCopierExecuteRejectsDestinationSymlinkLeaf(t *testing.T) {
	src, entries, _ := makeNativeCopierFixture(t, 1, 16)
	dst := t.TempDir()
	if err := os.Mkdir(filepath.Join(dst, "nested"), 0o755); err != nil {
		t.Fatalf("make destination nested dir: %v", err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(dst, "nested", "file-0.dat")); err != nil {
		t.Fatalf("create destination symlink: %v", err)
	}

	if _, err := (&Copier{SourceRoot: src, DestinationRoot: dst, FileMode: 0o640}).Execute(context.Background(), entries); err == nil {
		t.Fatal("copier accepted a destination symlink leaf for a regular entry")
	}
}

// TestCopierExecuteDoesNotOverwriteNewerDestination checks that newer destination files are left untouched.
func TestCopierExecuteDoesNotOverwriteNewerDestination(t *testing.T) {
	src, entries, _ := makeNativeCopierFixture(t, 1, 16)
	dst := t.TempDir()
	copier := &Copier{SourceRoot: src, DestinationRoot: dst, FileMode: 0o640}

	if _, err := copier.Execute(context.Background(), entries); err != nil {
		t.Fatalf("initial execute copier: %v", err)
	}
	dstFile := filepath.Join(dst, "nested", "file-0.dat")
	newerContent := bytes.Repeat([]byte("n"), 16)
	if err := os.WriteFile(dstFile, newerContent, 0o640); err != nil {
		t.Fatalf("write newer destination: %v", err)
	}
	newerTime := time.Unix(1_800_000_000, 0)
	if err := os.Chtimes(dstFile, newerTime, newerTime); err != nil {
		t.Fatalf("set newer destination time: %v", err)
	}

	ioResult, err := copier.Execute(context.Background(), entries)
	if err != nil {
		t.Fatalf("second execute copier: %v", err)
	}
	result := ioResult.(*result)
	if result.sent != 0 || result.SentBytes() != 0 {
		t.Fatalf("expected newer destination to avoid recopying, got sent=%d sentBytes=%d", result.sent, result.SentBytes())
	}
	got, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if !bytes.Equal(got, newerContent) {
		t.Fatalf("newer destination was overwritten: got %q", got)
	}
}

// discardInfoLogForBenchmark silences progress logging so the benchmark measures copy work.
func discardInfoLogForBenchmark(b *testing.B) {
	b.Helper()
	oldWriter := util.InfoLog.Writer()
	oldFlags := util.InfoLog.Flags()
	oldPrefix := util.InfoLog.Prefix()
	util.InfoLog.SetOutput(io.Discard)
	util.InfoLog.SetFlags(0)
	util.InfoLog.SetPrefix("")
	b.Cleanup(func() {
		util.InfoLog.SetOutput(oldWriter)
		util.InfoLog.SetFlags(oldFlags)
		util.InfoLog.SetPrefix(oldPrefix)
	})
}

// BenchmarkCopierExecuteSmallFiles measures native copy throughput for many small files.
func BenchmarkCopierExecuteSmallFiles(b *testing.B) {
	discardInfoLogForBenchmark(b)
	src, entries, totalBytes := makeNativeCopierFixture(b, 32, 1024)
	baseDst := b.TempDir()
	b.SetBytes(totalBytes)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		dst, err := os.MkdirTemp(baseDst, "dst-")
		if err != nil {
			b.Fatalf("make destination directory: %v", err)
		}
		b.StartTimer()
		if _, err := (&Copier{SourceRoot: src, DestinationRoot: dst, FileMode: 0o640}).Execute(context.Background(), entries); err != nil {
			b.Fatalf("execute copier: %v", err)
		}
		b.StopTimer()
		if err := os.RemoveAll(dst); err != nil {
			b.Fatalf("remove destination directory: %v", err)
		}
		b.StartTimer()
	}
}
