package proc

import (
	"fmt"
	"os"
)

func ReadLoadavg() [3]float64 {
	f, err := os.Open("/proc/loadavg")
	if err != nil {
		return [3]float64{}
	}
	defer f.Close()

	var l1, l5, l15 float64
	fmt.Fscan(f, &l1, &l5, &l15)

	return [3]float64{l1, l5, l15}
}
