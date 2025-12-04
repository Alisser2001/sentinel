//go:build linux || darwin || freebsd || openbsd || netbsd
// +build linux darwin freebsd openbsd netbsd

package proc

// #include <unistd.h>
import "C"

func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// DetectHZ attempts to detect system clock ticks per second (CLK_TCK).
// Uses sysconf(_SC_CLK_TCK) via cgo for maximum portability.
// Falls back to 100 if detection fails.
func DetectHZ() int {
	hz := int(C.sysconf(C._SC_CLK_TCK))
	if hz <= 0 {
		return 100 // Safe fallback
	}
	return hz
}
