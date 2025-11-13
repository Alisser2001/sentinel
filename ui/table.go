package ui

import (
	"fmt"
	"sentinel/model"
)

func Render(records []model.ProcRec, tasks, running int, l1, l5, l15, up float64) {
	now := "Sentinel"

	fmt.Printf("%s\n", now)
	fmt.Printf("Tasks: %d, running: %d\n", tasks, running)
	fmt.Printf("Load average: %.2f %.2f %.2f | Uptime: %.0fs\n", l1, l5, l15, up)

	fmt.Printf("%5s %-15s %3s %3s %1s %6s %6s %8s %8s %9s %s\n",
		"PID", "USER", "PR", "NI", "S", "%CPU", "%MEM", "VIRT(KB)", "RES(KB)", "TIME+", "COMMAND")

	shown := 0

	for _, r := range records {
		if !r.Alive {
			continue
		}
		if shown >= model.MaxRows {
			break
		}

		timeStr := FormatTimeTicks(r.CurProcTime, model.DefaultHZ)

		cmd := r.Cmd
		if len(cmd) > 30 {
			cmd = cmd[:30]
		}

		fmt.Printf("%5d %-15s %3d %3d %1c %6.2f %6.2f %8d %8d %9s %s\n",
			r.Pid,
			r.User,
			r.Prio,
			r.Nice,
			r.State,
			r.CPU,
			r.PMem,
			r.VSizeKB,
			r.RSSKB,
			timeStr,
			cmd,
		)
		shown++
	}
}