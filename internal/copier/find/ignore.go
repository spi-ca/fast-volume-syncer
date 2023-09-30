package find

import (
	"os"
	"path/filepath"
)

var (
	ignoreDirname = map[string]bool{
		"..":                                  true,
		".snapshot":                           true,
		".dropbox":                            true,
		".com.apple.timemachine.donotpresent": true,
		".DocumentRevisions-V100":             true,
		".fseventsd":                          true,
		".Spotlight-V100":                     true,
		".TemporaryItems":                     true,
		"._Temporary Items":                   true,
		"Temporary Items":                     true,
		".Trashes":                            true,
		".Trash":                              true,
		"Network Trash Folder":                true,
		".AppleDB":                            true,
		".AppleDouble":                        true,
		".AppleDesktop":                       true,
		"$RECYCLE.BIN":                        true,
	}
	ignoreFilename = map[string]bool{
		".":                     true,
		"..":                    true,
		".dropbox.attr":         true,
		".dropbox.cache":        true,
		".DS_Store":             true,
		".LSOverride":           true,
		"Icon\r":                true,
		"Icon\r\r":              true,
		".VolumeIcon.icns":      true,
		".apdisk":               true,
		"Thumbs.db":             true,
		"Thumbs.db:encryptable": true,
		"ehthumbs.db":           true,
		"ehthumbs_vista.db":     true,
		"desktop.ini":           true,
		"Desktop.ini":           true,
	}
)

func (s *Scanner) ignoreFilename(path string) bool {
	filename := filepath.Base(path)
	// 자기자신은 무시하자
	ignored, ok := ignoreFilename[filename]
	return ok && ignored
}

func (s *Scanner) ignoreDirname(path string) bool {
	filename := filepath.Base(path)
	// 자기자신은 무시하자
	ignored, ok := ignoreDirname[filename]
	return ok && ignored
}

func (s *Scanner) ignore(path string, mode os.FileMode) bool {
	if mode.IsDir() {
		if s.ignoreDirname(path) {
			return true
		}
	} else if s.ignoreFilename(path) {
		return true
	} else if s.ignoreDirname(filepath.Dir(path)) {
		return true
	}
	return false
}
