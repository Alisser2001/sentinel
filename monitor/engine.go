package monitor

import (
	"context"
	"log"
	"sentinel/model"
	"sentinel/proc"
	"sentinel/ui"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Engine struct {
	Collector *Collector
	program   *tea.Program
}

func NewEngine() *Engine {
	return &Engine{Collector: NewCollector()}
}

func (e *Engine) Run(ctx context.Context, interval time.Duration, hz int, logger *log.Logger) error {
	model.DefaultHZ = hz

	// Start bubbletea program
	tuiModel := ui.NewModel(interval)
	e.program = tea.NewProgram(tuiModel, tea.WithAltScreen())

	// Start background data collector
	go e.collectLoop(ctx, interval, logger)

	// Run TUI (blocks until quit)
	if _, err := e.program.Run(); err != nil {
		return err
	}

	return ctx.Err()
}

func (e *Engine) collectLoop(ctx context.Context, interval time.Duration, logger *log.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	prevTotal := int64(proc.ReadTotalCPUTime())
	memTotal := proc.ReadMemTotalKB()

	for {
		select {
		case <-ctx.Done():
			e.program.Quit()
			return

		case <-ticker.C:
			prevTotal, memTotal = e.handleTick(prevTotal, memTotal)
		}
	}
}

// handleTick performs one collection cycle: scans processes, updates metrics,
// compacts records, reads system load/uptime and sends data to the TUI.
// Returns updated prevTotal and memTotal.
func (e *Engine) handleTick(prevTotal int64, memTotal int64) (int64, int64) {
	tasks, running := e.Collector.Scan()

	curTotal := int64(proc.ReadTotalCPUTime())
	sysDelta := int64(1)
	if curTotal > prevTotal {
		sysDelta = curTotal - prevTotal
	}

	e.computeMetrics(sysDelta, memTotal)

	prevTotal = curTotal
	memTotal = proc.ReadMemTotalKB()

	e.Collector.Compact()

	loads := proc.ReadLoadavg()
	uptime := proc.ReadUptime()

	ui.SendData(e.program, e.Collector.Records, tasks, running, loads, uptime)
	return prevTotal, memTotal
}

// computeMetrics updates %CPU and %MEM for alive records using deltas.
func (e *Engine) computeMetrics(sysDelta int64, memTotal int64) {
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
}
