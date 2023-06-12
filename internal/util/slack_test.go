package util

import (
	"os"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestSendSlackMessage(t *testing.T) {
	SlackSender.Start()
	defer SlackSender.Close()
	log.AddHook(SlackSender)
	for i := 0; i < 100; i++ {
		log.Errorf("TestRunner_sendErrorMessage test (%d)", i)
	}
}
func TestSymlink(t *testing.T) {
	linkpath := "b /a"
	dest := "b"
	if err := os.Symlink(dest, linkpath); err != nil {
		log.Printf("failed to make a symbolic link(%s -> %s) :%v %t", linkpath, dest, err, os.IsNotExist(err))
	}

}
