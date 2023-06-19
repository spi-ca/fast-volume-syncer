package copier

import (
	"errors"
	"fmt"
)

var (
	ErrCopierSrcNotExist               = errors.New("source file not exists")
	ErrCopierUptodate                  = errors.New("destination is same")
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
