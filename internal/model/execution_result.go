package model

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
)

type ExecutionResult struct {
	PID int
	Err error

	stderrLastLogLineStartIdx int
	stderrLastLogLines        [5]string
}

func (r *ExecutionResult) AppendLogLine(line string) {
	r.stderrLastLogLines[r.stderrLastLogLineStartIdx] = line
	r.stderrLastLogLineStartIdx = (r.stderrLastLogLineStartIdx + 1) % len(r.stderrLastLogLines)
}

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

func (r *ExecutionResult) HandleError() error {
	return r.exitResult()
}

func (r *ExecutionResult) exitResult() error {
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
	if exitcode == 0 {
		return nil
	}

	buf := &strings.Builder{}
	lastLoglines := r.LastLogLine()

	if len(lastLoglines) > 0 {
		buf.WriteString(", ")
	}
	if num := len(lastLoglines); num > 0 {
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

	if err := r.Err; err != nil {
		return fmt.Errorf("%w%s", err, buf.String())
	} else {
		return fmt.Errorf("%s", buf.String())
	}
}
