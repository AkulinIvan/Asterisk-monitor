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
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	
	return DiagnosticsModel{
		monitor:  mon,
		viewport: vp,
		results:  []types.CheckResult{},
		ready:    true, // Сразу готов к работе
	}
}

func (m DiagnosticsModel) Init() tea.Cmd {
	m.updateContent()
	return nil
}

func (m DiagnosticsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			m.runQuickDiagnostics()
			return m, nil
		case "f", "F":
			m.runFullDiagnostics()
			return m, nil
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, nil
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
		}
		m.updateContent()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiagnosticsModel) View() string {
	if !m.ready {
		return "\nInitializing diagnostics..."
	}

	header := TitleStyle.Render("🔍 Asterisk Diagnostics") + "\n\n"
	footer := "\n" + m.footer()
	
	return header + m.viewport.View() + footer
}

func (m *DiagnosticsModel) runQuickDiagnostics() {
	m.results = []types.CheckResult{}
	m.updateContent()

	// Просто выполняем проверки синхронно
	checks := []struct {
		name string
		cmd  string
	}{
		{"Service Status", "systemctl is-active asterisk"},
		{"Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1"},
		{"Version Info", "asterisk -rx 'core show version' | head -1"},
	}

	for _, check := range checks {
		result := m.monitor.ExecuteCommand(check.name, check.cmd)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(300 * time.Millisecond)
	}

	// Добавляем SIP и каналы
	online, total := m.monitor.GetSIPPeersCount()
	sipResult := types.CheckResult{
		Name:      "SIP Peers",
		Status:    "success",
		Message:   fmt.Sprintf("%d online out of %d total", online, total),
		Timestamp: time.Now(),
	}
	if online == 0 && total > 0 {
		sipResult.Status = "warning"
		sipResult.Message = fmt.Sprintf("No peers online (total: %d)", total)
	}
	m.results = append(m.results, sipResult)
	m.updateContent()

	count := m.monitor.GetActiveCallsCount()
	channelsResult := types.CheckResult{
		Name:      "Active Channels",
		Status:    "success",
		Message:   fmt.Sprintf("%d active channels", count),
		Timestamp: time.Now(),
	}
	if count > 10 {
		channelsResult.Status = "warning"
		channelsResult.Message = fmt.Sprintf("High channel count: %d", count)
	}
	m.results = append(m.results, channelsResult)
	m.updateContent()
}

func (m *DiagnosticsModel) runFullDiagnostics() {
	m.results = []types.CheckResult{}
	m.updateContent()

	checks := []struct {
		name string
		cmd  string
	}{
		{"Service Status", "systemctl is-active asterisk"},
		{"Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1"},
		{"Version Info", "asterisk -rx 'core show version' | head -1"},
		{"Codecs", "asterisk -rx 'core show translation' | head -5"},
		{"Dialplan", "asterisk -rx 'dialplan show' | grep -c 'Context'"},
		{"Modules", "asterisk -rx 'module show' | grep -c 'Loaded'"},
		{"Network", "ping -c 2 8.8.8.8 | grep 'packet loss' || echo 'Network test failed'"},
		{"Ports", "netstat -tlnp | grep -E ':(5060|5038)' | grep LISTEN || echo 'No SIP/AMI ports found'"},
		{"System Load", "uptime"},
	}

	for _, check := range checks {
		result := m.monitor.ExecuteCommand(check.name, check.cmd)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(200 * time.Millisecond)
	}

	// Добавляем SIP и каналы
	online, total := m.monitor.GetSIPPeersCount()
	sipResult := types.CheckResult{
		Name:      "SIP Peers",
		Status:    "success",
		Message:   fmt.Sprintf("%d online out of %d total", online, total),
		Timestamp: time.Now(),
	}
	if online == 0 && total > 0 {
		sipResult.Status = "warning"
		sipResult.Message = fmt.Sprintf("No peers online (total: %d)", total)
	}
	m.results = append(m.results, sipResult)
	m.updateContent()

	count := m.monitor.GetActiveCallsCount()
	channelsResult := types.CheckResult{
		Name:      "Active Channels",
		Status:    "success",
		Message:   fmt.Sprintf("%d active channels", count),
		Timestamp: time.Now(),
	}
	if count > 10 {
		channelsResult.Status = "warning"
		channelsResult.Message = fmt.Sprintf("High channel count: %d", count)
	}
	m.results = append(m.results, channelsResult)
	m.updateContent()
}

func (m *DiagnosticsModel) updateContent() {
	if !m.ready {
		return
	}

	var content strings.Builder

	if len(m.results) == 0 {
		content.WriteString("No diagnostics run yet.\n\n")
		content.WriteString("Available commands:\n")
		content.WriteString("• Press 'r' for quick check\n")
		content.WriteString("• Press 'f' for full diagnostics\n") 
		content.WriteString("• Press 'c' to clear results\n")
		content.WriteString("• Press 'q' to quit\n")
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
	}

	// Summary
	if len(m.results) > 0 {
		builder.WriteString("\n" + strings.Repeat("─", 40) + "\n")
		builder.WriteString(fmt.Sprintf("📊 Summary: ✅ %d | ⚠️ %d | ❌ %d\n",
			successCount, warningCount, errorCount))
	}

	return builder.String()
}

func (m *DiagnosticsModel) footer() string {
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render("Press 'r' for quick check, 'f' for full diagnostics, 'c' to clear, 'q' to quit")
}