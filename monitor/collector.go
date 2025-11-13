package monitor

import (
	"os"
	"strconv"

	"sentinel/model"
	"sentinel/proc"
)

type Collector struct {
	Records []model.ProcRec
}

func NewCollector() *Collector {
	return &Collector{Records: make([]model.ProcRec, 0)}
}

func (c *Collector) findIdx(pid int) int {
	for i := range c.Records {
		if c.Records[i].Pid == pid {
			return i
		}
	}
	return -1
}

func (c *Collector) ensureRecord(pid int) int {
	idx := c.findIdx(pid)
	if idx >= 0 {
		c.Records[idx].Alive = true
		return idx
	}
	rec := model.ProcRec{
		Pid:   pid,
		Alive: true,
	}
	c.Records = append(c.Records, rec)
	return len(c.Records) - 1
}

func (c *Collector) Scan() (tasks, running int) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0, 0
	}

	for i := range c.Records {
		c.Records[i].Alive = false
	}

	tasks = 0
	running = 0

	for _, ent := range entries {
		if !proc.IsNumeric(ent.Name()) {
			continue
		}
		pid, _ := strconv.Atoi(ent.Name())
		idx := c.ensureRecord(pid)
		rec := &c.Records[idx]

		comm, state, ut, st, pr, ni, _, _, vsizeKB, rssKB, ok := proc.ReadProcStat(pid)
		if !ok {
			rec.Alive = false
			continue
		}

		rec.State = state
		rec.Prio = pr
		rec.Nice = ni
		rec.VSizeKB = vsizeKB
		rec.RSSKB = rssKB

		uid := proc.ReadStatusUID(pid)
		rec.Uid = uid
		rec.User = proc.UIDToName(uid)

		cmd := proc.ReadCmdline(pid)
		if cmd == "" {
			cmd = comm
		}
		rec.Cmd = cmd

		rec.CurProcTime = ut + st

		tasks++
		if state == 'R' {
			running++
		}
	}
	return
}

func (c *Collector) Compact() {
	dst := c.Records[:0]
	for _, r := range c.Records {
		if r.Alive {
			dst = append(dst, r)
		}
	}
	c.Records = dst
}