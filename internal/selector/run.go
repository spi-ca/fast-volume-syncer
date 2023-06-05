package selector

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"os"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

func DoMigration(ctx context.Context, selector *MigrationInfoSelector, volumeFileList string) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	infoChan := selector.Load(ctx, volumeFileList)

	w := json.NewEncoder(os.Stdout)
	buf := bytes.Buffer{}
	r := json.NewDecoder(&buf)

	//sem := semaphore.NewWeighted(int64(selector.WorkerSize))

	for item := range infoChan {
		err := w.Encode(&item)
		if err != nil {
			log.Printf("!!%v", err)
		}
		buf.WriteString(`{"source":{"host":"127.0.0.1","mount_point":"/from","options":"ro","volume":"storage-a"},"destination":{"host":"127.0.0.2","mount_point":"/4\n4/55/","options":"rw","volume":"vol_fixture-00000000-0000-4000-8000-000000000000"},"source_path":"fixture/ro/fixture-data/units/58/project/project-a","destination_path":"project/brain-prj"}`)
		temp := &common.MigrationInfo{}
		_ = r.Decode(temp)
		log.Printf("done! : %v", temp.Source)
	}
	//exec.CommandContext()
}
