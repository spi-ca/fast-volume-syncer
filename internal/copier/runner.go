// Package copier batches scanned entries and sends them to the selected copy backend.
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

// Runner wires scanning, chunking, and the selected copy implementation together.
type Runner struct {
	// FileMode is OR-ed into destination permissions for created files and directories.
	FileMode os.FileMode

	// Args holds rsync CLI options when UseRsync is enabled.
	Args args.RsyncArgs

	// UseRsync selects the rsync backend instead of the native copier.
	UseRsync bool

	// ScanDuration flushes a partial chunk when scanning pauses.
	ScanDuration time.Duration
	// FinderBinaryPath enables external `find -ls` scanning when set.
	FinderBinaryPath string

	// TaskSize limits concurrent chunk copy jobs.
	TaskSize int
	// ChunkSize is the target number of scanned entries per copy job.
	ChunkSize int
	// Retry configures chunk-level retry behavior for the selected backend.
	Retry args.RetryArgs
}

// Execute scans the source tree, runs chunk copies, and reports aggregate progress.
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

// prepareDirectory normalizes the source path and ensures the destination root exists.
func (r *Runner) prepareDirectory(sourcePath string, destinationPath string) (string, string, error) {
	// 아래 영역은 이제 host os 와 격리되었다.
	if err := util.EnsureNoSymlinkPath(sourcePath); err != nil {
		return "", "", fmt.Errorf("unsafe source path(%s): %w", sourcePath, err)
	}
	sourceStat, err := os.Stat(sourcePath)
	if err != nil {
		return "", "", fmt.Errorf("failed to get source file info(%s): %w", sourcePath, err)
	} else if sourceStat.IsDir() {
		// directory인 경우 slash 추가.
		sourcePath += pathSeperatorStr
	}

	// Destination roots are created with the configured private directory mode rather than inheriting broad source permissions.
	destinationFilemode := r.FileMode.Perm() | 0o700

	if err := util.EnsurePrivatePathPrefix(destinationPath); err != nil {
		return "", "", fmt.Errorf("unsafe destination directory(%s): %w", destinationPath, err)
	}
	if err := os.MkdirAll(destinationPath, destinationFilemode.Perm()); err != nil {
		return "", "", fmt.Errorf("failed to prepare destination directory(%s): %w", destinationPath, err)
	} else {
		util.InfoLog.Printf("directory %s created", destinationPath)
	}
	if err := os.Chmod(destinationPath, destinationFilemode.Perm()); err != nil {
		return "", "", fmt.Errorf("failed to apply destination directory mode(%s): %w", destinationPath, err)
	}
	if err := util.EnsurePrivatePath(destinationPath); err != nil {
		return "", "", fmt.Errorf("unsafe destination directory(%s): %w", destinationPath, err)
	}

	return sourcePath, destinationPath, nil
}
