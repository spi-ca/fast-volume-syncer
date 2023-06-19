//go:build !unix || windows
// +build !unix windows

package sys

func DupFD(oldfd int, newfd int) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
