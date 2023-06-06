package common

import (
	"bufio"
	"bytes"
	"log"
	"strings"
)

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
		_ = log.Output(1, trimmed)
	}
	return len(b), nil
}
