package proc

import (
	"bufio"
	"os"
	"strings"
	"strconv"
)

func ReadTotalCPUTime() uint64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0
	}
	fields := strings.Fields(line)

	if len(fields) < 2 {
		return 0
	}

	var total uint64
	for _, tok := range fields[1:] {
		v, err := strconv.ParseUint(tok, 10, 64)
		if err == nil {
			total += v
		}
	}
	return total
}