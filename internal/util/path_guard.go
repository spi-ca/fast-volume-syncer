// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsureNoSymlinkPath rejects existing symlink components in a full path.
func EnsureNoSymlinkPath(path string) error {
	return ensureNoSymlinkPath(path, true)
}

// EnsureNoSymlinkAncestors rejects symlink components in the parent directories of a path.
func EnsureNoSymlinkAncestors(path string) error {
	return ensureNoSymlinkPath(filepath.Dir(path), true)
}

// ensureNoSymlinkPath walks existing path components with Lstat and stops safely at missing tails.
func ensureNoSymlinkPath(path string, allowMissingTail bool) error {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", path, err)
	}
	current := filepath.VolumeName(cleanPath)
	if current == "" {
		current = string(os.PathSeparator)
	}
	trimmed := strings.TrimPrefix(cleanPath, current)
	for _, part := range strings.Split(trimmed, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if allowMissingTail && os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("inspect path %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("path must not traverse symlink component %q", current)
		}
	}
	return nil
}
