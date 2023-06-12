package find

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
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

func (s *Scanner) execute(ctx context.Context, root string, entryChan chan<- returns.Fileinfo, errorChan chan<- error) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic on Scanner.Scan : %v", err)
		}
		close(errorChan)
	}()

	var scanner func(context.Context, string, chan<- returns.Fileinfo) error
	if len(s.FinderBinaryPath) > 0 {
		log.Infof("directory scan using finder")
		scanner = s.executeFind
	} else {
		log.Infof("directory scan using filepath.WalkDir")
		scanner = s.scanDirectory
	}

	err := scanner(ctx, root, entryChan)
	if err != nil {
		errorChan <- fmt.Errorf("scanning failed : %w", err)
	}
}
