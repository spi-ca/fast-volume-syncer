package find

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"github.com/charlievieth/fastwalk"
)

func (s *Scanner) scanDirectory(ctx context.Context, root string, rowChan chan<- returns.Fileinfo) error {
	var errs []error
	iter := func(path string, d os.DirEntry, err error) error {
		if err == nil {
			//do nothing
		} else if errors.Is(err, filepath.SkipAll) { // for fastwalk
			return filepath.SkipDir
		} else if errors.Is(err, filepath.SkipDir) { // for fastwalk
			return filepath.SkipDir
		} else {
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

		mode := info.Mode()
		if s.ignore(relPath, mode) {
			return nil
		}

		entry := returns.Fileinfo{
			Path: relPath,
			Mode: mode,
			Size: info.Size(),
		}
		if (mode.Type() & fs.ModeSymlink) != 0 {
			entry.SymlinkPath, err = os.Readlink(path)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to execute readlink file(%s) info: %w", path, err))
				return nil
			}
		}
		select {
		case rowChan <- entry:
			return nil
		case <-ctx.Done():
			return filepath.SkipAll
		}
	}

	conf := fastwalk.Config{}
	err := fastwalk.Walk(&conf, root, iter)

	if err == nil {
		// do nothing
	} else if errors.Is(err, filepath.SkipAll) { // for fastwalk
		// do nothing
	} else if errors.Is(err, filepath.SkipDir) { // for fastwalk
		// do nothing
	} else {
		errs = append(errs, fmt.Errorf("walkdir(%s) has returned err: %w", root, err))
	}

	return errors.Join(errs...)
}
