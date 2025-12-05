package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"sentinel/config"
	"sentinel/model"
	"sentinel/proc"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("cyan")).
			Align(lipgloss.Left)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	highCPUStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("red"))
	medCPUStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("yellow"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("red")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Bold(true)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("yellow")).
			Bold(true).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("yellow")).
			Padding(1, 2)

	// Nuevos estilos para ayuda visual
	sortedColumnStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("green")).
				Bold(true).
				Underline(true)

	helpBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1).
			MarginTop(1)

	keybindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	keybindDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			Padding(0, 1)
)

// Messages
type tickMsg time.Time
type dataMsg struct {
	records     []model.ProcRec
	tasks       int
	running     int
	l1, l5, l15 float64
	uptime      float64
}

type statusMsg struct {
	text    string
	isError bool
}

// UI Modes
type uiMode int

const (
	normalMode uiMode = iota
	filterMode
	confirmKillMode
	confirmNiceMode
	helpMode // Nuevo modo para mostrar ayuda
	settingsMode
	editThresholdCPU
	editThresholdMEM
	addWebhookMode
	confirmDeleteWebhook
	selectWebhookMode
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

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case settingsMode:
			return m.handleSettingsMode(msg)
		case editThresholdCPU:
			return m.handleEditCPU(msg)
		case editThresholdMEM:
			return m.handleEditMEM(msg)
		case addWebhookMode:
			return m.handleAddWebhook(msg)
		}
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 12) // Más espacio para ayuda
		return m, nil

	case tickMsg:
		return m, tickCmd(m.interval)

	case dataMsg:
		m.records = msg.records
		m.tasks = msg.tasks
		m.running = msg.running
		m.l1 = msg.l1
		m.l5 = msg.l5
		m.l15 = msg.l15
		m.uptime = msg.uptime
		m.updateTable()
		return m, nil

	case statusMsg:
		m.statusText = msg.text
		m.statusError = msg.isError
		return m, nil
	}

	// Update filter input if in filter mode
	if m.mode == filterMode {
		m.filterInput, cmd = m.filterInput.Update(msg)
		m.filterText = m.filterInput.Value()
		m.updateTable()
		return m, cmd
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case normalMode:
		return m.handleNormalMode(msg)
	case filterMode:
		return m.handleFilterMode(msg)
	case confirmKillMode:
		return m.handleConfirmKill(msg)
	case confirmNiceMode:
		return m.handleConfirmNice(msg)
	case helpMode:
		return m.handleHelpMode(msg)
	}
	return m, nil
}

func (m Model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	// Mostrar ayuda
	case "?", "h":
		m.mode = helpMode
		return m, nil

	// Sorting
	case "c":
		m.sorter.Toggle(model.SortByCPUCol)
		m.updateTable()
	case "m":
		m.sorter.Toggle(model.SortByMEM)
		m.updateTable()
	case "p":
		m.sorter.Toggle(model.SortByPID)
		m.updateTable()
	case "u":
		m.sorter.Toggle(model.SortByUSER)
		m.updateTable()
	case "v":
		m.sorter.Toggle(model.SortByVSIZE)
		m.updateTable()
	case "r":
		m.sorter.Toggle(model.SortByRSS)
		m.updateTable()
	case "t":
		m.sorter.Toggle(model.SortByTIME)
		m.updateTable()

	// Filtering
	case "/":
		m.mode = filterMode
		m.filterInput.Focus()
		return m, textinput.Blink

	// Kill process
	case "k":
		if pid := m.getSelectedPID(); pid > 0 {
			m.selectedPID = pid
			m.mode = confirmKillMode
		}

	// Force kill
	case "K":
		if pid := m.getSelectedPID(); pid > 0 {
			if err := proc.ForceKillProcess(pid); err != nil {
				return m, m.showStatus(fmt.Sprintf("Error: %v", err), true)
			}
			return m, m.showStatus(fmt.Sprintf("Sent SIGKILL to PID %d", pid), false)
		}

	// Renice (increase priority)
	case "n":
		if pid := m.getSelectedPID(); pid > 0 {
			m.selectedPID = pid
			m.niceValue = -5
			m.mode = confirmNiceMode
		}

	// Renice (decrease priority)
	case "N":
		if pid := m.getSelectedPID(); pid > 0 {
			m.selectedPID = pid
			m.niceValue = 5
			m.mode = confirmNiceMode
		}

	case "s":
		m.mode = settingsMode
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = normalMode
		m.filterInput.Blur()
		return m, nil
	case "enter":
		m.mode = normalMode
		m.filterInput.Blur()
		m.filterText = m.filterInput.Value()
		m.updateTable()
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	m.updateTable()
	return m, cmd
}

