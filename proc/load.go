package proc

import (
	"os"
	"fmt"
)

func ReadLoadavg() (float64, float64, float64) {
	f, err := os.Open("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	defer f.Close()

	var l1, l5, l15 float64
	fmt.Fscan(f, &l1, &l5, &l15)

	return l1, l5, l15
}