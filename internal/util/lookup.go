// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"os"
	"path/filepath"
)

// LookupBinary resolves a configured executable to an absolute path and logs an empty result when it cannot be found.
func LookupBinary(givenPath string) string {
	if len(givenPath) < 1 {
		return ""
	}

	if filepath.IsAbs(givenPath) || filepath.Dir(givenPath) != "." {
		candidate, err := filepath.Abs(givenPath)
		if err != nil {
			ErrLog.Printf("binary(%s) path invalid: %v", givenPath, err)
			return ""
		}
		if info, err := os.Stat(candidate); err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
			ErrLog.Printf("binary(%s) not executable", givenPath)
			return ""
		}
		return candidate
	}
	for _, dir := range filepath.SplitList(TrustedPath()) {
		candidate := filepath.Join(dir, givenPath)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode().Perm()&0o111 != 0 {
			return candidate
		}
	}
	ErrLog.Printf("binary(%s) not found", givenPath)
	return ""
}
