package find

var (
	ignoreFilename = map[string]bool{
		".":                                   true,
		"..":                                  true,
		".dropbox":                            true,
		".dropbox.attr":                       true,
		".dropbox.cache":                      true,
		".DS_Store":                           true,
		".AppleDouble":                        true,
		".LSOverride":                         true,
		"Icon\r":                              true,
		"Icon\r\r":                            true,
		".DocumentRevisions-V100":             true,
		".fseventsd":                          true,
		".Spotlight-V100":                     true,
		".TemporaryItems":                     true,
		".Trashes":                            true,
		".VolumeIcon.icns":                    true,
		".com.apple.timemachine.donotpresent": true,
		".AppleDB":                            true,
		".AppleDesktop":                       true,
		"Network Trash Folder":                true,
		"Temporary Items":                     true,
		".apdisk":                             true,
		"Thumbs.db":                           true,
		"Thumbs.db:encryptable":               true,
		"ehthumbs.db":                         true,
		"ehthumbs_vista.db":                   true,
		"desktop.ini":                         true,
		"Desktop.ini":                         true,
	}
)
