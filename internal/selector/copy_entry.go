package selector

type copyEntry struct {
	Node              int
	SourceVolume      string
	DestinationVolume string
	SourcePath        string
	DestinationPath   string
	ProjectId         int
	ProjectName       string
	UsedSize          int64
	UsedSizeHuman     string
	VolumeType        string
	VolumeSize        int64
	VolumeSizeHuman   string
}
