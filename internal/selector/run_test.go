package selector

import (
	"context"
	"log"
	"testing"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

func TestDoMigration(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log.Printf("??")
	copyEntriesPath := "../../contrib/09_copy_entries.csv"
	s := MigrationInfoSelector{
		Source: common.StorageInfo{
			Host:       "127.0.0.1",
			MountPoint: "/from",
			Options:    "ro",
		},
		Destination: common.StorageInfo{
			Host:       "127.0.0.2",
			MountPoint: "/4\n4/55/",
			Options:    "rw",
		},
		NodeSelector: 6,
		WorkerSize:   5,
	}

	DoMigration(ctx, &s, copyEntriesPath)
}
