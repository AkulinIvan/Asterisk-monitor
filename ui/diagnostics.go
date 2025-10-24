package ui

import (
	"asterisk-monitor/types"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DiagnosticsModel struct {
	monitor  MonitorInterface
	viewport viewport.Model
	results  []types.CheckResult
	ready    bool
}

func NewDiagnosticsModel(mon MonitorInterface) DiagnosticsModel {
	vp := viewport.New(80, 20)
	return DiagnosticsModel{
		monitor:  mon,
		viewport: vp,
		results:  []types.CheckResult{},
	}
}

func (m DiagnosticsModel) Init() tea.Cmd {
	return nil
}

func (m DiagnosticsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			m.runQuickDiagnostics()
		case "f", "F":
			m.runFullDiagnostics()
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-2)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
		m.updateContent()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiagnosticsModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

func (m *DiagnosticsModel) runQuickDiagnostics() {
	m.results = []types.CheckResult{}
	m.updateContent()

	checks := []struct {
		name string
		cmd  string
	}{
		{"Service Status", "systemctl is-active asterisk"},
		{"Asterisk Process", "ps aux | grep -v grep | grep asterisk"},
		{"SIP Peers", "asterisk -rx 'sip show peers' | grep -c OK"},
		{"Active Channels", "asterisk -rx 'core show channels' | grep 'active channel'"},
		{"Version Info", "asterisk -rx 'core show version' | head -1"},
	}

	for _, check := range checks {
		result := m.monitor.ExecuteCommand(check.name, check.cmd)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(500 * time.Millisecond) // Visual feedback
	}
}

func (m *DiagnosticsModel) runFullDiagnostics() {
	m.results = []types.CheckResult{}
	m.updateContent()

	checks := []struct {
		name string
		cmd  string
	}{
		{"Asterisk Process", "ps aux | grep -v grep | grep asterisk"},
		{"Service Status", "systemctl is-active asterisk"},
		{"SIP Registration", "asterisk -rx 'sip show peers' | grep -c OK"},
		{"Active Calls", "asterisk -rx 'core show channels' | grep 'active channel'"},
		{"Codecs", "asterisk -rx 'core show translation' | head -10"},
		{"Dialplan", "asterisk -rx 'dialplan show' | grep -c 'Context'"},
		{"Modules", "asterisk -rx 'module show' | grep -c 'Loaded'"},
		{"Network", "ping -c 2 8.8.8.8 | grep 'packet loss'"},
		{"Ports", "netstat -tlnp | grep -E ':(5060|5038)' | grep LISTEN"},
		{"System Load", "uptime"},
	}

	for _, check := range checks {
		result := m.monitor.ExecuteCommand(check.name, check.cmd)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(300 * time.Millisecond)
	}
}

func (m *DiagnosticsModel) updateContent() {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("🔍 Asterisk Diagnostics"))
	content.WriteString("\n\n")

	if len(m.results) == 0 {
		content.WriteString("No diagnostics run yet. Press 'r' for quick check or 'f' for full diagnostics.\n")
	} else {
		content.WriteString(m.renderResults())
	}

	m.viewport.SetContent(content.String())
}

func (m *DiagnosticsModel) renderResults() string {
	var builder strings.Builder

	successCount := 0
	warningCount := 0
	errorCount := 0

	for _, result := range m.results {
		var statusIcon string
		switch result.Status {
		case "success":
			statusIcon = "✅"
			successCount++
		case "warning":
			statusIcon = "⚠️"
			warningCount++
		case "error":
			statusIcon = "❌"
			errorCount++
		default:
			statusIcon = "🔍"
		}

		builder.WriteString(fmt.Sprintf("%s %s: %s\n", statusIcon, result.Name, result.Message))
		if result.Error != "" {
			builder.WriteString(fmt.Sprintf("   Error: %s\n", result.Error))
		}
		builder.WriteString("\n")
	}

	// Summary
	builder.WriteString("--- Summary ---\n")
	builder.WriteString(fmt.Sprintf("✅ Success: %d | ⚠️ Warning: %d | ❌ Errors: %d\n",
		successCount, warningCount, errorCount))

	return borderStyle.Render(builder.String())
}

func (m *DiagnosticsModel) footer() string {
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render("Press 'r' for quick check, 'f' for full diagnostics, 'c' to clear, 'q' to quit")
}
