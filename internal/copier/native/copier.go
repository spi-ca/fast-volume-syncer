package native

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/avast/retry-go"

	"github.com/schollz/progressbar/v3"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/args"
	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"

	"github.com/djherbis/times"
)

const (
	compareEqual int = iota
	compareSrcNotExist
	compareDstNotExist
	compareDiffer
	compareDstIsNewer
)

type Copier struct {
	SourceRoot      string
	DestinationRoot string
	FileMode        os.FileMode
	Retry           args.RetryArgs
	chunkIdx        uint64
}

func (c *Copier) Execute(ctx context.Context, fileList []returns.Fileinfo) (result returns.IOResult, err error) {
	var (
		chunkIdx = atomic.AddUint64(&c.chunkIdx, 1)
	)

	if c.Retry.Attempts <= 0 {
		return c.execute(ctx, chunkIdx, fileList)
	}

	retryOptionArgs := c.Retry.Assemble(ctx)

	return result, retry.Do(func() (retryErr error) {
		result, retryErr = c.execute(ctx, chunkIdx, fileList)
		return
	}, retryOptionArgs...)
}

func (c *Copier) execute(ctx context.Context, chunkIdx uint64, fileList []returns.Fileinfo) (returns.IOResult, error) {
	res := &result{chunkIdx: chunkIdx, total: len(fileList), started: time.Now()}
	defer res.markEnd()

	bar := progressbar.NewOptions(res.total,
		progressbar.OptionSetWriter(util.LogWriter{}),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(time.Second),
		progressbar.OptionSetItsString("op"),
		progressbar.OptionSetDescription(fmt.Sprintf("[chk:%d]\t", chunkIdx)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "-",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	defer bar.Close()

	unrecoverable := false
forLoop:
	for _, entry := range fileList {
		copied, err := c.routeFileByTypes(ctx, chunkIdx, entry)
		_ = bar.Add(1)
		res.appendFilename(entry.Path)
		if err == nil {
			res.addTypeCount(entry.Mode)

			if copied >= 0 {
				res.sentBytes += copied
				res.sent++
			} else {
				res.processed++
			}
		} else if errors.Is(err, context.Canceled) {
			err = nil
			break forLoop
		} else if errors.Is(err, ErrCopierUptodate) {
			res.addTypeCount(entry.Mode)
			res.uptodate++
		} else if errors.Is(err, ErrCopierSrcNotExist) {
			res.disappeared++
		} else if errors.Is(err, ErrCopierSkipped) {
			res.addTypeCount(entry.Mode)
			res.skipped++
		} else if errors.Is(err, ErrCopierDstNoSpace) {
			res.errs = append(res.errs, err)
			unrecoverable = true
			break forLoop
		} else {
			res.errs = append(res.errs, err)
		}
	}

	err := res.HandleError()
	if err == nil {
		util.InfoLog.Print(res)
	} else if unrecoverable {
		err = retry.Unrecoverable(err)
	}
	return res, err
}

func (c *Copier) routeFileByTypes(ctx context.Context, chunkIdx uint64, srcInfo returns.Fileinfo) (int64, error) {

	var (
		srcMode = srcInfo.Mode
		dstMode = srcInfo.Mode.Perm() | c.FileMode.Perm()

		srcPath = filepath.Join(c.SourceRoot, srcInfo.Path)
		dstPath = filepath.Join(c.DestinationRoot, srcInfo.Path)

		copiedBytes int64 = -1
		err         error
	)

	if srcMode.IsDir() {
		err = c.processDirectory(dstPath, dstMode|0o100)
	} else if srcMode.Type()&fs.ModeSymlink != 0 {
		err = c.processSymbolicLink(srcInfo.SymlinkPath, dstPath)
	} else if srcMode.IsRegular() {
		copiedBytes, err = c.copyRegularFile(ctx, chunkIdx, srcPath, dstPath, dstMode)
	} else {
		err = fmt.Errorf("unexpected filemode(%s) :,%w", srcMode, ErrCopierSkipped)
	}
	if err != nil {
		return copiedBytes, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   err,
		}
	} else {
		return copiedBytes, nil
	}
}

func (c *Copier) processDirectory(dstPath string, dstMode os.FileMode) error {
	if dstPath == c.DestinationRoot {
		// 자기자신은 무시하자
		return nil
	}

	destExists := false

	existDstMode, err := os.Lstat(dstPath)

	if err == nil {
		if destExists = existDstMode.IsDir(); !destExists {
			// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
			err = os.RemoveAll(dstPath)
			if err != nil {
				return fmt.Errorf("failed to cleanup: %w; %w", ErrCopierProcessDiretoryFailed, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to get destination info: %w; %w", ErrCopierProcessDiretoryFailed, err)
	}

	if destExists {
		// directory path 확보상태(이미 생성됨)
		if existDstMode.Mode() == dstMode {
			return ErrCopierUptodate
		}

		err = os.Chmod(dstPath, dstMode)
		if err != nil {
			return fmt.Errorf("failed to change filemode(%s): %w; %w", dstMode, ErrCopierProcessDiretoryFailed, err)

		}
	} else {
		// directory path 확보상태(비어있음)
		err = os.MkdirAll(dstPath, dstMode)
		if err != nil {
			return fmt.Errorf("failed to make a directory(%s): %w; %w", dstMode, ErrCopierProcessDiretoryFailed, err)
		}
	}
	return nil
}

func (c *Copier) processSymbolicLink(linkPath, dstPath string) error {
	if dstMode, err := os.Lstat(dstPath); err == nil {
		if dstMode.Mode().Type()&fs.ModeSymlink != 0 {
			existDstLinkPath, readLinkErr := os.Readlink(dstPath)
			if readLinkErr == nil && existDstLinkPath == linkPath {
				// 대상파일의 링크정보가 일치함
				return ErrCopierUptodate
			}
		}

		// 대상 path가 symlink mode가 아닌 경우 대상을 날린다.
		err = os.RemoveAll(dstPath)
		if err != nil {
			return fmt.Errorf("failed to remove destination file: %w; %w", ErrCopierProcessSymbolicLinkFailed, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to get destination info: %w; %w", ErrCopierProcessSymbolicLinkFailed, err)
	} else if err = c.makeParentsExist(dstPath); err != nil { // 자식이 없다는것은 부모도 없을 수 있다는 의미.
		return err
	}

	if err := os.Symlink(linkPath, dstPath); err != nil {
		return fmt.Errorf("failed to make a symbolic link: %w; %w", ErrCopierProcessSymbolicLinkFailed, err)
	}

	return nil
}

func (c *Copier) copyRegularFile(ctx context.Context, chunkIdx uint64, srcPath string, dstPath string, dstMode os.FileMode) (int64, error) {
	differ, err := c.compareFile(srcPath, dstPath)
	if err != nil {
		return 0, err
	}
	switch differ {
	case compareEqual:
		return 0, ErrCopierUptodate
	case compareSrcNotExist:
		// source file disappears..
		return 0, errors.Join(ErrCopierSrcNotExist, err)
	case compareDiffer:
		return c.copyFile(ctx, chunkIdx, srcPath, dstPath, dstMode, true)
	case compareDstNotExist:
		return c.copyFile(ctx, chunkIdx, srcPath, dstPath, dstMode, false)
	case compareDstIsNewer:
		util.ErrLog.Printf("[chk:%d]destination(%s) is newer than source(%s)", chunkIdx, dstPath, dstMode)
		return 0, nil
	default:
		return 0, fmt.Errorf("failed to compare file, returns %d :%w", differ, ErrCopierCopyFailed)
	}
}

func (c *Copier) compareFile(srcPath string, dstPath string) (int, error) {
	srcMode, err := os.Lstat(srcPath)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return compareSrcNotExist, nil
	} else {
		return 0, fmt.Errorf("failed to open source file: %w; %w", ErrCopierCompareFailed, err)
	}
	dstMode, err := os.Lstat(dstPath)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return compareDstNotExist, nil
	} else {
		return 0, fmt.Errorf("failed to open destination file: %w; %w", ErrCopierCompareFailed, err)
	}

	srcTm := times.Get(srcMode)
	dstTm := times.Get(dstMode)

	if srcMode.Size() != dstMode.Size() {
		return compareDiffer, nil
	}

	if offset := srcTm.ModTime().Sub(dstTm.ModTime()); offset < 0 {
		return compareDiffer, nil
	} else if offset < 0 {
		return compareDstIsNewer, nil
	} else {
		return compareEqual, nil
	}
}

func (c *Copier) copyFile(parentCtx context.Context, chunkIdx uint64, srcPath, dstPath string, mode os.FileMode, dstExists bool) (int64, error) {
	dstDir := filepath.Dir(dstPath)
	src, err := os.OpenFile(srcPath, os.O_RDONLY, 0o644)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, errors.Join(ErrCopierSrcNotExist, err)
	} else {
		return 0, fmt.Errorf("failed to open source file: %w; %w", ErrCopierCopyFailed, err)
	}

	tm, err := times.StatFile(src)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, errors.Join(ErrCopierSrcNotExist, err)
	} else {
		return 0, fmt.Errorf("failed to get the source fileinfo :%w; %w", ErrCopierCopyFailed, err)
	}
	defer src.Close()

	if !dstExists {
		if err = c.makeParentsExist(dstPath); err != nil {
			// 자식이 없다는것은 부모도 없을 수 있다는 의미.
			return 0, err
		}
	}

	tmp, err := os.CreateTemp(dstDir, fmt.Sprintf(".tmp-%x-%d", int64(os.Getpid())^time.Now().Unix(), chunkIdx))
	if err == nil {
		// do nothing
	} else if errors.Is(err, syscall.ENOSPC) {
		return 0, errors.Join(ErrCopierDstNoSpace, err)
	} else {
		return 0, fmt.Errorf("failed to create a tempfile :%w; %w", ErrCopierCopyFailed, err)
	}

	tmpPath := tmp.Name()
	var copied int64
	ctx, causeFunc := context.WithCancelCause(context.Background())
	go func(copiedPtr *int64) {
		copiedBytes, copyErr := io.Copy(tmp, src)
		// do nothing
		*copiedPtr = copiedBytes
		causeFunc(copyErr)
	}(&copied)
	select {
	case <-parentCtx.Done():
		err = parentCtx.Err()
	case <-ctx.Done():
		cancelReason := context.Cause(ctx)
		if errors.Is(cancelReason, context.Canceled) {
			break
		} else if os.IsNotExist(cancelReason) {
			err = errors.Join(ErrCopierSrcNotExist, err)
		} else if errors.Is(err, syscall.ENOSPC) {
			return 0, errors.Join(ErrCopierDstNoSpace, err)
		} else {
			err = errors.Join(ErrCopierCopyFailed, err)
		}
	}
	_ = tmp.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return 0, err
	}

	if dstExists {
		err = os.RemoveAll(dstPath)
		if err != nil {
			_ = os.Remove(tmpPath)
			return 0, fmt.Errorf("failed to remove the destination :%w; %w", ErrCopierCopyFailed, err)
		}
	}

	err = os.Rename(tmpPath, dstPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return 0, fmt.Errorf("failed to rename a file :%w; %w", ErrCopierCopyFailed, err)
	}

	err = os.Chmod(dstPath, mode)
	if err != nil {
		_ = os.Remove(dstPath)
		return 0, fmt.Errorf("failed to change a filemode :%w; %w", ErrCopierCopyFailed, err)
	}

	err = os.Chtimes(dstPath, tm.AccessTime(), tm.ModTime())
	if err != nil {
		_ = os.Remove(dstPath)
		return 0, fmt.Errorf("failed to update times :%w; %w", ErrCopierCopyFailed, err)
	}
	return copied, nil
}

func (c *Copier) makeParentsExist(dstPath string) error {
	dirPath := filepath.Dir(dstPath)
	if existDstMode, err := os.Lstat(dirPath); err == nil {
		if !existDstMode.IsDir() {
			// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
			err = os.RemoveAll(dirPath)
			if err != nil {
				return fmt.Errorf("failed to cleanup: %w; %w", ErrCopierProcessSymbolicLinkFailed, err)
			}
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to get destination info: %w; %w", ErrCopierProcessSymbolicLinkFailed, err)
	} else if err = os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("failed to make a directory: %w; %w", ErrCopierProcessSymbolicLinkFailed, err)
	}
	return nil
}
