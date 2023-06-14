package copier

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type result struct {
	startIdx      int
	lastFilenames [10]string
	total         int
	sent          int
	uptodate      int
	skipped       int
	sentBytes     int64
	started       time.Time
	errs          []error
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

func (r *result) String() string {
	buf := &strings.Builder{}

	_, _ = fmt.Fprintf(buf, "Copier")

	if !r.started.IsZero() {
		elapsed := time.Now().Sub(r.started)
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
		_, _ = fmt.Fprintf(buf, " total(%d) = sent(%d) + uptodate(%d) + skipped(%d) + untouched(%d),",
			r.total, r.sent, r.uptodate, r.skipped, r.total-r.sent-r.uptodate-r.skipped,
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
		return fmt.Errorf("%w %s", err, buf.String())
	} else {
		return nil
	}
}
