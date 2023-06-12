package selector

import (
	"context"
	"encoding/csv"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"strconv"
	"strings"
)

type Runner struct {
	NodeSelector    int
	CopyInfoCSVPath string

	WorkerSize int

	Template Invoker
}

func (r *Runner) Execute(ctx context.Context) error {
	var f io.Reader
	if r.CopyInfoCSVPath == "-" {
		f = io.NopCloser(os.Stdout)
	} else if rawFile, err := os.OpenFile(r.CopyInfoCSVPath, os.O_RDONLY, 0o666); err != nil {
		return err
	} else {
		defer rawFile.Close()
		f = rawFile
	}

	entryChan := make(chan copyEntry, r.WorkerSize)

	go r.loadCopyEntryCSV(ctx, f, entryChan)

	joiner := &workerJoiner{
		workerSize: r.WorkerSize,
		invoker:    &r.Template,
	}

	return joiner.Execute(ctx, entryChan)
}

func (r *Runner) loadCopyEntryCSV(ctx context.Context, reader io.Reader, entryChan chan<- copyEntry) {
	defer close(entryChan)

	const entryNum = 15
	readItems := 0
	defer func() {
		log.Infof("read %d items", readItems)
		if err := recover(); err != nil {
			log.Errorf("panic on Runner.loadCopyEntryCSV : %v", err)
		}
	}()

	// csv reader 생성
	rdr := csv.NewReader(reader)
	// csv 내용 모두 읽기
	for {
		row, err := rdr.Read()
		if err == io.EOF {
			err = nil
			break
		} else if err != nil {
			log.Errorf("read csv failed: %v", err)
			break
		} else if len(row) < entryNum {
			continue
		} else if len(row[0]) < 1 {
			continue
		} else if firstChar := row[0][0]; firstChar < '0' || firstChar > '9' {
			continue
		}

		nodeNum, err := strconv.Atoi(row[0])
		if err != nil {
			log.Errorf("node field parse failed: %v", err)
			continue
		} else if r.NodeSelector >= 0 && r.NodeSelector != nodeNum {
			continue
		}

		var entry = copyEntry{}
		entry.Node = nodeNum
		entry.SourceVolume = strings.TrimSpace(row[1])
		entry.DestinationVolume = strings.TrimSpace(row[2])
		entry.SourcePath = strings.TrimSpace(row[3])
		entry.DestinationPath = strings.TrimSpace(row[4])
		entry.SourceProjectId, err = strconv.Atoi(row[5])
		if err != nil {
			log.Errorf("source_project_id field parse  failed: %v", err)
			continue
		}

		entry.SourceProjectName = strings.TrimSpace(row[6])

		entry.UsedSize, err = strconv.ParseInt(row[7], 10, 64)
		if err != nil {
			log.Errorf("used_size field parse  failed: %v", err)
			continue
		}

		entry.UsedSizeHuman = strings.TrimSpace(row[8])
		entry.VolumeType = strings.TrimSpace(row[9])
		entry.VolumeSize, err = strconv.ParseInt(row[10], 10, 64)
		if err != nil {
			log.Errorf("volume_size field parse  failed: %v", err)
			continue
		}
		entry.VolumeSizeHuman = strings.TrimSpace(row[11])
		entry.DestinationProjectName = strings.TrimSpace(row[12])
		entry.VolumeName = strings.TrimSpace(row[13])
		entry.SourceVolumeKey = strings.TrimSpace(row[14])
		select {
		case <-ctx.Done():
			return
		case entryChan <- entry:
			readItems++
		}
	}

}
