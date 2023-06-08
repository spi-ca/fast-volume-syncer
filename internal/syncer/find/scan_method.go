package find

import (
	"context"
	"os"
	"path/filepath"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/model"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

func (s *Scanner) scanDirectory(ctx context.Context, root string, rowChan chan<- model.Fileinfo) {
	defer func() {
		close(rowChan)
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on Scanner.scanDirectory: %v", err)
		}
	}()
	iter := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}

		info, err := d.Info()
		if err != nil {
			util.ErrLog.Printf("failed to get file info: %v", err)
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			util.ErrLog.Printf("failed to make relative file path info: %v", err)
			return filepath.SkipDir
		}

		entry := model.Fileinfo{
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
		util.ErrLog.Printf("walkdir(%s) has returned err: %v", root, err)
	}
}
