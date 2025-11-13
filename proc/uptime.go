package proc

import (
	"os"
	"fmt"
)

func ReadUptime() float64 {
	f, err := os.Open("/proc/uptime")
	if err != nil {
		return 0
	}
	defer f.Close()

	var up float64
	fmt.Fscan(f, &up)
	return up
}