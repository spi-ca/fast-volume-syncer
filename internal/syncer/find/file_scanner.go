package find

import (
	"context"
	"fmt"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type Scanner struct {
	FinderBinaryPath string
}

func (s *Scanner) Scan(ctx context.Context, root string, entryChan chan<- returns.Fileinfo) {
	defer func() {
		close(entryChan)
		if err := recover(); err != nil {
			util.SendSlackMessage(fmt.Sprintf("panic on Scanner.executeFind : %v", err))
		}
	}()

	var err error
	if len(s.FinderBinaryPath) > 0 {
		util.InfoLog.Printf("directory scan using finder")
		err = s.executeFind(ctx, root, entryChan)
	} else {
		util.InfoLog.Printf("directory scan using filepath.WalkDir")
		err = s.scanDirectory(ctx, root, entryChan)
	}
	if err != nil {
		util.SendSlackMessage(fmt.Sprintf("Scanning failed : %v", err))
	}
}
