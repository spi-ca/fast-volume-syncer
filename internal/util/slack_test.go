package util

import (
	"io"
	"testing"
)

func TestSendSlackMessage(t *testing.T) {
	SlackSender.Start()
	defer SlackSender.Close()
	prevWriter := ErrLog.Writer()
	defer func() {
		ErrLog.SetOutput(prevWriter)
		SlackSender.Close()
	}()
	ErrLog.SetOutput(io.MultiWriter(prevWriter, SlackSender))
	for i := 0; i < 5; i++ {
		ErrLog.Printf("TestRunner_sendErrorMessage test (%d)", i)
	}
}
