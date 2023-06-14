package util

import (
	"io"
	"log"
	"os"
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
func TestSymlink(t *testing.T) {
	linkpath := "b /a"
	dest := "b"
	if err := os.Symlink(dest, linkpath); err != nil {
		log.Printf("failed to make a symbolic link(%s -> %s) :%v %t", linkpath, dest, err, os.IsNotExist(err))
	}

}
