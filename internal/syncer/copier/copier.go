package copier

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/returns"

	"github.com/djherbis/times"
)

const (
	COMPARE_EQUAL int = iota
	COMPARE_SRC_NOT_EXIST
	COMPARE_DST_NOT_EXIST
	COMPARE_DIFFER
	COMPARE_DST_IS_NEWER
)

var (
	ErrCopierSrcNotExist               = errors.New("source file not exists")
	ErrCopierUptodate                  = errors.New("source file not exists")
	ErrCopierCopyFailed                = errors.New("failed to copy a file")
	ErrCopierProcessDiretoryFailed     = errors.New("failed to process a directory")
	ErrCopierProcessSymbolicLinkFailed = errors.New("failed to process a file entry")
	ErrCopierCompareFailed             = errors.New("failed to compare between two file paths")
	ErrCopierSkipped                   = errors.New("skipped file")
)

type copierError struct {
	srcPath string
	dstPath string
	cause   error
}

func (e copierError) Error() string {
	return fmt.Sprintf("failed to Copy(%s -> %s): %v", e.srcPath, e.dstPath, e.cause)
}

func (e copierError) Unwrap() error {
	return e.cause
}

type Copier struct {
	SourceRoot      string
	DestinationRoot string
	Umask           os.FileMode
	opIdx           uint64
}

func (t *Copier) copyNewFile(opIdx uint64, srcPath, dstPath string, mode os.FileMode) (int64, error) {
	dstDir := filepath.Dir(dstPath)
	src, err := os.OpenFile(srcPath, os.O_RDONLY, 0o644)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to open source file: %w; %w", ErrCopierCopyFailed, err),
		}
	}

	tm, err := times.StatFile(src)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get the source fileinfo :%w; %w", ErrCopierCopyFailed, err),
		}
	}
	defer src.Close()

	tmp, err := os.CreateTemp(dstDir, fmt.Sprintf(".tmp-%x-%d", int64(os.Getpid())^time.Now().Unix(), opIdx))
	if err != nil {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to create a tempfile :%w; %w", ErrCopierCopyFailed, err),
		}
	}

	tmpPath := tmp.Name()
	copied, err := io.Copy(tmp, src)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	} else {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierCopyFailed, err),
		}
	}

	err = os.Rename(tmpPath, dstPath)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to rename a file :%w; %w", ErrCopierCopyFailed, err),
		}
	}
	_ = tmp.Close()

	err = os.Chmod(dstPath, mode)
	if err != nil {
		_ = os.Remove(dstPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to change a filemode :%w; %w", ErrCopierCopyFailed, err),
		}
	}

	err = os.Chtimes(dstPath, tm.AccessTime(), tm.ModTime())
	if err != nil {
		_ = os.Remove(dstPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to update times :%w; %w", ErrCopierCopyFailed, err),
		}
	}
	return copied, nil
}

func (t *Copier) copyExistingFile(opIdx uint64, srcPath, dstPath string, mode os.FileMode) (int64, error) {
	dstDir := filepath.Dir(dstPath)
	src, err := os.OpenFile(srcPath, os.O_RDONLY, 0o644)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to open source file: %w; %w", ErrCopierCopyFailed, err),
		}
	}

	tm, err := times.StatFile(src)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get the source fileinfo :%w; %w", ErrCopierCopyFailed, err),
		}
	}
	defer src.Close()

	tmp, err := os.CreateTemp(dstDir, fmt.Sprintf(".tmp-%x-%d", int64(os.Getpid())^time.Now().Unix(), opIdx))
	if err != nil {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to create a tempfile :%w; %w", ErrCopierCopyFailed, err),
		}
	}

	tmpPath := tmp.Name()
	copied, err := io.Copy(tmp, src)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	} else {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierCopyFailed, err),
		}
	}

	err = os.Remove(dstPath)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to remove the destination :%w; %w", ErrCopierCopyFailed, err),
		}
	}

	err = os.Rename(tmpPath, dstPath)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to rename a file :%w; %w", ErrCopierCopyFailed, err),
		}
	}
	_ = tmp.Close()

	err = os.Chmod(dstPath, mode)
	if err != nil {
		_ = os.Remove(dstPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to change a filemode :%w; %w", ErrCopierCopyFailed, err),
		}
	}

	err = os.Chtimes(dstPath, tm.AccessTime(), tm.ModTime())
	if err != nil {
		_ = os.Remove(dstPath)
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to update times :%w; %w", ErrCopierCopyFailed, err),
		}
	}

	return copied, nil
}

