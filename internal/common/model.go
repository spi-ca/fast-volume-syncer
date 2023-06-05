package common

type RemoteInfo struct {
	MountInfo

	// 마운트 후 내부 Path
	SubPath string `json:"sub_path"`
}
