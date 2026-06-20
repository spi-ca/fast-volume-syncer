// Package copier batches scanned entries and sends them to the selected copy backend.
package copier

import (
	"context"
	"io/fs"
	"sync"
	"testing"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
)

// joinerTestResult is a minimal IOResult implementation for chunk-joiner tests.
type joinerTestResult struct {
	// total is the synthetic entry count returned by the fake worker.
	total int64
}

// Total returns the number of entries reported by the fake chunk worker.
func (r joinerTestResult) Total() int64 { return r.total }

// Files reports every fake entry as a regular file.
func (r joinerTestResult) Files() int64 { return r.total }

// Links reports zero symbolic links for the fake result.
func (r joinerTestResult) Links() int64 { return 0 }

// Directories reports zero directories for the fake result.
func (r joinerTestResult) Directories() int64 { return 0 }

// SentBytes reports zero transferred bytes for the fake result.
func (r joinerTestResult) SentBytes() int64 { return 0 }

// TestChunkJoinerDispatchesFullAndTrailingChunks checks that full chunks and the final partial chunk are submitted.
func TestChunkJoinerDispatchesFullAndTrailingChunks(t *testing.T) {
	entries := make(chan returns.Fileinfo)
	var mu sync.Mutex
	var chunkSizes []int
	joiner := &chunkJoiner{taskSize: 4, chunkSize: 2, scanDuration: time.Hour, copier: func(_ context.Context, chunk []returns.Fileinfo) (returns.IOResult, error) {
		mu.Lock()
		chunkSizes = append(chunkSizes, len(chunk))
		mu.Unlock()
		return joinerTestResult{total: int64(len(chunk))}, nil
	}}

	resultChan := joiner.Execute(context.Background(), entries)
	for i := 0; i < 5; i++ {
		entries <- returns.Fileinfo{Path: "file", Mode: fs.FileMode(0o644)}
	}
	close(entries)

	var total int64
	for result := range resultChan {
		if result.Error != nil {
			t.Fatalf("chunk result error = %v", result.Error)
		}
		total += result.Result.Total()
	}
	if total != 5 {
		t.Fatalf("total processed entries = %d, want 5", total)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(chunkSizes) != 3 {
		t.Fatalf("chunk count = %d, want 3: %#v", len(chunkSizes), chunkSizes)
	}
}

// BenchmarkChunkJoinerDispatch measures chunk batching overhead with in-memory fake workers.
func BenchmarkChunkJoinerDispatch(b *testing.B) {
	for i := 0; i < b.N; i++ {
		entries := make(chan returns.Fileinfo)
		joiner := &chunkJoiner{taskSize: 4, chunkSize: 32, scanDuration: time.Hour, copier: func(_ context.Context, chunk []returns.Fileinfo) (returns.IOResult, error) {
			return joinerTestResult{total: int64(len(chunk))}, nil
		}}
		resultChan := joiner.Execute(context.Background(), entries)
		for j := 0; j < 128; j++ {
			entries <- returns.Fileinfo{Path: "file", Mode: fs.FileMode(0o644)}
		}
		close(entries)
		for range resultChan {
		}
	}
}
