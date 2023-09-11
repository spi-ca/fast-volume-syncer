package copier

import (
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type ioResult struct {
	in int64

	total       int64
	files       int64
	links       int64
	directories int64

	sentBytes int64
}

func (r ioResult) Duration() time.Duration { return time.Duration(r.in) * time.Microsecond }
func (r ioResult) Total() int64            { return r.total }
func (r ioResult) Files() int64            { return r.files }
func (r ioResult) Links() int64            { return r.links }
func (r ioResult) Directories() int64      { return r.directories }
func (r ioResult) SentBytes() int64        { return r.sentBytes }

func (r *ioResult) Append(other returns.IOResult) {
	atomic.AddInt64(&r.in, other.Duration().Microseconds())

	atomic.AddInt64(&r.total, other.Total())
	atomic.AddInt64(&r.files, other.Files())
	atomic.AddInt64(&r.links, other.Links())
	atomic.AddInt64(&r.directories, other.Directories())

	atomic.AddInt64(&r.sentBytes, other.SentBytes())
}

func (r ioResult) String() string {
	buf := &strings.Builder{}

	if r.in > 0 {
		elapsed := r.Duration()
		_, _ = fmt.Fprintf(buf, " in %s", elapsed)
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
		_, _ = fmt.Fprintf(buf, " total(%d) = files(%d) + directories(%d) + symlinks(%d) + skipped(%d),",
			r.total, r.files, r.directories, r.links, r.total-r.files-r.directories-r.links,
		)
	}

	return buf.String()
}
