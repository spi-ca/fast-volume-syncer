// Package selector parses copy-entry CSV rows and fans them out to sync workers.
package selector

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// AcquirePidFile locks the daemon pid file, writes the current pid, and returns cleanup.
func AcquirePidFile(filename string) (func(), error) {
	dirpath := filepath.Dir(filename)
	if err := ensureDaemonDirectory(dirpath); err != nil {
		return nil, err
	}
	if err := validatePidDirectory(dirpath); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("failed to crated pidfile(%s): %w", filename, err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to stat pidfile(%s): %w", filename, err)
	} else if !info.Mode().IsRegular() {
		_ = f.Close()
		return nil, fmt.Errorf("pidfile must be a regular file(%s)", filename)
	} else if err := validateOwnedPrivatePath(filename, info); err != nil {
		_ = f.Close()
		return nil, err
	}
	pidStat, _ := info.Sys().(*syscall.Stat_t)

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to lock pidfile(%s): %w", filename, err)
	}

	_, err = f.Seek(0, io.SeekCurrent)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to seek pidfile(%s): %w", filename, err)
	}
	err = f.Truncate(0)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to truncate pidfile(%s): %w", filename, err)
	}
	_, err = f.WriteString(fmt.Sprintf("%d\n", os.Getpid()))
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("failed to write pidfile(%s): %w", filename, err)
	}
	return func() {
		if pidStat != nil {
			if currentInfo, err := os.Stat(filename); err == nil {
				if currentStat, ok := currentInfo.Sys().(*syscall.Stat_t); ok && currentStat.Dev == pidStat.Dev && currentStat.Ino == pidStat.Ino {
					_ = os.Remove(filename)
				}
			}
		}
		_ = f.Close()
	}, nil
}

// ReadPidFile reads the first line of the pid file and parses it as a process id.
func ReadPidFile(filename string) (int, error) {
	if err := validatePidDirectory(filepath.Dir(filename)); err != nil {
		return -1, err
	}
	f, err := os.OpenFile(filename, os.O_RDONLY|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return -1, fmt.Errorf("failed to crated pidfile(%s): %w", filename, err)
	}
	if info, err := f.Stat(); err != nil {
		_ = f.Close()
		return -1, fmt.Errorf("failed to stat pidfile(%s): %w", filename, err)
	} else if !info.Mode().IsRegular() {
		_ = f.Close()
		return -1, fmt.Errorf("pidfile must be a regular file(%s)", filename)
	} else if err := validateOwnedPrivatePath(filename, info); err != nil {
		_ = f.Close()
		return -1, err
	}

	defer f.Close()

	reader := bufio.NewReader(f)

	line, isPrefix, err := reader.ReadLine()
	if err != nil {
		return -1, fmt.Errorf("failed to read pidfile(%s): %w", filename, err)
	} else if isPrefix {
		return -1, fmt.Errorf("first line is too long, pidfile(%s)", filename)
	}

	return int(util.SimpleStrconv(line)), nil
}

// ensureDaemonDirectory creates a daemon control directory one component at a time without following symlinks.
func ensureDaemonDirectory(dirpath string) error {
	absDir, err := filepath.Abs(dirpath)
	if err != nil {
		return fmt.Errorf("failed to resolve daemon directory(%s): %w", dirpath, err)
	}
	current := filepath.VolumeName(absDir)
	if current == "" {
		current = string(os.PathSeparator)
	}
	trimmed := strings.TrimPrefix(absDir, current)
	for _, part := range strings.Split(trimmed, string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to inspect daemon directory(%s): %w", current, err)
			}
			if err := os.Mkdir(current, 0o700); err != nil && !os.IsExist(err) {
				return fmt.Errorf("failed to make daemon directory(%s): %w", current, err)
			}
			info, err = os.Lstat(current)
			if err != nil {
				return fmt.Errorf("failed to inspect daemon directory(%s): %w", current, err)
			}
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("daemon directory must not contain symlinks(%s)", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("daemon directory ancestor must be a directory(%s)", current)
		}
		if err := validateOwnedPrivatePath(current, info); err != nil {
			return err
		}
	}
	return nil
}

// validatePidAncestors rejects symlinked existing ancestors before creating a pid-file directory.
func validatePidAncestors(dirpath string) error {
	absDir, err := filepath.Abs(dirpath)
	if err != nil {
		return fmt.Errorf("failed to resolve pid directory(%s): %w", dirpath, err)
	}
	current := filepath.VolumeName(absDir)
	if current == "" {
		current = string(os.PathSeparator)
	}
	trimmed := strings.TrimPrefix(absDir, current)
	for _, part := range strings.Split(trimmed, string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return fmt.Errorf("failed to inspect pid directory ancestor(%s): %w", current, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("pid directory must not contain symlinks(%s)", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("pid directory ancestor must be a directory(%s)", current)
		}
		if err := validateOwnedPrivatePath(current, info); err != nil {
			return err
		}
	}
	return nil
}

// validatePidDirectory rejects symlinked or shared pid-file parent directories.
func validatePidDirectory(dirpath string) error {
	absDir, err := filepath.Abs(dirpath)
	if err != nil {
		return fmt.Errorf("failed to resolve pid directory(%s): %w", dirpath, err)
	}
	resolvedDir, err := filepath.EvalSymlinks(absDir)
	if err != nil {
		return fmt.Errorf("failed to resolve pid directory symlinks(%s): %w", dirpath, err)
	}
	if filepath.Clean(absDir) != filepath.Clean(resolvedDir) {
		return fmt.Errorf("pid directory must not contain symlinks(%s)", dirpath)
	}
	info, err := os.Stat(absDir)
	if err != nil {
		return fmt.Errorf("failed to stat pid directory(%s): %w", dirpath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("pid directory must be a directory(%s)", dirpath)
	}
	if err := validateOwnedPrivatePath(absDir, info); err != nil {
		return err
	}
	return nil
}

// validateOwnedPrivatePath rejects shared or foreign-owned daemon control paths.
func validateOwnedPrivatePath(path string, info os.FileInfo) error {
	if info.Mode().Perm()&0o022 != 0 && info.Mode()&os.ModeSticky == 0 {
		return fmt.Errorf("daemon control path must not be group/world writable(%s)", path)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	uid := int(stat.Uid)
	if uid != 0 && uid != os.Geteuid() {
		return fmt.Errorf("daemon control path owner uid %d can modify %s; want root or euid %d", uid, path, os.Geteuid())
	}
	return nil
}
