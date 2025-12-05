package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	switch m.mode {
	case helpMode:
		return m.renderHelp()
	case settingsMode:
		return m.renderSettings()
	case editThresholdCPU:
		return "Edit CPU Threshold:\n\n" + m.cpuInput.View() + "\n\n[enter=save, esc=cancel]"
	case editThresholdMEM:
		return "Edit MEM Threshold:\n\n" + m.memInput.View() + "\n\n[enter=save, esc=cancel]"
	case addWebhookMode:
		return m.renderAddWebhook()
	}

	var b strings.Builder
	b.WriteString(m.renderTitle())
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render(m.renderHeader()))
	b.WriteString("\n\n")
	b.WriteString(baseStyle.Render(m.table.View()))
	b.WriteString("\n")

	if m.mode == normalMode {
		b.WriteString(m.renderQuickHelp())
		b.WriteString("\n")
	}

	if m.statusText != "" {
		b.WriteString("\n")
		b.WriteString(m.renderStatus())
		b.WriteString("\n")
	}

	if m.mode == filterMode {
		b.WriteString("\n")
		b.WriteString(m.renderFilterBar())
	}

	if m.mode == confirmKillMode {
		b.WriteString("\n")
		b.WriteString(m.renderConfirmKill())
	}

	if m.mode == confirmNiceMode {
		b.WriteString("\n")
		b.WriteString(m.renderConfirmNice())
	}

	return b.String()
}

func (m Model) renderTitle() string {
	title := titleStyle.Render("🔍 SENTINEL - System Monitor")
	return lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Width(m.width).
		Align(lipgloss.Center).
		Render(title)
}

func (m Model) renderHeader() string {
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
	return header
}

func (m Model) renderQuickHelp() string {
	quickHelp := fmt.Sprintf(
		"%s Sort | %s Filter | %s Actions | %s Settings | %s Help | %s Quit",
		keybindStyle.Render("[c/m/p/u/v/r/t]"),
		keybindStyle.Render("[/]"),
		keybindStyle.Render("[k/n]"),
		keybindStyle.Render("[s]"),
		keybindStyle.Render("[?]"),
		keybindStyle.Render("[q]"),
	)
	return keybindDescStyle.Render(quickHelp)
}

func (m Model) renderStatus() string {
	style := successStyle
	if m.statusError {
		style = errorStyle
	}
	return style.Render(m.statusText)
}

func (m Model) renderFilterBar() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("cyan")).Render("Filter: ") +
		m.filterInput.View() +
		keybindDescStyle.Render(" (Enter to apply, Esc to cancel)")
}

func (m Model) renderConfirmKill() string {
	msg := fmt.Sprintf("⚠️  Kill process %d? (y/n)", m.selectedPID)
	return confirmStyle.Render(msg)
}

func (m Model) renderConfirmNice() string {
	action := "increase"
	if m.niceValue > 0 {
		action = "decrease"
	}
	msg := fmt.Sprintf("⚙️  %s priority of PID %d by %d? (y/n)",
		strings.Title(action), m.selectedPID, abs(m.niceValue))
	return confirmStyle.Render(msg)
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

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