func (t *Copier) compareFile(opIdx uint64, srcPath string, dstPath string, srcSize int64) (int, error) {
	destMode, err := os.Lstat(dstPath)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return COMPARE_DST_NOT_EXIST, nil
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to open destination file: %w; %w", ErrCopierCompareFailed, err),
		}
	}

	srcTm, err := times.Lstat(srcPath)
	if err == nil {
		// do nothing
	} else if os.IsNotExist(err) {
		return COMPARE_SRC_NOT_EXIST, nil
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get a source mtime :%w; %w", ErrCopierCompareFailed, err),
		}
	}

	dstTm, err := times.Lstat(dstPath)
	if err != nil {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get a destination mtime :%w; %w", ErrCopierCompareFailed, err),
		}
	}

	if srcTm.ModTime().Before(dstTm.ModTime()) {
		return COMPARE_DST_IS_NEWER, nil
	}

	if srcSize != destMode.Size() {
		return COMPARE_DIFFER, nil
	} else {
		return COMPARE_EQUAL, nil
	}
}

func (t *Copier) copyRegularFile(opIdx uint64, srcPath string, dstPath string, srcSize int64, dstMode os.FileMode) (int64, error) {
	differ, err := t.compareFile(opIdx, srcPath, dstPath, srcSize)
	if err != nil {
		return 0, err
	}
	switch differ {
	case COMPARE_EQUAL:
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   ErrCopierUptodate,
		}
	case COMPARE_SRC_NOT_EXIST:
		// source file disappears..
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   errors.Join(ErrCopierSrcNotExist, err),
		}
	case COMPARE_DIFFER:
		return t.copyExistingFile(opIdx, srcPath, dstPath, dstMode)
	case COMPARE_DST_NOT_EXIST:
		return t.copyNewFile(opIdx, srcPath, dstPath, dstMode)
	case COMPARE_DST_IS_NEWER:
		util.ErrLog.Printf("[Copier op:%d]destination(%s) is newer than source(%s)", opIdx, dstPath, dstMode)
		return 0, nil
	default:
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to compare file, returns %d :%w", differ, ErrCopierCopyFailed),
		}
	}
}

func (t *Copier) processDirectory(opIdx uint64, srcPath string, dstPath string, dstMode os.FileMode) error {
	destExists := false

	destMode, err := os.Lstat(dstPath)

	if err == nil {
		if destExists = destMode.IsDir(); !destExists {
			// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
			if err = os.RemoveAll(dstPath); err != nil {
				return &copierError{
					srcPath: srcPath,
					dstPath: dstPath,
					cause:   fmt.Errorf("failed to cleanup: %w; %w", ErrCopierProcessDiretoryFailed, err),
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get destination info: %w; %w", ErrCopierProcessDiretoryFailed, err),
		}
	}

	if destExists {
		// directory path 확보상태(이미 생성됨)
		if destMode.Mode() == dstMode {
			return &copierError{
				srcPath: srcPath,
				dstPath: dstPath,
				cause:   ErrCopierUptodate,
			}
		}

		err = os.Chmod(dstPath, dstMode)
		if err != nil {
			return &copierError{
				srcPath: srcPath,
				dstPath: dstPath,
				cause:   fmt.Errorf("failed to change filemode(%s): %w; %w", dstMode, ErrCopierProcessDiretoryFailed, err),
			}
		}
	} else {
		// directory path 확보상태(비어있음)
		err = os.MkdirAll(dstPath, dstMode)
		if err != nil {
			return &copierError{
				srcPath: srcPath,
				dstPath: dstPath,
				cause:   fmt.Errorf("failed to make a directory(%s): %w; %w", dstMode, ErrCopierProcessDiretoryFailed, err),
			}
		}
	}
	return nil
}

func (t *Copier) processSymbolicLink(opIdx uint64, srcPath string, dstPath string, linkPath string) error {
	if destMode, err := os.Lstat(dstPath); err == nil {
		if destMode.Mode().Type()&fs.ModeSymlink != 0 {
			destLinkPath, readLinkErr := os.Readlink(dstPath)
			if readLinkErr == nil && destLinkPath == linkPath {
				// 대상파일의 링크정보가 일치함
				return &copierError{
					srcPath: srcPath,
					dstPath: dstPath,
					cause:   ErrCopierUptodate,
				}
			}
		}

		// 대상 path가 symlink mode가 아닌 경우 대상을 날린다.
		if err = os.RemoveAll(dstPath); err != nil {
			return &copierError{
				srcPath: srcPath,
				dstPath: dstPath,
				cause:   fmt.Errorf("failed to remove destination file: %w; %w", ErrCopierProcessSymbolicLinkFailed, err),
			}
		}
	} else if !os.IsNotExist(err) {
		return &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get destination info: %w; %w", ErrCopierProcessSymbolicLinkFailed, err),
		}
	}

	// directory보다 링크가 먼저 온 case
	dirPath := filepath.Dir(dstPath)
	if destMode, err := os.Lstat(dirPath); err == nil {
		if !destMode.IsDir() {
			// 대상 path가 directory mode가 아닌 경우 대상을 날린다.
			if err = os.RemoveAll(dirPath); err != nil {
				return &copierError{
					srcPath: srcPath,
					dstPath: dstPath,
					cause:   fmt.Errorf("failed to cleanup: %w; %w", ErrCopierProcessSymbolicLinkFailed, err),
				}
			}
		}
	} else if !os.IsNotExist(err) {
		return &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to get destination info: %w; %w", ErrCopierProcessSymbolicLinkFailed, err),
		}
	} else if err = os.MkdirAll(dirPath, 0o755); err != nil {
		return &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to make a directory: %w; %w", ErrCopierProcessSymbolicLinkFailed, err),
		}
	}

	if err := os.Symlink(linkPath, dstPath); err != nil {
		return &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("failed to make a symbolic link: %w; %w", ErrCopierProcessSymbolicLinkFailed, err),
		}
	}

	return nil
}

