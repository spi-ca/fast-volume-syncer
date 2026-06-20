package returns

import (
	"errors"
	"strings"
	"testing"
)

func TestExecutionResultLastLogLineKeepsLastTenInOrder(t *testing.T) {
	res := &ExecutionResult{}
	for _, line := range []string{"one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"} {
		res.AppendLogLine(line)
	}

	got := res.LastLogLine()
	want := []string{"three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected last log lines\nwant: %#v\n got: %#v", want, got)
	}
}

func TestExecutionResultHandleErrorReturnsNilForSuccessEvenWithLogs(t *testing.T) {
	res := &ExecutionResult{}
	res.AppendLogLine("warning before success")
	if err := res.HandleError(); err != nil {
		t.Fatalf("expected nil error for successful result, got %v", err)
	}
}

func TestExecutionResultHandleErrorIncludesLastLogsForNonExitError(t *testing.T) {
	res := &ExecutionResult{Err: errors.New("copy failed")}
	res.AppendLogLine("first diagnostic")
	res.AppendLogLine("second diagnostic")

	err := res.HandleError()
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"copy failed", "last 2 logs", "first diagnostic", "second diagnostic"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error message missing %q: %s", want, msg)
		}
	}
}
