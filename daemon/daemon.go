package daemon

import (
	"context"
	"fmt"
	"log"
	"time"

	"sentinel/alert"
	"sentinel/config"
	"sentinel/model"
	"sentinel/monitor"
	"sentinel/proc"

	"github.com/fsnotify/fsnotify"
)

type Daemon struct {
	engine     *monitor.Engine
	cfg        *config.SentinelConfig
	logger     *log.Logger
	interval   time.Duration
	hz         int
	lastAlerts map[int]time.Time
}

func New(interval time.Duration, hz int, logger *log.Logger) *Daemon {
	cfg, _ := config.LoadConfig()

	return &Daemon{
		engine:     monitor.NewEngine(),
		cfg:        cfg,
		logger:     logger,
		interval:   interval,
		hz:         hz,
		lastAlerts: make(map[int]time.Time),
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	go d.watchConfig()

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	prevTotal := int64(proc.ReadTotalCPUTime())
	memTotal := proc.ReadMemTotalKB()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			tasks, running := d.engine.Collector.Scan()

			curTotal := int64(proc.ReadTotalCPUTime())
			sysDelta := int64(1)
			if curTotal > prevTotal {
				sysDelta = curTotal - prevTotal
			}

			for i := range d.engine.Collector.Records {
				r := &d.engine.Collector.Records[i]
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

				d.checkAlerts(r)
			}

			prevTotal = curTotal
			memTotal = proc.ReadMemTotalKB()

			_ = tasks
			_ = running
		}
	}
}

func (d *Daemon) checkAlerts(r *model.ProcRec) {
	now := time.Now()

	if t, ok := d.lastAlerts[r.Pid]; ok {
		if now.Sub(t) < 60*time.Second {
			return
		}
	}

	if r.CPU >= d.cfg.CPUThreshold {
		alert.SendDiscord(
			d.cfg.Webhooks[d.cfg.ActiveWebhook],
			fmt.Sprintf("⚠ High CPU: PID %d (%s)", r.Pid, r.Cmd),
		)
		d.lastAlerts[r.Pid] = now
	}

	if r.PMem >= d.cfg.MemThreshold {
		alert.SendDiscord(
			d.cfg.Webhooks[d.cfg.ActiveWebhook],
			fmt.Sprintf("⚠ High Memory: PID %d (%s)", r.Pid, r.Cmd),
		)
		d.lastAlerts[r.Pid] = now
	}
}

func (d *Daemon) watchConfig() {
	w, _ := fsnotify.NewWatcher()
	w.Add(config.ConfigPath())

	for {
		select {
		case e := <-w.Events:
			if e.Op&fsnotify.Write == fsnotify.Write {
				cfg, err := config.LoadConfig()
				if err == nil {
					d.cfg = cfg
					d.logger.Println("config reloaded")
				}
			}
		}
	}
}
