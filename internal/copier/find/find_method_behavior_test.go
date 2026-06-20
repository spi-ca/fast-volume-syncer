// Package find scans source trees with either `find -ls` or an in-process walker.
package find

import (
	"io/fs"
	"testing"
)

// regularFindLine is a representative `find -ls` line for a regular file.
const regularFindLine = "23643898148733866   15 -rw-r--r--   1 root     root        14415 May  9  2022 /fixture/root/file.json"

// symlinkFindLine is a representative `find -ls` line for a symbolic link.
const symlinkFindLine = "7881299523698987    1 lrwxrwxrwx   1 root     root           52 May 23 20:04 /fixture/root/vocab.json -> ../../blobs/vocab.json"

// TestScannerParseFindEntryRegularFile checks that `find -ls` regular-file rows are parsed correctly.
func TestScannerParseFindEntryRegularFile(t *testing.T) {
	entry, err := (&Scanner{}).parseFindEntry([]byte(regularFindLine))
	if err != nil {
		t.Fatalf("parse regular find line: %v", err)
	}
	if entry.Path != "/fixture/root/file.json" {
		t.Fatalf("expected parsed path, got %q", entry.Path)
	}
	if entry.Size != 14415 {
		t.Fatalf("expected byte-size field 14415, got %d", entry.Size)
	}
	if !entry.Mode.IsRegular() || entry.Mode.Perm() != 0o644 {
		t.Fatalf("expected regular 0644 mode, got %s", entry.Mode)
	}
}

// TestScannerParseFindEntrySymlink checks that symlink targets are split from the printed path.
func TestScannerParseFindEntrySymlink(t *testing.T) {
	entry, err := (&Scanner{}).parseFindEntry([]byte(symlinkFindLine))
	if err != nil {
		t.Fatalf("parse symlink find line: %v", err)
	}
	if entry.Path != "/fixture/root/vocab.json" {
		t.Fatalf("expected parsed symlink path, got %q", entry.Path)
	}
	if entry.SymlinkPath != "../../blobs/vocab.json" {
		t.Fatalf("expected parsed symlink target, got %q", entry.SymlinkPath)
	}
	if entry.Mode.Type()&fs.ModeSymlink == 0 {
		t.Fatalf("expected symlink mode, got %s", entry.Mode)
	}
}

// BenchmarkScannerParseFindEntry measures regex parsing cost for `find -ls` rows.
func BenchmarkScannerParseFindEntry(b *testing.B) {
	s := &Scanner{}
	line := []byte(regularFindLine)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		entry, err := s.parseFindEntry(line)
		if err != nil {
			b.Fatalf("parse find line: %v", err)
		}
		if entry.Path == "" {
			b.Fatal("expected parsed path")
		}
	}
}
