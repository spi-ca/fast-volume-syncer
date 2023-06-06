package find

import (
	"context"
	"log"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

type Scanner struct {
	FinderBinaryPath string

	TaskSize  int
	ChunkSize int
}

func (s *Scanner) Scan(ctx context.Context, root string) <-chan common.Fileinfo {
	log.Printf("chunk size is %d", s.ChunkSize)
	entryChan := make(chan common.Fileinfo, s.TaskSize*s.ChunkSize)
	if len(s.FinderBinaryPath) > 0 {
		log.Printf("directory scan using finder")
		go s.executeFind(ctx, root, entryChan)
	} else {
		log.Printf("directory scan using filepath.WalkDir")
		go s.scanDirectory(ctx, root, entryChan)
	}
	return entryChan
}
