package find

import (
	"context"
	"log"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
)

func TestScanner_scanDirectory(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Scanner{}
	infoChan := make(chan returns.Fileinfo)
	errChan := make(chan error, 1)
	go func() {
		defer close(infoChan)
		errChan <- s.scanDirectory(ctx, ".", infoChan)
	}()
	for entry := range infoChan {

		log.Printf("entry %v", entry)
	}
	if err := <-errChan; err != nil {
		t.Fatal(err)
	}
	log.Printf("ended ")
}
