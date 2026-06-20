// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"strings"
)

var (
	// InfoLog writes regular operational logs to stdout.
	InfoLog = log.Default()
	// ErrLog writes error logs to stderr.
	ErrLog = log.New(os.Stderr, "", log.LstdFlags)
)

// init points InfoLog at stdout so normal command output and info logs share the same descriptor.
func init() {
	InfoLog.SetOutput(os.Stdout)
}

// LogWriter adapts streamed command output into InfoLog line records.
type LogWriter struct {
}

// Write splits incoming bytes on newlines, skips blank lines, and re-emits each line through InfoLog.
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

// SetLogFlags keeps the stdout and stderr loggers on the same formatting flags.
func SetLogFlags(flag int) {
	InfoLog.SetFlags(flag)
	ErrLog.SetFlags(flag)
}
