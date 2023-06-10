package find

import (
	"context"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Scanner struct {
	FinderBinaryPath string
}

func (s *Scanner) Scan(ctx context.Context, root string, entryChan chan<- returns.Fileinfo) {
	defer func() {
		if err := recover(); err != nil {
			util.ErrLog.Printf("panic on Scanner.Scan : %v", err)
		}
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
		util.ErrLog.Printf("Scanning failed : %v", err)
	}
}
