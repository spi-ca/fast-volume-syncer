package find

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
)

func (s *Scanner) scanDirectory(ctx context.Context, root string, rowChan chan<- returns.Fileinfo) error {
	var errs []error
	iter := func(path string, d os.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return filepath.SkipDir
		}

		info, err := d.Info()
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get file(%s) info: %w", path, err))
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to make relative file path(%s) info: %w", path, err))
			return filepath.SkipDir
		}

		entry := returns.Fileinfo{
			Path: relPath,
			Mode: info.Mode(),
			Size: info.Size(),
		}
		if (info.Mode().Type() & fs.ModeSymlink) != 0 {
			entry.SymlinkPath, err = os.Readlink(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to execute readlink file(%s) info: %w", path, err))
			}
			return nil
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
		errs = append(errs, fmt.Errorf("walkdir(%s) has returned err: %w", root, err))
	}
	return errors.Join(errs...)
}
