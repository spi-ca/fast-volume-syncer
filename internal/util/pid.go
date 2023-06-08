package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
)

func AcquirePidFile(filename string) (func(), error) {
	dirpath := filepath.Dir(filename)
	err := os.MkdirAll(dirpath, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to make dir of pidfile(%s): %w", dirpath, err)
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to crated pidfile(%s): %w", filename, err)
	}

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
		_ = f.Close()
		_ = os.Remove(filename)
	}, nil
}

func ReadPidFile(filename string) (int, error) {
	f, err := os.OpenFile(filename, os.O_RDONLY, 0o644)
	if err != nil {
		return -1, fmt.Errorf("failed to crated pidfile(%s): %w", filename, err)
	}

	defer f.Close()

	reader := bufio.NewReader(f)

	line, isPrefix, err := reader.ReadLine()
	if err != nil {
		return -1, fmt.Errorf("failed to read pidfile(%s): %w", filename, err)
	} else if isPrefix {
		return -1, fmt.Errorf("first line is too long, pidfile(%s)", filename)
	}

	return int(SimpleStrconv(line)), nil
}
