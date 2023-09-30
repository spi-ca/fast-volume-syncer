package sys

import (
	"io/fs"
)

const (
	s_ISUID = 0o4000
	s_ISGID = 0o2000
	s_ISVTX = 0o1000

	s_IRUSR = 0o0400
	s_IWUSR = 0o0200
	s_IXUSR = 0o0100

	s_IRGRP = 0o0040
	s_IWGRP = 0o0020
	s_IXGRP = 0o0010

	s_IROTH = 0o0004
	s_IWOTH = 0o0002
	s_IXOTH = 0o0001
)

var (
	filemodeMap = [][]struct {
		mode fs.FileMode
		char byte
	}{
		{
			{fs.ModeSymlink, 'l'},
			{fs.ModeSocket, 's'},
			{0o0, '-'},
			{fs.ModeDevice, 'b'},
			{fs.ModeDir, 'd'},
			{fs.ModeDevice | fs.ModeCharDevice, 'c'},
			{fs.ModeNamedPipe, 'p'},
			{fs.ModeIrregular, '?'},
		},

		{{s_IRUSR, 'r'}},
		{{s_IWUSR, 'w'}},
		{
			{s_IXUSR | s_ISUID, 's'},
			{s_ISUID, 'S'},
			{s_IXUSR, 'x'},
		},

		{{s_IRGRP, 'r'}},
		{{s_IWGRP, 'w'}},
		{
			{s_IXGRP | s_ISGID, 's'},
			{s_ISGID, 'S'},
			{s_IXGRP, 'x'},
		},

		{{s_IROTH, 'r'}},
		{{s_IWOTH, 'w'}},
		{
			{s_IXOTH | s_ISVTX, 't'},
			{s_ISVTX, 'T'},
			{s_IXOTH, 'x'},
		},
	}
)

func UnFilemodeStr(modeStr string) fs.FileMode {
	var mode fs.FileMode

	for i, table := range filemodeMap {
		if i >= len(modeStr) {
			break
		}
		chr := modeStr[i]
		for _, bitchar := range table {
			if chr == bitchar.char {
				mode |= bitchar.mode
			}
		}
	}
	return mode
}

func UnFilemode(modeStr []byte) fs.FileMode {
	var mode fs.FileMode

	for i, table := range filemodeMap {
		chr := modeStr[i]
		for _, bitchar := range table {
			if chr == bitchar.char {
				mode |= bitchar.mode
			}
		}
	}
	return mode
}
