package util

import (
	"testing"
)

func TestSendSlackMessage(t *testing.T) {
	SendSlackMessage("TestRunner_sendErrorMessage test")
}
