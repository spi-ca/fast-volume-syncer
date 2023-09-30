//go:build !unix || windows
// +build !unix windows

package sys

func ReplaceFD(oldfd int, newfd int) (err error) {
	return fmt.Errorf("this os(%s) not supported", runtime.GOOS)
}
