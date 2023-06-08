package util

//import (
//	"fmt"
//	"io/ioutil"
//	"os"
//	"syscall"
//)
//
//// Write a pid file, but first make sure it doesn't exist with a running pid.
//func writePidFile(pidFile string) error {
//	// Read in the pid file as a slice of bytes.
//
//	if piddata, err := io.ReadFile(pidFile); err == nil {
//		// Convert the file contents to an integer.
//		if pid := SimpleStrconv(piddata); pid > 0 {
//			// Look for the pid in the process list.
//			if process, err := os.FindProcess(int(pid)); err == nil {
//				// Send the process a signal zero kill.
//				if err := process.Signal(syscall.Signal(0)); err == nil {
//					// We only get an error if the pid isn't running, or it's not ours.
//					return fmt.Errorf("pid already running: %d", pid)
//				}
//			}
//		}
//	}
//
//	// If we get here, then the pidfile didn't exist,
//	// or the pid in it doesn't belong to the user running this app.
//	return ioutil.WriteFile(pidFile, []byte(fmt.Sprintf("%d", os.Getpid())), 0o664)
//}