func (t *Copier) routeFileByTypes(opIdx uint64, srcInfo returns.Fileinfo) (int64, error) {
	srcMode := srcInfo.Mode
	dstMode := srcInfo.Mode.Perm() | t.Umask

	srcPath := filepath.Join(t.SourceRoot, srcInfo.Path)
	dstPath := filepath.Join(t.DestinationRoot, srcInfo.Path)

	if srcMode.IsDir() {
		return 0, t.processDirectory(opIdx, srcPath, dstPath, dstMode|0o100)
	} else if srcMode.Type()&fs.ModeSymlink != 0 {
		return 0, t.processSymbolicLink(opIdx, srcPath, dstPath, srcInfo.SymlinkPath)
	} else if srcMode.IsRegular() {
		return t.copyRegularFile(opIdx, srcPath, dstPath, srcInfo.Size, dstMode)
	} else {
		return 0, &copierError{
			srcPath: srcPath,
			dstPath: dstPath,
			cause:   fmt.Errorf("unexpected filemode(%s) :,%w", srcMode, ErrCopierSkipped),
		}
	}
}

func (t *Copier) Execute(ctx context.Context, fileList []returns.Fileinfo) error {
	opIdx := atomic.AddUint64(&t.opIdx, 1)
	filenameSet := make(map[string]int)
	for idx, info := range fileList {
		filenameSet[info.Path] = idx
	}

	res := &result{total: len(fileList), started: time.Now()}

	bar := progressbar.NewOptions(res.total,
		progressbar.OptionSetWriter(util.LogWriter{}),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(time.Second),
		progressbar.OptionSetItsString("op"),
		progressbar.OptionSetDescription(fmt.Sprintf("[Copier op:%d]", opIdx)),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "-",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
	defer bar.Close()

forLoop:
	for _, entry := range fileList {
		copied, err := t.routeFileByTypes(opIdx, entry)
		_ = bar.Add(1)
		res.appendFilename(entry.Path)
		res.sentBytes += copied
		if err == nil {
			res.sent++
		} else if errors.Is(err, ErrCopierUptodate) {
			res.uptodate++
		} else if errors.Is(err, ErrCopierSrcNotExist) {
			util.ErrLog.Printf("[Copier op:%d]%v", opIdx, err)
		} else if errors.Is(err, ErrCopierSkipped) {
			res.skipped++
			util.ErrLog.Printf("[Copier op:%d]%v", opIdx, err)
		} else {
			res.errs = append(res.errs, err)
		}

		select {
		case <-ctx.Done():
			break forLoop
		default:
		}
	}

	err := res.HandleError()
	if err == nil {
		util.InfoLog.Print(res)
	}
	return err
}
