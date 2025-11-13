package ui

import "fmt"

func FormatTimeTicks(ticks uint64, hz int) string {
	totalCS := (ticks * 100) / uint64(hz)

	h := totalCS / 360000
	m := (totalCS % 360000) / 6000
	s := (totalCS % 6000) / 100
	cs := totalCS % 100

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d.%02d", m, s, cs)
}
