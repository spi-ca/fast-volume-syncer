package selector

import (
	"strconv"
	"strings"
)

type copyEntry struct {
	Node                   int
	SourceVolume           string
	DestinationVolume      string
	SourcePath             string
	DestinationPath        string
	SourceProjectId        int
	SourceProjectName      string
	UsedSize               int64
	UsedSizeHuman          string
	VolumeType             string
	VolumeSize             int64
	VolumeSizeHuman        string
	DestinationProjectName string
	VolumeName             string
	SourceVolumeKey        string
}

func (e copyEntry) String() string {
	builder := strings.Builder{}
	builder.WriteString("copyEntry(")
	builder.WriteString("Node: ")
	builder.WriteString(strconv.FormatInt(int64(e.Node), 10))
	builder.WriteString(", SourceVolume: ")
	builder.WriteString(e.SourceVolume)
	builder.WriteString(", DestinationVolume: ")
	builder.WriteString(e.DestinationVolume)
	builder.WriteString(", SourcePath: ")
	builder.WriteString(e.SourcePath)
	builder.WriteString(", DestinationPath: ")
	builder.WriteString(e.DestinationPath)
	builder.WriteString(", SourceProjectId: ")
	builder.WriteString(strconv.FormatInt(int64(e.SourceProjectId), 10))
	builder.WriteString(", SourceProjectName: ")
	builder.WriteString(e.SourceProjectName)
	builder.WriteString(", UsedSize: ")
	builder.WriteString(strconv.FormatInt(int64(e.UsedSize), 10))
	builder.WriteString(", UsedSizeHuman: ")
	builder.WriteString(e.UsedSizeHuman)
	builder.WriteString(", VolumeType: ")
	builder.WriteString(e.VolumeType)
	builder.WriteString(", VolumeSize: ")
	builder.WriteString(strconv.FormatInt(int64(e.VolumeSize), 10))
	builder.WriteString(", VolumeSizeHuman: ")
	builder.WriteString(e.VolumeSizeHuman)
	builder.WriteString(", DestinationProjectName: ")
	builder.WriteString(e.DestinationProjectName)
	builder.WriteString(", VolumeName: ")
	builder.WriteString(e.VolumeName)
	builder.WriteString(", SourceVolumeKey: ")
	builder.WriteString(e.SourceVolumeKey)
	builder.WriteString(")")
	return builder.String()
}
