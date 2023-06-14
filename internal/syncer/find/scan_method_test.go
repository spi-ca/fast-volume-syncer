package find

import (
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"
	"context"
	"log"
	"testing"
)

func TestScanner_scanDirectory(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := &Scanner{}
	infoChan := make(chan returns.Fileinfo)
	go s.scanDirectory(ctx, ".", infoChan)
	for entry := range infoChan {

		log.Printf("entry %v", entry)
	}
	log.Printf("ended ")
}
