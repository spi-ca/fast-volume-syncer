// Package returns defines result objects shared by worker, sync, and mount flows.
package returns

import (
	"io/fs"
	"strconv"
	"strings"
)

// Fileinfo describes one discovered filesystem entry and its printable metadata.
type Fileinfo struct {
	// Path is the entry path relative to the scan or copy root.
	Path string
	// Mode is the fs.FileMode rendered at the start of String output.
	Mode fs.FileMode
	// Size is the entry size rendered as the second String column.
	Size int64
	// SymlinkPath is appended as the symlink target when the entry is a link.
	SymlinkPath string
}

// String renders the entry as "mode<TAB>size<TAB>path" with an optional symlink suffix.
func (e Fileinfo) String() string {
	builder := strings.Builder{}
	builder.WriteString(e.Mode.String())
	builder.WriteByte('\t')
	builder.WriteString(strconv.FormatInt(e.Size, 10))
	builder.WriteByte('\t')
	builder.WriteString(e.Path)
	if len(e.SymlinkPath) > 0 {
		builder.WriteString(" -> ")
		builder.WriteString(e.SymlinkPath)
	}
	return builder.String()
}