func (m Model) handleConfirmKill(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if err := proc.TerminateProcess(m.selectedPID); err != nil {
			m.mode = normalMode
			return m, m.showStatus(fmt.Sprintf("Error: %v", err), true)
		}
		m.mode = normalMode
		return m, m.showStatus(fmt.Sprintf("Sent SIGTERM to PID %d", m.selectedPID), false)

	case "n", "N", "esc", "q":
		m.mode = normalMode
		return m, nil
	}
	return m, nil
}

func (m Model) handleConfirmNice(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		currentNice, _ := proc.GetProcessPriority(m.selectedPID)
		newNice := currentNice + m.niceValue

		if newNice < -20 {
			newNice = -20
		} else if newNice > 19 {
			newNice = 19
		}

		if err := proc.SetProcessPriority(m.selectedPID, newNice); err != nil {
			m.mode = normalMode
			return m, m.showStatus(fmt.Sprintf("Error: %v", err), true)
		}
		m.mode = normalMode
		return m, m.showStatus(fmt.Sprintf("Changed nice of PID %d to %d", m.selectedPID, newNice), false)

	case "n", "N", "esc", "q":
		m.mode = normalMode
		return m, nil
	}
	return m, nil
}

func (m Model) handleHelpMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?", "h":
		m.mode = normalMode
		return m, nil
	}
	return m, nil
}

func (m Model) handleSettingsMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "q", "esc":
		m.mode = normalMode
		return m, nil

	case "e":
		m.mode = editThresholdCPU
		m.cpuInput.Focus()
		return m, nil

	case "m":
		m.mode = editThresholdMEM
		m.memInput.Focus()
		return m, nil

	case "a":
		m.mode = addWebhookMode
		m.addingWebhookStep = 0
		m.webhookNameInput.SetValue("")
		m.webhookURLInput.SetValue("")
		m.webhookNameInput.Focus()
		return m, nil

	case "d":
		if len(m.webhookNames) > 0 {
			m.mode = confirmDeleteWebhook
		}
		return m, nil

	case "w":
		if len(m.webhookNames) > 0 {
			name := m.webhookNames[m.selectedWebhookIndex]
			m.cfg.ActiveWebhook = name
			config.SaveConfig(m.cfg)
		}
		return m, nil

	case "up":
		if m.selectedWebhookIndex > 0 {
			m.selectedWebhookIndex--
		}
		return m, nil

	case "down":
		if m.selectedWebhookIndex < len(m.webhookNames)-1 {
			m.selectedWebhookIndex++
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleEditCPU(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		v := m.cpuInput.Value()
		f, _ := strconv.ParseFloat(v, 64)
		m.cfg.CPUThreshold = f
		config.SaveConfig(m.cfg)
		m.mode = settingsMode
		return m, nil

	case "esc":
		m.mode = settingsMode
		return m, nil
	}

	var cmd tea.Cmd
	m.cpuInput, cmd = m.cpuInput.Update(msg)
	return m, cmd
}

func (m Model) handleEditMEM(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		v := m.memInput.Value()
		f, _ := strconv.ParseFloat(v, 64)
		m.cfg.MemThreshold = f
		config.SaveConfig(m.cfg)
		m.mode = settingsMode
		return m, nil

	case "esc":
		m.mode = settingsMode
		return m, nil
	}

	var cmd tea.Cmd
	m.memInput, cmd = m.memInput.Update(msg)
	return m, cmd
}

