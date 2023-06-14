package find

import (
	"context"
	"fmt"
	"path/filepath"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Scanner struct {
	FinderBinaryPath string
	EntryChannelSize int
}

func (s *Scanner) Scan(ctx context.Context, root string) (<-chan returns.Fileinfo, <-chan error) {
	entryChan := make(chan returns.Fileinfo, s.EntryChannelSize)
	errorChan := make(chan error, 1)
	go s.execute(ctx, root, entryChan, errorChan)
	return entryChan, errorChan
}

func (s *Scanner) ignoreFilename(path string) bool {
	filename := filepath.Base(path)
	// 자기자신은 무시하자
	ignored, ok := ignoreFilename[filename]
	return ok && ignored
}

func (s *Scanner) execute(ctx context.Context, root string, entryChan chan<- returns.Fileinfo, errorChan chan<- error) {
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on Scanner.Scan : %v", err)
		}
		close(entryChan)
		close(errorChan)
	}()

	var scanner func(context.Context, string, chan<- returns.Fileinfo) error
	if len(s.FinderBinaryPath) > 0 {
		util.InfoLog.Printf("directory scan using finder")
		scanner = s.executeFind
	} else {
		util.InfoLog.Printf("directory scan using filepath.WalkDir")
		scanner = s.scanDirectory
	}

	err := scanner(ctx, root, entryChan)
	if err != nil {
		errorChan <- fmt.Errorf("scanning failed : %w", err)
	}
}
