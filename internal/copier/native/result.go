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

type result struct {
	chunkIdx      uint64
	startIdx      int
	lastFilenames [10]string
	total         int
	sent          int

	files       int
	links       int
	directories int

	processed   int
	uptodate    int
	disappeared int
	skipped     int
	sentBytes   int64

	started, ended time.Time

	errs []error
}

func (r *result) appendFilename(filename string) {
	r.lastFilenames[r.startIdx] = filename
	r.startIdx = (r.startIdx + 1) % len(r.lastFilenames)
}

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

func (r *result) addTypeCount(mode fs.FileMode) {
	if mode.IsDir() {
		r.directories++
	} else if mode.Type()&fs.ModeSymlink != 0 {
		r.links++
	} else if mode.IsRegular() {
		r.files++
	}
}
func (r *result) markEnd() { r.ended = time.Now() }

func (r result) Duration() time.Duration {
	if r.ended.After(r.started) {
		return r.ended.Sub(r.started)
	} else {
		return 0
	}
}
func (r result) Total() int64       { return int64(r.total) }
func (r result) Files() int64       { return int64(r.files) }
func (r result) Links() int64       { return int64(r.links) }
func (r result) Directories() int64 { return int64(r.directories) }
func (r result) SentBytes() int64   { return r.sentBytes }

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
