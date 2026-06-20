// Package returns defines result objects shared by worker, sync, and mount flows.
package returns

import "strings"

// MountInfo describes one NFS mount target and the mount command derived from it.
type MountInfo struct {
	// Host is the NFS server hostname or address.
	Host string `json:"host"`

	// Path is the exported directory or volume path on the NFS server.
	Path string `json:"path"`

	// Options contains additional comma-separated mount options.
	Options string `json:"options"`
}

// Type returns the filesystem type used when constructing the mount command.
func (m MountInfo) Type() string {
	return "nfs"
}

// Source returns the NFS source in host:/export/path form.
func (m MountInfo) Source() string {
	builder := strings.Builder{}
	builder.WriteString(m.Host)
	builder.WriteString(":/")
	builder.WriteString(strings.Trim(m.Path, "/"))
	return builder.String()
}

// RefinedOptions prepends addr=<host> and keeps any user-supplied mount options.
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

// MountArg returns the argument list appended after the mount executable name.
func (m MountInfo) MountArg() (args []string) {
	args = append(args, "-t")
	args = append(args, m.Type())
	args = append(args, "-o")
	args = append(args, m.RefinedOptions())
	args = append(args, m.Source())
	return
}

// String renders the full mount command for logging and diagnostics.
func (m MountInfo) String() string {
	builder := strings.Builder{}
	builder.WriteString("mount")

	for _, item := range m.MountArg() {
		builder.WriteByte(' ')
		builder.WriteString(item)
	}
	return builder.String()
}
