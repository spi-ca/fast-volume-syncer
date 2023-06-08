package find

import (
	"context"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Scanner struct {
	FinderBinaryPath string

	TaskSize  int
	ChunkSize int
}

func (s *Scanner) Scan(ctx context.Context, root string) <-chan returns.Fileinfo {
	util.InfoLog.Printf("chunk size is %d", s.ChunkSize)
	entryChan := make(chan returns.Fileinfo, s.TaskSize*s.ChunkSize)
	if len(s.FinderBinaryPath) > 0 {
		util.InfoLog.Printf("directory scan using finder")
		go s.executeFind(ctx, root, entryChan)
	} else {
		util.InfoLog.Printf("directory scan using filepath.WalkDir")
		go s.scanDirectory(ctx, root, entryChan)
	}
	return entryChan
}
