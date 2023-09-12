package copier

import (
	"fmt"
	"strings"
	"sync/atomic"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type ioResult struct {
	total       int64
	files       int64
	links       int64
	directories int64

	sentBytes int64
}

func (r ioResult) Total() int64       { return r.total }
func (r ioResult) Files() int64       { return r.files }
func (r ioResult) Links() int64       { return r.links }
func (r ioResult) Directories() int64 { return r.directories }
func (r ioResult) SentBytes() int64   { return r.sentBytes }

func (r *ioResult) Append(other returns.IOResult) {
	atomic.AddInt64(&r.total, other.Total())
	atomic.AddInt64(&r.files, other.Files())
	atomic.AddInt64(&r.links, other.Links())
	atomic.AddInt64(&r.directories, other.Directories())

	atomic.AddInt64(&r.sentBytes, other.SentBytes())
}

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