func (m Model) handleAddWebhook(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "enter":
		if m.addingWebhookStep == 0 {
			if m.webhookNameInput.Value() == "" {
				return m, nil
			}

			m.addingWebhookStep = 1
			m.webhookNameInput.Blur()
			m.webhookURLInput.Focus()
			return m, nil
		}

		if m.addingWebhookStep == 1 {
			name := m.webhookNameInput.Value()
			url := m.webhookURLInput.Value()

			if name != "" && url != "" {
				m.cfg.Webhooks[name] = url
				config.SaveConfig(m.cfg)

				m.webhookNames = append(m.webhookNames, name)
			}

			m.addingWebhookStep = 0
			m.webhookNameInput.SetValue("")
			m.webhookURLInput.SetValue("")
			m.webhookNameInput.Focus()
			m.mode = settingsMode
			return m, nil
		}

	case "esc":
		m.addingWebhookStep = 0
		m.webhookNameInput.Blur()
		m.webhookURLInput.Blur()
		m.mode = settingsMode
		return m, nil
	}

	var cmd tea.Cmd

	if m.addingWebhookStep == 0 {
		m.webhookNameInput, cmd = m.webhookNameInput.Update(msg)
		return m, cmd
	}

	m.webhookURLInput, cmd = m.webhookURLInput.Update(msg)
	return m, cmd
}

func (m *Model) updateTable() {
	// Apply filter
	filtered := m.records
	if m.filterText != "" {
		filtered = make([]model.ProcRec, 0, len(m.records))
		searchLower := strings.ToLower(m.filterText)
		for _, r := range m.records {
			if !r.Alive {
				continue
			}
			// Include search in Cmd, User and Comm (program name)
			if strings.Contains(strings.ToLower(r.Cmd), searchLower) ||
				strings.Contains(strings.ToLower(r.User), searchLower) ||
				strings.Contains(strings.ToLower(r.Comm), searchLower) {
				filtered = append(filtered, r)
			}
		}
	}

	// Sort on a copy to avoid mutating shared collector slice
	sorted := make([]model.ProcRec, len(filtered))
	copy(sorted, filtered)
	m.sorter.Sort(sorted)

	// Update column headers with sort indicator
	columns := m.table.Columns()
	sortIndicator := "↓"
	if !m.sorter.Descending {
		sortIndicator = "↑"
	}

	// Reset all column titles
	columns[0].Title = "PID"
	columns[1].Title = "USER"
	columns[2].Title = "PROGRAM"
	columns[3].Title = "%CPU"
	columns[4].Title = "%MEM"
	columns[5].Title = "VSIZE"
	columns[6].Title = "RSS"
	columns[7].Title = "S"
	columns[8].Title = "TIME+"
	columns[9].Title = "COMMAND"

	// Add indicator to sorted column
	switch m.sorter.Column {
	case model.SortByPID:
		columns[0].Title = "PID " + sortIndicator
	case model.SortByUSER:
		columns[1].Title = "USER " + sortIndicator
	case model.SortByCPUCol:
		columns[3].Title = "%CPU " + sortIndicator
	case model.SortByMEM:
		columns[4].Title = "%MEM " + sortIndicator
	case model.SortByVSIZE:
		columns[5].Title = "VSIZE " + sortIndicator
	case model.SortByRSS:
		columns[6].Title = "RSS " + sortIndicator
	case model.SortByTIME:
		columns[8].Title = "TIME+ " + sortIndicator
	}

	m.table.SetColumns(columns)

	// Preserve current selection by PID across refreshes
	selectedPID := m.getSelectedPID()

	// Build table rows
	rows := make([]table.Row, 0, len(sorted))
	for _, r := range sorted {
		if !r.Alive {
			continue
		}

		cpu := fmt.Sprintf("%.1f", r.CPU)
		mem := fmt.Sprintf("%.1f", r.PMem)

		// Color coding
		if r.CPU > 50 {
			cpu = highCPUStyle.Render(cpu)
		} else if r.CPU > 20 {
			cpu = medCPUStyle.Render(cpu)
		}

		if r.PMem > 10 {
			mem = highCPUStyle.Render(mem)
		} else if r.PMem > 5 {
			mem = medCPUStyle.Render(mem)
		}

		// Separar programa y argumentos
		program := r.Comm
		args := ""

		if r.Cmd != "" {
			// Separar cmdline en programa y argumentos
			parts := strings.SplitN(r.Cmd, " ", 2)
			if len(parts) > 0 {
				// Extraer solo el nombre del ejecutable (sin path)
				fullPath := parts[0]
				if idx := strings.LastIndex(fullPath, "/"); idx >= 0 {
					program = fullPath[idx+1:]
				} else {
					program = fullPath
				}
			}
			if len(parts) > 1 {
				args = parts[1]
			}
		} else {
			// Si no hay cmdline, usar comm entre []
			program = "[" + r.Comm + "]"
		}

		// Truncar si es necesario
		if len(program) > 15 {
			program = program[:12] + "..."
		}
		if len(args) > 45 {
			args = args[:42] + "..."
		}

		// Show CPU time with centiseconds like top's TIME+
		timeStr := FormatTimeTicks(r.CurProcTime, model.DefaultHZ)

		rows = append(rows, table.Row{
			fmt.Sprintf("%d", r.Pid),
			r.User,
			program,
			cpu,
			mem,
			FormatKB(r.VSizeKB),
			FormatKB(r.RSSKB),
			string(r.State),
			timeStr,
			args,
		})

		if len(rows) >= model.MaxRows {
			break
		}
	}

	m.table.SetRows(rows)

	// Restore selection to the same PID if still present
	if selectedPID > 0 && len(rows) > 0 {
		for i := range rows {
			var pid int
			fmt.Sscanf(rows[i][0], "%d", &pid)
			if pid == selectedPID {
				m.table.SetCursor(i)
				break
			}
		}
	}
}

