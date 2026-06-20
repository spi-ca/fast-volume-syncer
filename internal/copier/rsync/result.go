// Package rsync copies file chunks by driving the rsync CLI.
package rsync

import (
	"fmt"
	"io/fs"
	"math"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/avast/retry-go"

	"amuz.es/src/spi-ca/fast-volume-syncer/internal/util"
)

// exitCodeResult describes how a specific rsync exit code should be treated.
type exitCodeResult struct {
	// Success reports whether the exit code should be treated as success.
	Success bool
	// Retryable reports whether chunk-level retry should be allowed.
	Retryable bool
	// Message is the human-readable rsync exit-code explanation.
	Message string
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
		20:  {false, false, "Received SIGUSR1 or SIGINT"},
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

// result records rsync chunk progress, exit-code context, and recent activity.
type result struct {
	// chunkIdx identifies the chunk in logs and error messages.
	chunkIdx uint64
	// startIdx is the ring-buffer cursor for lastFilenames.
	startIdx int
	// lastFilenames keeps the most recent paths reported by rsync.
	lastFilenames [10]string
	// total is the number of scheduled entries in the chunk.
	total int

	// files counts regular-file entries confirmed by stdout parsing.
	files int
	// links counts symbolic-link entries confirmed by stdout parsing.
	links int
	// directories counts directory entries confirmed by stdout parsing.
	directories int

	// sent counts entries rsync reported as transferred.
	sent int
	// processed counts stdout lines that did not map back to the input set.
	processed int
	// uptodate counts entries rsync reported as already current.
	uptodate int
	// sentBytes accumulates transferred bytes using source metadata.
	sentBytes int64
	// pid is the running rsync process ID for logging.
	pid int

	// started and ended bound the rsync process lifetime for the chunk.
	started, ended time.Time

	// err stores the raw process wait error for exit-code interpretation.
	err error

	// stderrLastLogLineStartIdx is the ring-buffer cursor for stderrLastLogLines.
	stderrLastLogLineStartIdx int
	// stderrLastLogLines keeps the most recent rsync stderr lines.
	stderrLastLogLines [10]string
}

// Duration reports how long the rsync process ran.
func (r result) Duration() time.Duration {
	if r.ended.After(r.started) {
		return r.ended.Sub(r.started)
	} else {
		return 0
	}
}

// Total returns the number of scheduled entries.
func (r result) Total() int64 { return int64(r.total) }

// Files returns the number of regular-file entries counted in stdout parsing.
func (r result) Files() int64 { return int64(r.files) }

// Links returns the number of symbolic-link entries counted in stdout parsing.
func (r result) Links() int64 { return int64(r.links) }

// Directories returns the number of directory entries counted in stdout parsing.
func (r result) Directories() int64 { return int64(r.directories) }

// SentBytes returns the byte total inferred from transferred file metadata.
func (r result) SentBytes() int64 { return r.sentBytes }

// appendLogLine stores a stderr line in the fixed-size recent-log ring buffer.
func (r *result) appendLogLine(line string) {
	r.stderrLastLogLines[r.stderrLastLogLineStartIdx] = line
	r.stderrLastLogLineStartIdx = (r.stderrLastLogLineStartIdx + 1) % len(r.stderrLastLogLines)
}

// lastLogLine returns the recent stderr ring buffer in chronological order.
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

// appendFilename stores filename in the fixed-size recent-file ring buffer.
func (r *result) appendFilename(filename string) {
	r.lastFilenames[r.startIdx] = filename
	r.startIdx = (r.startIdx + 1) % len(r.lastFilenames)
}

// listFilename returns the recent-file ring buffer in chronological order.
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

// addTypeCount increments the file-type counters for one parsed entry.
func (r *result) addTypeCount(mode fs.FileMode) {
	if mode.IsDir() {
		r.directories++
	} else if mode.Type()&fs.ModeSymlink != 0 {
		r.links++
	} else if mode.IsRegular() {
		r.files++
	}
}

// markEnd stamps the end time once rsync finishes.
func (r *result) markEnd() { r.ended = time.Now() }

// String formats chunk progress, exit-code context, and recent filenames for logs.
func (r result) String() string {
	buf := &strings.Builder{}

	_, _ = fmt.Fprintf(buf, "[chk:%d]rsync(%d)", r.chunkIdx, r.pid)

	err, _ := r.rsyncExitResult()
	if err != nil {
		buf.WriteString(err.Error())
	}

	if !r.started.IsZero() {
		elapsed := time.Since(r.started)
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
		_, _ = fmt.Fprintf(buf, " total(%d) = sent(%d) + uptodate(%d) + untouched(%d), processed(%d)",
			r.total, r.sent, r.uptodate, r.total-r.sent-r.uptodate, r.processed,
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

// HandleError converts rsync exit semantics into retryable or unrecoverable Go errors.
func (r *result) HandleError() error {
	err, retryable := r.rsyncExitResult()
	if err != nil && !retryable {
		err = retry.Unrecoverable(err)
	}
	return err
}

// rsyncExitResult maps the process exit code to retryability and attaches recent logs/files.
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
		return fmt.Errorf("[chk:%d]%s: %w%s", r.chunkIdx, res.Message, err, buf.String()), res.Retryable
	} else {
		return fmt.Errorf("[chk:%d]%s%s", r.chunkIdx, res.Message, buf.String()), res.Retryable
	}
}
