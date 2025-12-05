package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"sentinel/config"
	"sentinel/model"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model holds TUI state
type Model struct {
	table       table.Model
	records     []model.ProcRec
	tasks       int
	running     int
	l1, l5, l15 float64
	uptime      float64
	sorter      *model.Sorter
	interval    time.Duration
	width       int
	height      int

	// Filtering
	filterInput textinput.Model
	filterText  string
	mode        uiMode

	// Status messages
	statusText  string
	statusError bool

	// Kill/Nice confirmation
	selectedPID int
	niceValue   int

	cfg                  *config.SentinelConfig
	webhookNames         []string
	selectedWebhookIndex int

	cpuInput          textinput.Model
	memInput          textinput.Model
	webhookNameInput  textinput.Model
	webhookURLInput   textinput.Model
	addingWebhookStep int
}

func NewModel(interval time.Duration) Model {
	columns := []table.Column{
		{Title: "PID", Width: 7},
		{Title: "USER", Width: 10},
		{Title: "PROGRAM", Width: 15},
		{Title: "%CPU", Width: 7},
		{Title: "%MEM", Width: 7},
		{Title: "VSIZE", Width: 9},
		{Title: "RSS", Width: 9},
		{Title: "S", Width: 3},
		{Title: "TIME+", Width: 9},
		{Title: "COMMAND", Width: 45},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color("cyan"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// Setup filter input
	ti := textinput.New()
	ti.Placeholder = "filter by command or user..."
	ti.CharLimit = 50

	cfg, _ := config.LoadConfig()

	cpuInput := textinput.New()
	cpuInput.Placeholder = "CPU threshold %"
	cpuInput.CharLimit = 4
	cpuInput.SetValue(fmt.Sprintf("%.0f", cfg.CPUThreshold))

	memInput := textinput.New()
	memInput.Placeholder = "MEM threshold %"
	memInput.CharLimit = 4
	memInput.SetValue(fmt.Sprintf("%.0f", cfg.MemThreshold))

	webhookName := textinput.New()
	webhookName.Placeholder = "webhook name"

	webhookURL := textinput.New()
	webhookURL.Placeholder = "webhook URL"

	whNames := make([]string, 0, len(cfg.Webhooks))
	for name := range cfg.Webhooks {
		whNames = append(whNames, name)
	}

	return Model{
		table:                t,
		sorter:               model.NewSorter(),
		interval:             interval,
		filterInput:          ti,
		mode:                 normalMode,
		cfg:                  cfg,
		webhookNames:         whNames,
		cpuInput:             cpuInput,
		memInput:             memInput,
		webhookNameInput:     webhookName,
		webhookURLInput:      webhookURL,
		selectedWebhookIndex: 0,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(m.interval),
		tea.EnterAltScreen,
	)
}

func tickCmd(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// --- CSV export for experiment comparison (pidstat-like) ---
var (
	exportCSVOnce sync.Once
	exportCSVFile *os.File
)

func openExportCSV() {
	exportCSVOnce.Do(func() {
		path := os.Getenv("SENTINEL_EXPORT_CSV")

		// DEBUG: Imprimir en stderr
		fmt.Fprintf(os.Stderr, "[DEBUG] SENTINEL_EXPORT_CSV = '%s'\n", path)

		if path == "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] CSV export disabled (env var not set)\n")
			return
		}

		fmt.Fprintf(os.Stderr, "[DEBUG] Opening CSV file: %s\n", path)

		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to open CSV export: %v\n", err)
			return
		}

		exportCSVFile = f
		fmt.Fprintf(os.Stderr, "[DEBUG] CSV file opened successfully\n")

		// Write header if file is empty
		stat, err := f.Stat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to stat file: %v\n", err)
			return
		}

		if stat.Size() == 0 {
			header := "timestamp_ms,pid,user,comm,cpu_pct,mem_pct,vsize_kb,rss_kb,state,threads,time_plus,cmdline\n"
			_, err = f.WriteString(header)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to write header: %v\n", err)
				return
			}
			fmt.Fprintf(os.Stderr, "[DEBUG] CSV header written\n")
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] CSV file already has data (%d bytes)\n", stat.Size())
		}
	})
}

func exportRecordCSV(t time.Time, r model.ProcRec) {
	if exportCSVFile == nil || !r.Alive {
		return
	}

	timeStr := FormatTimeTicks(r.CurProcTime, model.DefaultHZ)
	cmdline := r.Cmd
	if cmdline == "" {
		cmdline = r.Comm
	}

	// Replace commas to avoid CSV issues
	cmdline = ""
	for _, ch := range r.Cmd {
		if ch == ',' {
			cmdline += " "
		} else {
			cmdline += string(ch)
		}
	}

	fmt.Fprintf(exportCSVFile,
		"%d,%d,%s,%s,%.1f,%.1f,%d,%d,%s,%d,%s,%s\n",
		t.UnixMilli(),
		r.Pid,
		r.User,
		r.Comm,
		r.CPU,
		r.PMem,
		r.VSizeKB,
		r.RSSKB,
		string(r.State),
		timeStr,
		cmdline,
	)
}

// SendData is called by engine to push new data
func SendData(p *tea.Program, records []model.ProcRec, tasks, running int, loads [3]float64, uptime float64) {
	// Export CSV if enabled
	openExportCSV()
	if exportCSVFile != nil {
		now := time.Now()
		for _, r := range records {
			exportRecordCSV(now, r)
		}
	}

	p.Send(dataMsg{
		records: records,
		tasks:   tasks,
		running: running,
		l1:      loads[0],
		l5:      loads[1],
		l15:     loads[2],
		uptime:  uptime,
	})
}
