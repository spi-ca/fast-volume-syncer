package selector

import (
	"context"
	"log"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

func TestLoadMigrationInfo(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Printf("??")
	s := MigrationInfoSelector{
		Source: common.StorageInfo{
			Host:       "127.0.0.1",
			MountPoint: "/33",
			Options:    "ro",
		},
		Destination: common.StorageInfo{
			Host:       "127.0.0.2",
			MountPoint: "/44/55/",
			Options:    "rw",
		},
		NodeSelector: 1,
		WorkerSize:   5,
	}

	items := s.Load(ctx, "../../contrib/09_copy_entries.csv")
	for item := range items {
		log.Printf("!!%v", item)
	}
}
