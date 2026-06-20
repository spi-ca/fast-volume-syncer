// Package returns defines result objects shared by worker, sync, and mount flows.
package returns

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// ExecutionResult tracks process completion data and the last ten stderr log lines.
type ExecutionResult struct {
	// PID identifies the child process that produced this result.
	PID int
	// Err stores the process error returned by exec or wait handling.
	Err error

	// stderrLastLogLineStartIdx points at the oldest slot in the circular log buffer.
	stderrLastLogLineStartIdx int
	// stderrLastLogLines stores a ring buffer of the most recent stderr lines.
	stderrLastLogLines [10]string
}

// AppendLogLine records one stderr line into the fixed-size circular buffer.
func (r *ExecutionResult) AppendLogLine(line string) {
	r.stderrLastLogLines[r.stderrLastLogLineStartIdx] = line
	r.stderrLastLogLineStartIdx = (r.stderrLastLogLineStartIdx + 1) % len(r.stderrLastLogLines)
}

// LastLogLine returns the buffered stderr lines in oldest-to-newest order.
func (r *ExecutionResult) LastLogLine() []string {
	loglines := make([]string, 0, len(r.stderrLastLogLines))
	for i := 0; i < len(r.stderrLastLogLines); i++ {
		logline := r.stderrLastLogLines[(r.stderrLastLogLineStartIdx+i)%len(r.stderrLastLogLines)]
		if len(logline) > 0 {
			loglines = append(loglines, logline)
		}
	}
	return loglines
}

// HandleError converts Err and buffered stderr lines into the final reported error.
func (r *ExecutionResult) HandleError() error {
	exitcode := 0
	if err := r.Err; err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitcode = ws.ExitStatus()
		} else {
			exitcode = -1
		}
	}

	buf := &strings.Builder{}
	lastLoglines := r.LastLogLine()

	if num := len(lastLoglines); num > 0 {
		buf.WriteString(", ")
		_, _ = fmt.Fprintf(buf, "\n=> last %d log", num)
		if num > 1 {
			buf.WriteByte('s')
		}
		buf.WriteString(" = [\n")
		for idx, filename := range lastLoglines {
			buf.WriteString("\t'")
			buf.WriteString(filename)
			buf.WriteByte('\'')
			if idx+1 < num {
				buf.WriteByte(',')
			}
			buf.WriteByte('\n')
		}
		buf.WriteByte(']')
	}

	if exitcode == 0 {
		if buf.Len() > 0 {
			util.ErrLog.Print(buf.String())
		}
		return nil
	} else if err := r.Err; err != nil {
		return fmt.Errorf("%w%s", err, buf.String())
	} else {
		return fmt.Errorf("%s", buf.String())
	}
}
