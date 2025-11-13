package proc

import (
	"os"
	"bufio"
	"strings"
	"strconv"
)

func ReadMemTotalKB() int64 {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 1
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			for _, tok := range fields {
				if v, err := strconv.ParseInt(tok, 10, 64); err == nil && v > 0 {
					return v
				}
			}
		}
	}
	return 1
}