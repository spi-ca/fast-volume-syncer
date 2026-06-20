// Package selector parses copy-entry CSV rows and fans them out to sync workers.
package selector

import (
	"strconv"
	"strings"
)

// copyEntry holds one parsed selector CSV row after trimming and numeric conversion.
type copyEntry struct {
	// Node is the node number used for selector-side row filtering.
	Node int
	// SourceVolume is the source storage path passed to the sync child.
	SourceVolume string
	// DestinationVolume is the destination storage path passed to the sync child.
	DestinationVolume string
	// SourcePath is the source subpath passed to the sync child.
	SourcePath string
	// DestinationPath is the destination subpath passed to the sync child.
	DestinationPath string
	// SourceProjectId is source metadata kept for logging and diagnostics.
	SourceProjectId int
	// SourceProjectName is source metadata kept for logging and diagnostics.
	SourceProjectName string
	// UsedSize is the recorded used byte count from the CSV row.
	UsedSize int64
	// UsedSizeHuman is the human-readable used size from the CSV row.
	UsedSizeHuman string
	// VolumeType carries the volume classification from the CSV row.
	VolumeType string
	// VolumeSize is the recorded provisioned byte count from the CSV row.
	VolumeSize int64
	// VolumeSizeHuman is the human-readable provisioned size from the CSV row.
	VolumeSizeHuman string
	// DestinationProjectName is destination metadata kept for logging and diagnostics.
	DestinationProjectName string
	// VolumeName is the source volume name captured in the CSV metadata.
	VolumeName string
	// SourceVolumeKey is the source-system identifier captured in the CSV metadata.
	SourceVolumeKey string
}

// String returns a verbose log representation of the selected copy entry.
func (e copyEntry) String() string {
	builder := strings.Builder{}
	builder.WriteString("copyEntry(")
	builder.WriteString("Node: ")
	builder.WriteString(strconv.Itoa(e.Node))
	builder.WriteString(", SourceVolume: ")
	builder.WriteString(e.SourceVolume)
	builder.WriteString(", DestinationVolume: ")
	builder.WriteString(e.DestinationVolume)
	builder.WriteString(", SourcePath: ")
	builder.WriteString(e.SourcePath)
	builder.WriteString(", DestinationPath: ")
	builder.WriteString(e.DestinationPath)
	builder.WriteString(", SourceProjectId: ")
	builder.WriteString(strconv.Itoa(e.SourceProjectId))
	builder.WriteString(", SourceProjectName: ")
	builder.WriteString(e.SourceProjectName)
	builder.WriteString(", UsedSize: ")
	builder.WriteString(strconv.FormatInt(e.UsedSize, 10))
	builder.WriteString(", UsedSizeHuman: ")
	builder.WriteString(e.UsedSizeHuman)
	builder.WriteString(", VolumeType: ")
	builder.WriteString(e.VolumeType)
	builder.WriteString(", VolumeSize: ")
	builder.WriteString(strconv.FormatInt(e.VolumeSize, 10))
	builder.WriteString(", VolumeSizeHuman: ")
	builder.WriteString(e.VolumeSizeHuman)
	builder.WriteString(", DestinationProjectName: ")
	builder.WriteString(e.DestinationProjectName)
	builder.WriteString(", VolumeName: ")
	builder.WriteString(e.VolumeName)
	builder.WriteString(", SourceVolumeKey: ")
	if e.SourceVolumeKey == "" {
		builder.WriteString("<empty>")
	} else {
		builder.WriteString("<redacted>")
	}
	builder.WriteString(")")
	return builder.String()
}
