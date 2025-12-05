package ui

import (
	"fmt"
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

// SendData is called by engine to push new data
func SendData(p *tea.Program, records []model.ProcRec, tasks, running int, l1, l5, l15, uptime float64) {
	p.Send(dataMsg{
		records: records,
		tasks:   tasks,
		running: running,
		l1:      l1,
		l5:      l5,
		l15:     l15,
		uptime:  uptime,
	})
}
