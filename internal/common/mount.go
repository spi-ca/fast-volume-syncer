package common

import "strings"

type MountInfo struct {
	// Storage 서버 주소
	Host string `json:"host"`

	// 서버내의 마운트 주소(볼륨)
	Path string `json:"path"`

	// 마운트 옵션
	Options string `json:"options"`
}

func (m MountInfo) Type() string {
	return "nfs"
}

func (m MountInfo) Source() string {
	builder := strings.Builder{}
	builder.WriteString(m.Host)
	builder.WriteString(":/")
	builder.WriteString(strings.Trim(m.Path, "/"))
	return builder.String()
}

func (m MountInfo) RefinedOptions() string {
	builder := strings.Builder{}
	builder.WriteString("addr=")
	builder.WriteString(m.Host)
	if len(m.Options) > 0 {
		builder.WriteByte(',')
		builder.WriteString(m.Options)
	}
	return builder.String()
}

func (m MountInfo) MountArg() (args []string) {
	args = append(args, "-t")
	args = append(args, m.Type())
	args = append(args, "-o")
	args = append(args, m.RefinedOptions())
	args = append(args, m.Source())
	return
}

func (m MountInfo) String() string {
	builder := strings.Builder{}
	builder.WriteString("mount")

	for _, item := range m.MountArg() {
		builder.WriteByte(' ')
		builder.WriteString(item)
	}
	return builder.String()
}
