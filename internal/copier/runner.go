package copier

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier/native"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier/find"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/copier/rsync"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

var (
	pathSeperatorStr = string(filepath.Separator)
)

type Runner struct {
	FileMode os.FileMode

	Args args.RsyncArgs

	UseRsync bool

	ScanDuration     time.Duration
	FinderBinaryPath string

	TaskSize  int
	ChunkSize int
	Retry     args.RetryArgs
}

func (r *Runner) Execute(ctx context.Context, sourcePath string, destinationPath string) error {
	sourcePath, destinationPath, err := r.prepareDirectory(sourcePath, destinationPath)
	if err != nil {
		return err
	}

	util.InfoLog.Print("=> split files")
	util.InfoLog.Printf("TaskSize: %d, ChunkSize: %d, srcPath: %s, dstPath: %s", r.TaskSize, r.ChunkSize, sourcePath, destinationPath)

	scanner := find.Scanner{
		FinderBinaryPath: r.FinderBinaryPath,
		EntryChannelSize: r.TaskSize * r.ChunkSize,
	}

	joiner := chunkJoiner{
		taskSize:     r.TaskSize,
		chunkSize:    r.ChunkSize,
		scanDuration: r.ScanDuration,
	}

	if r.UseRsync {
		copyMethodHandler := rsync.Task{
			Arguments:       r.Args,
			FileMode:        r.FileMode,
			Retry:           r.Retry,
			SourcePath:      sourcePath,
			DestinationPath: destinationPath,
		}
		joiner.copier = copyMethodHandler.Execute
	} else {
		copyMethodHandler := native.Copier{
			SourceRoot:      sourcePath,
			DestinationRoot: destinationPath,
			FileMode:        r.FileMode,
			Retry:           r.Retry,
		}
		joiner.copier = copyMethodHandler.Execute
	}

	entryChan, scannerErrorChan := scanner.Scan(ctx, sourcePath)
	ioResultChan := joiner.Execute(ctx, entryChan)

	var (
		errs   []error
		result ioResult
	)

	for ioResult := range ioResultChan {
		result.Append(ioResult.Result)
		util.InfoLog.Printf("[acc]copy processed: %s", result)
		if ioResult.Error != nil {
			errs = append(errs, fmt.Errorf("chunk processing failed : %w", ioResult.Error))
		}
	}

	if scannerErr, ok := <-scannerErrorChan; ok {
		errs = append(errs, scannerErr)
	}
	err = errors.Join(errs...)

	if err == nil && ctx.Err() == nil {
		util.InfoLog.Printf("copy complete! (%s->%s), processed: %s", sourcePath, destinationPath, result)
	} else {
		util.InfoLog.Printf("copy stopped(%s->%s), processed: %s", sourcePath, destinationPath, result)
	}

	return err
}

func (r *Runner) prepareDirectory(sourcePath string, destinationPath string) (string, string, error) {
	// 아래 영역은 이제 host os 와 격리되었다.
	sourceStat, err := os.Stat(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get source file info(%s): %w", sourcePath, err)
	} else if sourceStat.IsDir() {
		// directory인 경우 slash 추가.
		sourcePath += pathSeperatorStr
	}

	// destination 확인
	destinationFilemode := sourceStat.Mode() | 0o700

	if err := os.MkdirAll(destinationPath, destinationFilemode.Perm()); err == nil {
		util.InfoLog.Printf("directory %s created", destinationPath)
	}

	return sourcePath, destinationPath, nil
}
