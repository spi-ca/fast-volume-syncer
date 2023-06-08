package util

import (
	"fmt"
	"testing"
)

func TestSendSlackMessage(t *testing.T) {
	SlackSender.Start()
	defer SlackSender.Close()
	for i := 0; i < 100; i++ {
		SendSlackMessage(fmt.Sprintf("TestRunner_sendErrorMessage test (%d)", i))
	}
}
