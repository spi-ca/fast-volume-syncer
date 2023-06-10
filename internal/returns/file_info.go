package returns

import (
	"io/fs"
	"strconv"
	"strings"
)

type Fileinfo struct {
	Path        string
	Mode        fs.FileMode
	Size        int64
	SymlinkPath string
}

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
