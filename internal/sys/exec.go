package sys

import (
	"fmt"
	"os"
)

var (
	selfExecutablePath string
)

func init() {
	if exePath, err := os.Executable(); err != nil {
		panic(fmt.Errorf("failed to get self-path: %w", err))
	} else {
		selfExecutablePath = exePath
	}
}

func Executable() string {
	return selfExecutablePath
}
