package rsync

import (
	"fmt"
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
		20:  {false, false, "Received SIGUSR1 or SIGINT"},
		21:  {false, false, "Some error returned by waitpid()"},
		22:  {false, true, "Error allocating core memory buffers"},
		23:  {false, false, "Partial transfer due to error"},
		24:  {true, false, "Partial transfer due to vanished source files"},
		25:  {true, true, "The --max-delete limit stopped deletions"},
		30:  {false, true, "Timeout in data send/receive"},
		35:  {false, true, "Timeout waiting for daemon connection"},
		127: {false, false, "It can mean you don't even have rsync binary installed on your system."},
		255: {false, true, "Rsync just passed exit code from a command it used to connect - typically SSH."},
	}
)

func isExitedNormally(exitcode int, err error) (error, bool) {
	result := exitCodeMap[-1]
	if _result, ok := exitCodeMap[exitcode]; ok {
		result = _result
	}

	if result.Success && err == nil {
		return nil, result.Retryable
	} else if err != nil {
		return fmt.Errorf("exitcode %d,%s: %w", exitcode, result.Message, err), result.Retryable
	} else {
		return fmt.Errorf("exitcode %d,%s", exitcode, result.Message), result.Retryable
	}
}
