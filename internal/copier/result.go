// Package copier batches scanned entries and sends them to the selected copy backend.
package copier

import (
	"fmt"
	"strings"
	"sync/atomic"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// ioResult accumulates totals across chunk copy results.
type ioResult struct {
	// total is the number of scanned entries accounted for so far.
	total int64
	// files counts copied or skipped regular files.
	files int64
	// links counts symbolic-link entries.
	links int64
	// directories counts directory entries.
	directories int64

	// sentBytes tracks bytes copied by the backend.
	sentBytes int64
}

// Total returns the accumulated entry count.
func (r ioResult) Total() int64 { return r.total }

// Files returns the accumulated regular-file count.
func (r ioResult) Files() int64 { return r.files }

// Links returns the accumulated symbolic-link count.
func (r ioResult) Links() int64 { return r.links }

// Directories returns the accumulated directory count.
func (r ioResult) Directories() int64 { return r.directories }

// SentBytes returns the accumulated copied byte count.
func (r ioResult) SentBytes() int64 { return r.sentBytes }

// Append atomically merges one chunk result into the running totals.
func (r *ioResult) Append(other returns.IOResult) {
	atomic.AddInt64(&r.total, other.Total())
	atomic.AddInt64(&r.files, other.Files())
	atomic.AddInt64(&r.links, other.Links())
	atomic.AddInt64(&r.directories, other.Directories())

	atomic.AddInt64(&r.sentBytes, other.SentBytes())
}

// String formats the accumulated counters for progress logging.
func (r ioResult) String() string {
	buf := &strings.Builder{}

	buf.WriteString(" sent ")
	buf.WriteString(util.FileSizeIEC(r.sentBytes))

	if r.total > 0 {
		_, _ = fmt.Fprintf(buf, " total(%d) = files(%d) + directories(%d) + symlinks(%d) + skipped(%d),",
			r.total, r.files, r.directories, r.links, r.total-r.files-r.directories-r.links,
		)
	}

	return buf.String()
}
