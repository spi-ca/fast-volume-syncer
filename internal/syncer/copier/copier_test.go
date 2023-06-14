package copier

import (
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"context"
	"log"
	"path/filepath"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/syncer/find"
)

func TestCopier_copyNewFile(t1 *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &find.Scanner{
		FinderBinaryPath: "",
		EntryChannelSize: 0,
	}
	src, _ := filepath.Abs("dest")
	dest, _ := filepath.Abs("dest2")
	infoChan, scannerErrorChan := s.Scan(ctx, src)
	t := &Copier{
		SourceRoot:      src,
		DestinationRoot: dest,
		Umask:           0o770,
	}
	files := []returns.Fileinfo{}
	for entry := range infoChan {
		files = append(files, entry)
		//log.Print(entry)

	}
	err := t.Execute(ctx, files)
	if err != nil {
		log.Printf("err %v", err)
	}
	if scannerErr, ok := <-scannerErrorChan; ok {
		log.Fatalf("failed to scan :%v", scannerErr)
	}
}
