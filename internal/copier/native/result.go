// Package native copies scanned entries with direct filesystem operations.
package native

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"strings"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// result records native chunk progress, counters, and the last touched files.
type result struct {
	// chunkIdx identifies the chunk in logs and error messages.
	chunkIdx uint64
	// startIdx is the ring-buffer cursor for lastFilenames.
	startIdx int
	// lastFilenames keeps the most recent paths seen in this chunk.
	lastFilenames [10]string
	// total is the number of entries scheduled for the chunk.
	total int
	// sent counts regular files copied into place.
	sent int

	// files counts regular-file entries observed in the chunk.
	files int
	// links counts symbolic-link entries observed in the chunk.
	links int
	// directories counts directory entries observed in the chunk.
	directories int

	// processed counts entries deliberately left untouched, such as newer destinations.
	processed int
	// uptodate counts entries that already matched the destination.
	uptodate int
	// disappeared counts sources that vanished during processing.
	disappeared int
	// skipped counts unsupported entry types.
	skipped int
	// sentBytes accumulates bytes copied for regular files.
	sentBytes int64

	// started and ended bound the chunk execution window.
	started, ended time.Time

	// errs collects per-entry failures for HandleError.
	errs []error
}

// appendFilename stores filename in the fixed-size recent-file ring buffer.
func (r *result) appendFilename(filename string) {
	r.lastFilenames[r.startIdx] = filename
	r.startIdx = (r.startIdx + 1) % len(r.lastFilenames)
}

// listFilename returns the recent-file ring buffer in chronological order.
func (r *result) listFilename() []string {
	filenames := make([]string, 0, len(r.lastFilenames))
	for i := 0; i < len(r.lastFilenames); i++ {
		filename := r.lastFilenames[(r.startIdx+i)%len(r.lastFilenames)]
		if len(filename) > 0 {
			filenames = append(filenames, filename)
		}
	}
	return filenames
}

// addTypeCount increments the file-type counters for one entry.
func (r *result) addTypeCount(mode fs.FileMode) {
	if mode.IsDir() {
		r.directories++
	} else if mode.Type()&fs.ModeSymlink != 0 {
		r.links++
	} else if mode.IsRegular() {
		r.files++
	}
}

// markEnd stamps the end time once chunk execution finishes.
func (r *result) markEnd() { r.ended = time.Now() }

// Duration reports how long chunk execution ran.
func (r result) Duration() time.Duration {
	if r.ended.After(r.started) {
		return r.ended.Sub(r.started)
	} else {
		return 0
	}
}

// Total returns the number of scheduled entries.
func (r result) Total() int64 { return int64(r.total) }

// Files returns the number of regular-file entries counted in the chunk.
func (r result) Files() int64 { return int64(r.files) }

// Links returns the number of symbolic-link entries counted in the chunk.
func (r result) Links() int64 { return int64(r.links) }

// Directories returns the number of directory entries counted in the chunk.
func (r result) Directories() int64 { return int64(r.directories) }

// SentBytes returns the number of bytes copied by this chunk.
func (r result) SentBytes() int64 { return r.sentBytes }

// String formats chunk progress, throughput, and recent filenames for logs.
func (r result) String() string {
	buf := &strings.Builder{}

	_, _ = fmt.Fprintf(buf, "[chk:%d]", r.chunkIdx)

	if !r.started.IsZero() {
		elapsed := time.Since(r.started)
		_, _ = fmt.Fprintf(buf, " in %2.2f ms", float32(elapsed.Microseconds())/1000)
		if r.sentBytes > 0 {
			buf.WriteString(" sent ")
			buf.WriteString(util.FileSizeIEC(r.sentBytes))
			bytesPerSeconds := int64(float64(r.sentBytes) / math.Max(elapsed.Seconds(), 0.001))
			if bytesPerSeconds > 0 {
				buf.WriteString("(")
				buf.WriteString(util.FileSizeIEC(bytesPerSeconds))
				buf.WriteString("/s)")
			}
		}
	} else if r.sentBytes > 0 {
		buf.WriteString(" sent ")
		buf.WriteString(util.FileSizeIEC(r.sentBytes))
	}
	if r.total > 0 {
		_, _ = fmt.Fprintf(buf, " total(%d) = sent(%d) + processed(%d) + uptodate(%d) + disappeared(%d) + skipped(%d) + untouched(%d),",
			r.total, r.sent, r.processed, r.uptodate, r.disappeared, r.skipped, r.total-r.sent-r.processed-r.uptodate-r.disappeared-r.skipped,
		)
	}
	if len(r.errs) > 0 {
		listFiles := r.listFilename()
		if num := len(listFiles); num > 0 {
			_, _ = fmt.Fprintf(buf, "\n=> last %d sent file", num)
			if num > 1 {
				buf.WriteByte('s')
			}
			buf.WriteString(" = [\n")
			for idx, filename := range listFiles {
				buf.WriteString("\t'")
				buf.WriteString(filename)
				buf.WriteByte('\'')
				if idx+1 < num {
					buf.WriteByte(',')
				}
				buf.WriteByte('\n')
			}
			buf.WriteByte(']')
		}
	}

	return buf.String()
}

// HandleError joins recorded entry failures and annotates them with recent filenames.
func (r *result) HandleError() error {
	buf := &strings.Builder{}
	lastListFiles := r.listFilename()

	if num := len(lastListFiles); num > 0 {
		_, _ = fmt.Fprintf(buf, "\n=> last %d sent file", num)
		if num > 1 {
			buf.WriteByte('s')
		}
		buf.WriteString(" = [\n")
		for idx, filename := range lastListFiles {
			buf.WriteString("\t'")
			buf.WriteString(filename)
			buf.WriteByte('\'')
			if idx+1 < num {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteByte(']')
	}

	err := errors.Join(r.errs...)
	if err != nil {
		return fmt.Errorf("[chk:%d]%w %s", r.chunkIdx, err, buf.String())
	} else {
		return nil
	}
}
