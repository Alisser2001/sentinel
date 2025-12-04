package monitor

import (
	"os"
	"sentinel/model"
	"sentinel/proc"
	"sort"
)

type Collector struct {
	Records []model.ProcRec
	PidMap  map[int]int
}

func NewCollector() *Collector {
	return &Collector{
		Records: make([]model.ProcRec, 0, model.MaxRows),
		PidMap:  make(map[int]int),
	}
}

func (c *Collector) Scan() (int, int) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, 0
	}

	totalTasks := 0
	runningTasks := 0

	seen := make(map[int]bool)

	for _, e := range entries {
		name := e.Name()
		if !proc.IsNumeric(name) {
			continue
		}

		pid := 0
		for _, ch := range name {
			pid = pid*10 + int(ch-'0')
		}

		totalTasks++

		comm, state, utime, stime, prio, nice, _, _, vsizeKB, rssKB, ok := proc.ReadProcStat(pid)
		if !ok {
			continue
		}

		if state == 'R' {
			runningTasks++
		}

		seen[pid] = true

		uid := proc.ReadStatusUID(pid)
		user := proc.UIDToName(uid)
		cmd := proc.ReadCmdline(pid)

		curProcTime := utime + stime

		idx, exists := c.PidMap[pid]
		if exists {
			rec := &c.Records[idx]
			rec.Alive = true
			rec.User = user
			rec.Comm = comm // ← ACTUALIZAR
			rec.State = state
			rec.Prio = prio
			rec.Nice = nice
			rec.CurProcTime = curProcTime
			rec.VSizeKB = vsizeKB
			rec.RSSKB = rssKB
			rec.Cmd = cmd
		} else {
			newRec := model.ProcRec{
				Pid:          pid,
				Uid:          uid,
				User:         user,
				Comm:         comm, // ← GUARDAR COMM
				State:        state,
				Prio:         prio,
				Nice:         nice,
				PrevProcTime: 0,
				CurProcTime:  curProcTime,
				CPU:          0,
				VSizeKB:      vsizeKB,
				RSSKB:        rssKB,
				PMem:         0,
				Cmd:          cmd,
				Alive:        true,
			}
			c.Records = append(c.Records, newRec)
			c.PidMap[pid] = len(c.Records) - 1
		}
	}

	for i := range c.Records {
		rec := &c.Records[i]
		if !seen[rec.Pid] {
			rec.Alive = false
		}
	}

	return totalTasks, runningTasks
}

func (c *Collector) Compact() {
	alive := c.Records[:0]
	for i := range c.Records {
		if c.Records[i].Alive {
			alive = append(alive, c.Records[i])
		}
	}
	c.Records = alive

	c.PidMap = make(map[int]int)
	for i := range c.Records {
		c.PidMap[c.Records[i].Pid] = i
	}
}

func SortByCPU(records []model.ProcRec) {
	sort.Slice(records, func(i, j int) bool {
		return records[i].CPU > records[j].CPU
	})
}
