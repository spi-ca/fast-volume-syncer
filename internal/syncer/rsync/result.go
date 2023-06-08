package rsync

import (
	"fmt"
	"math"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/avast/retry-go"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

type exitCodeResult struct {
	Success   bool
	Retryable bool
	Message   string
}

var (
	exitCodeMap = map[int]exitCodeResult{
		-1:  {false, false, "unknown error"},
		0:   {true, false, "Success"},
		1:   {false, false, "Syntax or usage error"},
		2:   {false, false, "Protocol incompatibility"},
		3:   {false, false, "Errors selecting input/output files, dirs"},
		4:   {false, false, "Requested action not supported: an attempt was made to manipulate 64-bit files on a platform that cannot support them; or an option was specified that is supported by the client and not by the server."},
		5:   {false, true, "Error starting client-server protocol"},
		6:   {false, true, "Daemon unable to append to log-file"},
		10:  {false, true, "Error in socket I/O"},
		11:  {false, true, "Error in file I/O"},
		12:  {false, true, "Error in rsync protocol data stream"},
		13:  {false, false, "Errors with program diagnostics"},
		14:  {false, true, "Error in IPC code"},
		20:  {true, false, "Received SIGUSR1 or SIGINT"}, // successful failure
		21:  {false, false, "Some error returned by waitpid()"},
		22:  {false, true, "Error allocating core memory buffers"},
		23:  {false, false, "Partial transfer due to error"},
		24:  {true, false, "Partial transfer due to vanished source files"}, // successful failure
		25:  {true, true, "The --max-delete limit stopped deletions"},       // successful failure
		30:  {false, true, "Timeout in data send/receive"},
		35:  {false, true, "Timeout waiting for daemon connection"},
		127: {false, false, "It can mean you don't even have rsync binary installed on your system."},
		255: {false, true, "Rsync just passed exit code from a command it used to connect - typically SSH."},
	}
)

type result struct {
	startIdx      int
	lastFilenames [5]string
	total         int
	sent          int
	processing    int
	uptodate      int
	sentBytes     int64
	pid           int
	err           error
	started       time.Time

	stderrLastLogLineStartIdx int
	stderrLastLogLines        [5]string
}

func (r *result) appendLogLine(line string) {
	r.stderrLastLogLines[r.stderrLastLogLineStartIdx] = line
	r.stderrLastLogLineStartIdx = (r.stderrLastLogLineStartIdx + 1) % len(r.stderrLastLogLines)
}

func (r *result) lastLogLine() []string {
	loglines := make([]string, 0, len(r.stderrLastLogLines))
	for i := 0; i < len(r.stderrLastLogLines); i++ {
		logline := r.stderrLastLogLines[(r.stderrLastLogLineStartIdx+i)%len(r.stderrLastLogLines)]
		if len(logline) > 0 {
			loglines = append(loglines, logline)
		}
	}
	return loglines
}

func (r *result) appendFilename(filename string) {
	r.lastFilenames[r.startIdx] = filename
	r.startIdx = (r.startIdx + 1) % len(r.lastFilenames)
}

func (r *result) listFilename() []string {
	filenames := make([]string, 0, len(r.lastFilenames))
	for i := 0; i < len(r.lastFilenames); i++ {
		filename := r.lastFilenames[(r.startIdx+i)%len(r.lastFilenames)]
		if len(filename) > 0 {
			filenames = append(filenames, filename)
		}
	}
	return filenames
}

func (r *result) String() string {
	buf := &strings.Builder{}

	_, _ = fmt.Fprintf(buf, "rsync(%d)", r.pid)

	err, _ := r.rsyncExitResult()
	if err != nil {
		buf.WriteString(err.Error())
	}

	if !r.started.IsZero() {
		elapsed := time.Now().Sub(r.started)
		_, _ = fmt.Fprintf(buf, " in %2.2f ms", float32(elapsed.Microseconds())/1000)
		if r.sentBytes > 0 {
			buf.WriteString(" sent ")
			buf.WriteString(util.FileSizeIEC(r.sentBytes))
			bytesPerSeconds := int64(float64(r.sentBytes) / math.Max(elapsed.Seconds(), 0.001))
			if bytesPerSeconds > 0 {
				buf.WriteString("(")
				buf.WriteString(util.FileSizeIEC(bytesPerSeconds))
				buf.WriteString("/s)")
			}
		}
	} else if r.sentBytes > 0 {
		buf.WriteString(" sent ")
		buf.WriteString(util.FileSizeIEC(r.sentBytes))
	}
	if r.total > 0 {
		_, _ = fmt.Fprintf(buf, " total(%d) = sent(%d) + uptodate(%d) + untouched(%d), processing(%d)",
			r.total, r.sent, r.uptodate, r.total-r.sent-r.uptodate, r.processing,
		)
	}
	if err != nil {
		listFiles := r.listFilename()
		if num := len(listFiles); num > 0 {
			_, _ = fmt.Fprintf(buf, "\n=> last %d sent file", num)
			if num > 1 {
				buf.WriteByte('s')
			}
			buf.WriteString(" = [\n")
			for idx, filename := range listFiles {
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
	}

	return buf.String()
}

func (r *result) HandleError() error {
	err, retryable := r.rsyncExitResult()
	if err != nil && !retryable {
		err = retry.Unrecoverable(err)
	}
	return err
}

func (r *result) rsyncExitResult() (error, bool) {

	exitcode := 0
	if err := r.err; err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitcode = ws.ExitStatus()
		} else {
			exitcode = -1
		}
	}
	res := exitCodeMap[-1]
	if _res, ok := exitCodeMap[exitcode]; ok {
		res = _res
	}

	if res.Success {
		return nil, true
	}

	buf := &strings.Builder{}
	lastLoglines := r.lastLogLine()
	lastListFiles := r.listFilename()

	if len(lastLoglines) > 0 || len(lastListFiles) > 0 {
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
	if num := len(lastListFiles); num > 0 {
		_, _ = fmt.Fprintf(buf, "\n=> last %d sent file", num)
		if num > 1 {
			buf.WriteByte('s')
		}
		buf.WriteString(" = [\n")
		for idx, filename := range lastListFiles {
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
	if err := r.err; err != nil {
		return fmt.Errorf("%s: %w%s", res.Message, err, buf.String()), res.Retryable
	} else {
		return fmt.Errorf("%s%s", res.Message, buf.String()), res.Retryable
	}
}
