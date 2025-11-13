package monitor

import (
	"time"
	"fmt"
	"sentinel/proc"
	"sentinel/ui"
)

type Engine struct {
	Collector *Collector
}

func NewEngine() *Engine {
	return &Engine{Collector: NewCollector()}
}

func (e *Engine) Run() {
	prevTotal := proc.ReadTotalCPUTime()
	memTotal := proc.ReadMemTotalKB()

	for {
		tasks, running := e.Collector.Scan()

		curTotal := proc.ReadTotalCPUTime()
		sysDelta := uint64(1)
		if curTotal > prevTotal {
			sysDelta = curTotal - prevTotal
		}

		// calcular %CPU y %MEM
		for i := range e.Collector.Records {
			r := &e.Collector.Records[i]
			if !r.Alive {
				continue
			}
			if r.PrevProcTime == 0 {
				r.CPU = 0
			} else {
				procDelta := uint64(0)
				if r.CurProcTime > r.PrevProcTime {
					procDelta = r.CurProcTime - r.PrevProcTime
				}
				r.CPU = float64(procDelta) * 100.0 / float64(sysDelta)
			}
			if memTotal > 0 {
				r.PMem = float64(r.RSSKB) * 100.0 / float64(memTotal)
			}
			r.PrevProcTime = r.CurProcTime
		}

		prevTotal = curTotal
		memTotal = proc.ReadMemTotalKB()

		// compactar
		e.Collector.Compact()

		// ordenar
		SortByCPU(e.Collector.Records)

		// datos globales
		l1, l5, l15 := proc.ReadLoadavg()
		up := proc.ReadUptime()

		// renderizar
		fmt.Print("\033[H\033[J")
		ui.Render(e.Collector.Records, tasks, running, l1, l5, l15, up)

		time.Sleep(400 * time.Millisecond)
	}
}