// Package util provides logging, formatting, flag-binding, and binary lookup helpers.
package util

import (
	"fmt"
	"math"
)

var (
	// sizesIEC contains IEC suffixes for powers of 1024.
	sizesIEC = []string{
		"B",
		"KiB",
		"MiB",
		"GiB",
		"TiB",
		"PiB",
		"EiB",
		"ZiB",
		"YiB",
	}
	// sizes contains decimal suffixes for powers of 1000.
	sizes = []string{
		"B",
		"KB",
		"MB",
		"GB",
		"TB",
		"PB",
		"EB",
		"ZB",
		"YB",
	}
)

// logn computes a logarithm in the caller-supplied base.
func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

// humanateBytes formats a byte count with the provided base and suffix table.
func humanateBytes(s int64, base float64, sizes []string) string {
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := float64(s) / math.Pow(base, math.Floor(e))
	f := "%.0f"
	if val < 10 {
		f = "%.1f"
	}
	return fmt.Sprintf(f+"%s", val, suffix)
}

// FileSizeIEC formats bytes with IEC suffixes such as KiB and MiB.
func FileSizeIEC(s int64) string {
	return humanateBytes(s, 1024, sizesIEC)
}

// FileSize formats bytes with decimal suffixes such as KB and MB.
func FileSize(s int64) string {
	return humanateBytes(s, 1000, sizes)
}
