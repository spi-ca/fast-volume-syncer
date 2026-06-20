//go:build windows
// +build windows

// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnsurePrivatePath rejects symlinks under a filesystem path on Windows.
func EnsurePrivatePath(path string) error {
	return ensurePrivatePathWindows(path, false)
}

// EnsurePrivatePathPrefix rejects unsafe existing path components and allows a missing tail on Windows.
func EnsurePrivatePathPrefix(path string) error {
	return ensurePrivatePathWindows(path, true)
}

// ensurePrivatePathWindows walks an absolute path and rejects existing symlink components.
func ensurePrivatePathWindows(path string, allowMissingTail bool) error {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve private path %q: %w", path, err)
	}
	current := filepath.VolumeName(cleanPath)
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
			return fmt.Errorf("inspect private path %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("private path must not traverse symlink component %q", current)
		}
	}
	return nil
}
