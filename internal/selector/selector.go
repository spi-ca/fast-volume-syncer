package selector

import (
	"context"
	"encoding/csv"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/common"
)

type MigrationInfoSelector struct {
	Source       common.StorageInfo
	Destination  common.StorageInfo
	NodeSelector int
	WorkerSize   int
}

func (s *MigrationInfoSelector) loadCopyEntryCSV(ctx context.Context, path string) <-chan copyEntry {
	entryChan := make(chan copyEntry, s.WorkerSize)
	go func(rowChan chan<- copyEntry) {
		defer close(rowChan)
		const entryNum = 12
		f, err := os.OpenFile(path, os.O_RDONLY, 0o666)
		if err != nil {
			log.Printf("file open failed: %v", err)
			return
		}

		defer f.Close()
		// csv reader 생성

		rdr := csv.NewReader(f)
		// ignore header
		row, err := rdr.Read()
		if err != nil {
			log.Printf("readline failed: %v", err)
			return
		}

		// csv 내용 모두 읽기
		for row, err = rdr.Read(); err == nil; row, err = rdr.Read() {
			if err == io.EOF {
				err = nil
				break
			} else if err != nil {
				log.Printf("read csv failed: %v", err)
				return
			}

			if len(row) < entryNum {
				continue
			} else if len(row) > entryNum {
				row = row[:entryNum]
			}

			var entry = copyEntry{}
			entry.Node, err = strconv.Atoi(row[0])
			if err != nil {
				log.Printf("node field parse  failed: %v", err)
				continue
			}

			entry.SourceVolume = strings.TrimSpace(row[1])
			entry.DestinationVolume = strings.TrimSpace(row[2])
			entry.SourcePath = strings.TrimSpace(row[3])
			entry.DestinationPath = strings.TrimSpace(row[4])

			entry.ProjectId, err = strconv.Atoi(row[5])
			if err != nil {
				log.Printf("project_id field parse  failed: %v", err)
				continue
			}

			entry.ProjectName = strings.TrimSpace(row[6])

			entry.UsedSize, err = strconv.ParseInt(row[7], 10, 64)
			if err != nil {
				log.Printf("project_id field parse  failed: %v", err)
				continue
			}

			entry.UsedSizeHuman = strings.TrimSpace(row[8])
			entry.VolumeType = strings.TrimSpace(row[9])
			entry.VolumeSize, err = strconv.ParseInt(row[10], 10, 64)
			if err != nil {
				log.Printf("project_id field parse  failed: %v", err)
				continue
			}
			entry.VolumeSizeHuman = strings.TrimSpace(row[11])
			select {
			case <-ctx.Done():
				return
			case rowChan <- entry:
			}
		}
	}(entryChan)
	return entryChan
}

func (s *MigrationInfoSelector) Load(ctx context.Context, path string) <-chan common.MigrationInfo {
	infoChan := make(chan common.MigrationInfo, s.WorkerSize)
	go func(rowChan chan<- common.MigrationInfo) {
		defer close(rowChan)
		items := s.loadCopyEntryCSV(ctx, path)
		for item := range items {
			if s.NodeSelector > -1 && s.NodeSelector != item.Node {
				continue
			}

			info := common.MigrationInfo{
				Source: common.MountInfo{
					s.Source,
					strings.Trim(item.SourceVolume, "/"),
				},
				Destination: common.MountInfo{
					s.Destination,
					strings.Trim(item.DestinationVolume, "/"),
				},
				SourcePath:      strings.Trim(item.SourcePath, "/"),
				DestinationPath: strings.Trim(item.DestinationPath, "/"),
			}
			select {
			case rowChan <- info:
			case <-ctx.Done():
				return
			}
		}
		log.Print("copy info readout!")
	}(infoChan)
	return infoChan
}
