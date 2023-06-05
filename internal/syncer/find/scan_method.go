package find

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

func (s *Scanner) scanDirectory(ctx context.Context, root string, rowChan chan<- common.Fileinfo) {
	defer func() {
		close(rowChan)
		if err := recover(); err != nil {
			log.Printf("panic on Scanner.scanDirectory: %v", err)
		}
	}()
	iter := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		info, err := d.Info()
		if err != nil {
			log.Printf("failed to get file info: %v", err)
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			log.Printf("failed to make relative file path info: %v", err)
			return filepath.SkipDir
		}

		entry := common.Fileinfo{
			Path: relPath,
			Mode: info.Mode(),
			Size: info.Size(),
		}

		select {
		case rowChan <- entry:
			return nil
		case <-ctx.Done():
			return filepath.SkipAll
		}
	}
	err := filepath.WalkDir(root, iter)
	if err != nil {
		log.Printf("walkdir(%s) has returned err: %v", root, err)
	}
}
