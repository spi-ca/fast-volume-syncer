package returns

import (
	"io/fs"
	"strconv"
	"strings"
)

type Fileinfo struct {
	Path string
	Mode fs.FileMode
	Size int64
}

func (e Fileinfo) String() string {
	builder := strings.Builder{}
	builder.WriteString(e.Mode.String())
	builder.WriteByte('\t')
	builder.WriteString(strconv.FormatInt(e.Size, 10))
	builder.WriteByte('\t')
	builder.WriteString(e.Path)
	return builder.String()
}
