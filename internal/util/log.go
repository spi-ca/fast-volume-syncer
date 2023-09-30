package util

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"strings"
)

var (
	InfoLog = log.Default()
	ErrLog  = log.New(os.Stderr, "", log.LstdFlags)
)

func init() {
	InfoLog.SetOutput(os.Stdout)
}

type LogWriter struct {
}

func (w LogWriter) Write(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, nil
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		trimmed := strings.TrimSpace(scanner.Text())
		if len(trimmed) < 1 {
			continue
		}

		if crIndex := strings.LastIndexByte(trimmed, '\r'); crIndex > 0 {
			trimmed = trimmed[crIndex:]
		}
		_ = InfoLog.Output(1, trimmed)
	}
	return len(b), nil
}

func SetLogFlags(flag int) {
	InfoLog.SetFlags(flag)
	ErrLog.SetFlags(flag)
}
