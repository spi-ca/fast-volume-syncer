// Package find scans source trees with either `find -ls` or an in-process walker.
package find

import (
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
	"context"
	"fmt"
)

// Scanner chooses the scanning strategy and buffers discovered entries.
type Scanner struct {
	// FinderBinaryPath switches scanning to an external `find -ls` process when set.
	FinderBinaryPath string
	// EntryChannelSize sizes the buffered entry stream returned by Scan.
	EntryChannelSize int
}

// Scan starts scanning root and returns the entry stream plus a one-shot error channel.
func (s *Scanner) Scan(ctx context.Context, root string) (<-chan returns.Fileinfo, <-chan error) {
	entryChan := make(chan returns.Fileinfo, s.EntryChannelSize)
	errorChan := make(chan error, 1)
	go s.execute(ctx, root, entryChan, errorChan)
	return entryChan, errorChan
}

// execute selects the scanner implementation and closes both channels on exit.
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
		util.InfoLog.Printf("directory scan using find binary")
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
