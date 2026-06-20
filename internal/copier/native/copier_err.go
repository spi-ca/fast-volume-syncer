// Package native copies scanned entries with direct filesystem operations.
package native

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
	ErrCopierDstNoSpace                = errors.New("destination is full")
)

// copierError adds source and destination paths to a backend copy failure.
type copierError struct {
	// srcPath is the fully qualified source path for the failed entry.
	srcPath string
	// dstPath is the fully qualified destination path for the failed entry.
	dstPath string
	// cause is the wrapped filesystem or classification error.
	cause error
}

// Error formats the failed copy pair together with the wrapped cause.
func (e copierError) Error() string {
	return fmt.Sprintf("failed to Copy(%s -> %s): %v", e.srcPath, e.dstPath, e.cause)
}

// Unwrap returns the underlying copy failure.
func (e copierError) Unwrap() error {
	return e.cause
}
