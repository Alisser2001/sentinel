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

// FormatKB formats kilobytes with appropriate unit (KB, MB, GB)
func FormatKB(kb int64) string {
	if kb < 1024 {
		return fmt.Sprintf("%dK", kb)
	}
	mb := float64(kb) / 1024.0
	if mb < 1024 {
		return fmt.Sprintf("%.1fM", mb)
	}
	gb := mb / 1024.0
	return fmt.Sprintf("%.2fG", gb)
}

// FormatUptime formats uptime in seconds to human readable string
func FormatUptime(seconds float64) string {
	days := int(seconds) / 86400
	hours := (int(seconds) % 86400) / 3600
	mins := (int(seconds) % 3600) / 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// FormatTime formats CPU time in jiffies to human readable format
func FormatTime(jiffies uint64, hz int) string {
	if hz <= 0 {
		hz = 100
	}
	seconds := jiffies / uint64(hz)
	mins := seconds / 60
	secs := seconds % 60

	if mins >= 60 {
		hours := mins / 60
		mins = mins % 60
		return fmt.Sprintf("%d:%02d:%02d", hours, mins, secs)
	}
	return fmt.Sprintf("%d:%02d", mins, secs)
}
