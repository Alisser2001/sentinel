package monitor

import (
	"sort"
	"sentinel/model"
)

func SortByCPU(records []model.ProcRec) {
	sort.Slice(records, func(i, j int) bool {
		a := records[i]
		b := records[j]

		if a.Alive != b.Alive {
			return a.Alive && !b.Alive
		}
		if a.CPU != b.CPU {
			return a.CPU > b.CPU
		}
		if a.RSSKB != b.RSSKB {
			return a.RSSKB > b.RSSKB
		}
		return a.Pid < b.Pid
	})
}