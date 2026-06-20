//go:build !windows
// +build !windows

// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// EnsurePrivatePath rejects symlinks and other-user-writable components under a filesystem path.
func EnsurePrivatePath(path string) error {
	return ensurePrivatePath(path, false)
}

// EnsurePrivatePathPrefix rejects unsafe existing path components and allows a missing tail.
func EnsurePrivatePathPrefix(path string) error {
	return ensurePrivatePath(path, true)
}

// ensurePrivatePath walks a path from its absolute form so relative CLI paths are checked under the current directory.
func ensurePrivatePath(path string, allowMissingTail bool) error {
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve private path %q: %w", path, err)
	}
	current := filepath.VolumeName(cleanPath)
	if current == "" {
		current = string(os.PathSeparator)
	}
	trimmed := strings.TrimPrefix(cleanPath, current)
	lastPublicWritableDir := false
	for _, part := range strings.Split(trimmed, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if allowMissingTail && os.IsNotExist(err) {
				if lastPublicWritableDir {
					return fmt.Errorf("private path missing tail cannot be created below public writable directory(%s)", filepath.Dir(current))
				}
				return nil
			}
			return fmt.Errorf("inspect private path %q: %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("private path must not traverse symlink component %q", current)
		}
		if !info.IsDir() {
			lastPublicWritableDir = false
			continue
		}
		publicWritable := info.Mode().Perm()&0o022 != 0
		lastPublicWritableDir = publicWritable
		if publicWritable && info.Mode()&os.ModeSticky == 0 {
			return fmt.Errorf("private path component must not be group/world writable(%s)", current)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			continue
		}
		uid := int(stat.Uid)
		ownerCanWrite := info.Mode().Perm()&0o200 != 0
		if ownerCanWrite && uid != 0 && uid != os.Geteuid() {
			return fmt.Errorf("private path component owner uid %d can modify %s; want root or euid %d", uid, current, os.Geteuid())
		}
	}
	return nil
}