func (m Model) getSelectedPID() int {
	selected := m.table.SelectedRow()
	if len(selected) == 0 {
		return 0
	}

	var pid int
	fmt.Sscanf(selected[0], "%d", &pid)
	return pid
}

func (m Model) showStatus(text string, isError bool) tea.Cmd {
	return func() tea.Msg {
		return statusMsg{text: text, isError: isError}
	}
}

func (m Model) View() string {
	if m.mode == helpMode {
		return m.renderHelp()
	}

	if m.mode == settingsMode {
		return m.renderSettings()
	}

	if m.mode == editThresholdCPU {
		return "Edit CPU Threshold:\n\n" + m.cpuInput.View() + "\n\n[enter=save, esc=cancel]"
	}

	if m.mode == editThresholdMEM {
		return "Edit MEM Threshold:\n\n" + m.memInput.View() + "\n\n[enter=save, esc=cancel]"
	}

	if m.mode == addWebhookMode {
		return m.renderAddWebhook()
	}

	var b strings.Builder

	// Title bar con nombre del programa
	title := titleStyle.Render("🔍 SENTINEL - System Monitor")
	b.WriteString(lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	// Header with system info
	direction := sortedColumnStyle.Render("↓")
	if !m.sorter.Descending {
		direction = sortedColumnStyle.Render("↑")
	}

	header := fmt.Sprintf(
		"Tasks: %d total, %d running | Load: %.2f %.2f %.2f | Uptime: %s | Sort: %s %s",
		m.tasks, m.running, m.l1, m.l5, m.l15,
		FormatUptime(m.uptime),
		sortedColumnStyle.Render(m.sorter.ColumnName()),
		direction,
	)

	if m.filterText != "" {
		header += fmt.Sprintf(" | Filter: %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render(m.filterText))
	}

	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n\n")

	// Table
	b.WriteString(baseStyle.Render(m.table.View()))
	b.WriteString("\n")

	// Quick help
	if m.mode == normalMode {
		quickHelp := fmt.Sprintf(
			"%s Sort | %s Filter | %s Actions | %s Settings | %s Help | %s Quit",
			keybindStyle.Render("[c/m/p/u/v/r/t]"),
			keybindStyle.Render("[/]"),
			keybindStyle.Render("[k/n]"),
			keybindStyle.Render("[s]"),
			keybindStyle.Render("[?]"),
			keybindStyle.Render("[q]"),
		)
		b.WriteString(keybindDescStyle.Render(quickHelp))
		b.WriteString("\n")
	}

	// Status message
	if m.statusText != "" {
		style := successStyle
		if m.statusError {
			style = errorStyle
		}
		b.WriteString("\n")
		b.WriteString(style.Render(m.statusText))
		b.WriteString("\n")
	}

	// Filter input
	if m.mode == filterMode {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render("Filter: "))
		b.WriteString(m.filterInput.View())
		b.WriteString(keybindDescStyle.Render(" (Enter to apply, Esc to cancel)"))
	}

	// Confirmation modals
	if m.mode == confirmKillMode {
		msg := fmt.Sprintf("⚠️  Kill process %d? (y/n)", m.selectedPID)
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(msg))
	}

	if m.mode == confirmNiceMode {
		action := "increase"
		if m.niceValue > 0 {
			action = "decrease"
		}
		msg := fmt.Sprintf("⚙️  %s priority of PID %d by %d? (y/n)",
			strings.Title(action), m.selectedPID, abs(m.niceValue))
		b.WriteString("\n")
		b.WriteString(confirmStyle.Render(msg))
	}

	return b.String()
}

