package util

import "bytes"

func SimpleStrconv(in []byte) (number int64) {
	if len(in) == 0 {
		return -1
	}
	var (
		out  []uint8
		sign = in[0] == '-'
	)
	if sign {
		out = make([]uint8, 0, len(in)-1)
		in = in[1:]
	} else {
		out = make([]uint8, 0, len(in))
	}
	for _, digit := range in {
		if digit < '0' || digit > '9' {
			continue
		}
		out = append(out, digit-'0')
	}
	scanned := len(out)
	if scanned == 0 {
		return -1
	}
	var pow int64 = 1
	for i := scanned; i > 0; i-- {
		number += pow * int64(out[i-1])
		pow *= 10
	}
	if sign {
		number = -number
	}
	return
}

func ScanLineFeed(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}
