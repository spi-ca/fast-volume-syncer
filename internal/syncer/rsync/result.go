package rsync

import (
	"fmt"
	"strings"
)

type result struct {
	startIdx      int
	lastFilenames [5][]byte
	total         int
	sent          int
	uptodate      int
}

func (r *result) appendFilename(filename []byte) {
	r.lastFilenames[r.startIdx] = filename
	r.startIdx = (r.startIdx + 1) % len(r.lastFilenames)
}

func (r *result) listFilename() []string {
	filenames := make([]string, 0, len(r.lastFilenames))
	for i := 0; i < len(r.lastFilenames); i++ {
		filename := r.lastFilenames[(r.startIdx+i)%len(r.lastFilenames)]
		if len(filename) > 0 {
			filenames = append(filenames, string(filename))
		}
	}
	return filenames
}

func (r *result) String() string {
	listFiles := r.listFilename()
	return fmt.Sprintf("total(%d) = sent(%d) + uptodate(%d) + untouched(%d), last %d sent file = [%s]",
		r.total, r.sent, r.uptodate, r.total-r.sent-r.uptodate,
		len(listFiles), strings.Join(listFiles, ","),
	)
}