func (m Model) renderHelp() string {
	var b strings.Builder

	title := titleStyle.Render("🔍 SENTINEL - Keyboard Shortcuts")
	b.WriteString(lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render(title))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		keys  []struct{ key, desc string }
	}{
		{
			title: "📊 SORTING",
			keys: []struct{ key, desc string }{
				{"c", "Sort by %CPU"},
				{"m", "Sort by %MEM"},
				{"p", "Sort by PID"},
				{"u", "Sort by USER"},
				{"v", "Sort by VSIZE"},
				{"r", "Sort by RSS"},
				{"t", "Sort by TIME+"},
				{"", "Press same key to toggle ascending/descending"},
			},
		},
		{
			title: "🔍 FILTERING",
			keys: []struct{ key, desc string }{
				{"/", "Enter filter mode"},
				{"Enter", "Apply filter"},
				{"Esc", "Cancel filter"},
				{"", "Filter searches in COMMAND and USER fields"},
			},
		},
		{
			title: "⚙️  PROCESS MANAGEMENT",
			keys: []struct{ key, desc string }{
				{"k", "Send SIGTERM (graceful kill)"},
				{"K", "Send SIGKILL (force kill)"},
				{"n", "Increase priority (nice -5)"},
				{"N", "Decrease priority (nice +5)"},
				{"", "Requires appropriate permissions"},
			},
		},
		{
			title: "🎮 NAVIGATION",
			keys: []struct{ key, desc string }{
				{"↑/↓ or j/k", "Move selection"},
				{"PgUp/PgDn", "Page up/down"},
				{"Home/End", "Go to first/last"},
			},
		},
		{
			title: "📋 GENERAL",
			keys: []struct{ key, desc string }{
				{"s", "Open settings (thresholds & notifications)"},
				{"?/h", "Show/hide this help"},
				{"q", "Quit program"},
				{"Ctrl+C", "Force quit"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("cyan")).
			Bold(true).
			Render(section.title))
		b.WriteString("\n")

		for _, binding := range section.keys {
			if binding.key == "" {
				b.WriteString(keybindDescStyle.Render("  ℹ " + binding.desc))
			} else {
				line := fmt.Sprintf("  %s  %s",
					keybindStyle.Render(lipgloss.NewStyle().Width(12).Render(binding.key)),
					keybindDescStyle.Render(binding.desc))
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(keybindDescStyle.Render("Press any key to return..."))

	return helpBoxStyle.Render(b.String())
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
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

func (m Model) renderSettings() string {
	var b strings.Builder

	b.WriteString("===== SETTINGS =====\n\n")

	b.WriteString(fmt.Sprintf("CPU Threshold: %.0f\n", m.cfg.CPUThreshold))
	b.WriteString(fmt.Sprintf("MEM Threshold: %.0f\n\n", m.cfg.MemThreshold))

	b.WriteString("Webhooks:\n")
	for i, name := range m.webhookNames {
		marker := " "
		if name == m.cfg.ActiveWebhook {
			marker = "*"
		}
		sel := " "
		if i == m.selectedWebhookIndex {
			sel = ">"
		}
		url := m.cfg.Webhooks[name]
		b.WriteString(fmt.Sprintf("%s %s %s → %s\n", sel, marker, name, url))
	}

	b.WriteString("\nActions:\n")
	b.WriteString(" e  Edit CPU Threshold\n")
	b.WriteString(" m  Edit MEM Threshold\n")
	b.WriteString(" a  Add Webhook\n")
	b.WriteString(" d  Delete Webhook\n")
	b.WriteString(" w  Set Selected as Active\n")
	b.WriteString(" q  Back\n")

	return b.String()
}

func (m Model) renderAddWebhook() string {
	var b strings.Builder

	b.WriteString("=== Add Webhook ===\n\n")

	b.WriteString("Name:\n")
	b.WriteString(m.webhookNameInput.View())
	b.WriteString("\n\n")

	b.WriteString("URL:\n")
	b.WriteString(m.webhookURLInput.View())
	b.WriteString("\n\n")

	b.WriteString("[enter = next/save]   [esc = cancel]\n")

	return b.String()
}
