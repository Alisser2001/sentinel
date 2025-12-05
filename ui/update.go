package ui

import (
	"fmt"
	"strconv"
	"strings"

	"sentinel/config"
	"sentinel/model"
	"sentinel/proc"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)
const errorFmt = "Error: %v"

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
		m.table.SetHeight(msg.Height - 12)
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
				return m, m.showStatus(fmt.Sprintf(errorFmt, err), true)
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
			return m, m.showStatus(fmt.Sprintf(errorFmt, err), true)
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
			return m, m.showStatus(fmt.Sprintf(errorFmt, err), true)
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
	filtered := m.applyFilter(m.records, m.filterText)

	// Sort on a copy
	sorted := make([]model.ProcRec, len(filtered))
	copy(sorted, filtered)
	m.sorter.Sort(sorted)

	// Update column headers
	m.table.SetColumns(m.buildColumns())

	// Preserve selection
	selectedPID := m.getSelectedPID()
	rows := m.buildRows(sorted)
	m.table.SetRows(rows)
	m.restoreSelection(rows, selectedPID)
}

// buildColumns constructs the table columns with sort indicators applied.
func (m *Model) buildColumns() []table.Column {
	columns := m.table.Columns()
	sortIndicator := "↓"
	if !m.sorter.Descending {
		sortIndicator = "↑"
	}

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
	return columns
}

// buildRows converts sorted process records into table rows with styling and truncation.
func (m *Model) buildRows(sorted []model.ProcRec) []table.Row {
	rows := make([]table.Row, 0, len(sorted))
	for _, r := range sorted {
		if !r.Alive {
			continue
		}

		cpu := fmt.Sprintf("%.1f", r.CPU)
		mem := fmt.Sprintf("%.1f", r.PMem)

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

		program, args := m.programAndArgs(r)

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
	return rows
}

// programAndArgs derives the display program name and arguments from a record.
func (m *Model) programAndArgs(r model.ProcRec) (string, string) {
	program := r.Comm
	args := ""

	if r.Cmd != "" {
		parts := strings.SplitN(r.Cmd, " ", 2)
		if len(parts) > 0 {
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
		program = "[" + r.Comm + "]"
	}

	if len(program) > 15 {
		program = program[:12] + "..."
	}
	if len(args) > 45 {
		args = args[:42] + "..."
	}
	return program, args
}

// restoreSelection moves the cursor back to the previously selected PID if present.
func (m *Model) restoreSelection(rows []table.Row, selectedPID int) {
	if selectedPID <= 0 || len(rows) == 0 {
		return
	}
	for i := range rows {
		var pid int
		fmt.Sscanf(rows[i][0], "%d", &pid)
		if pid == selectedPID {
			m.table.SetCursor(i)
			break
		}
	}
}

// applyFilter returns a filtered slice of process records based on the provided text.
// When text is empty, returns the input records unchanged.
func (m *Model) applyFilter(records []model.ProcRec, text string) []model.ProcRec {
	if text == "" {
		return records
	}

	searchLower := strings.ToLower(text)
	filtered := make([]model.ProcRec, 0, len(records))
	for _, r := range records {
		if !r.Alive {
			continue
		}
		cmd := strings.ToLower(r.Cmd)
		user := strings.ToLower(r.User)
		comm := strings.ToLower(r.Comm)
		if strings.Contains(cmd, searchLower) ||
			strings.Contains(user, searchLower) ||
			strings.Contains(comm, searchLower) {
			filtered = append(filtered, r)
		}
	}
	return filtered
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
