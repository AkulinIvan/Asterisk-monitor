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
	// Просто инициализируем, без автоматического запуска диагностики
	return nil
}

func (m DiagnosticsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			m.results = []types.CheckResult{}
			m.updateContent()
			// Запускаем быструю диагностику последовательно
			return m, tea.Sequence(
				m.delayCmd(100),
				m.runCheck("Service Status", "systemctl is-active asterisk"),
				m.delayCmd(200),
				m.runCheck("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1"),
				m.delayCmd(200),
				m.runSIPCheck,
				m.delayCmd(200),
				m.runChannelsCheck,
				m.delayCmd(200),
				m.runCheck("Version Info", "asterisk -rx 'core show version' | head -1"),
			)
		case "f", "F":
			m.results = []types.CheckResult{}
			m.updateContent()
			// Запускаем полную диагностику
			return m, tea.Sequence(
				m.delayCmd(100),
				m.runCheck("Service Status", "systemctl is-active asterisk"),
				m.delayCmd(200),
				m.runCheck("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1"),
				m.delayCmd(200),
				m.runSIPCheck,
				m.delayCmd(200),
				m.runChannelsCheck,
				m.delayCmd(200),
				m.runCheck("Version Info", "asterisk -rx 'core show version' | head -1"),
				m.delayCmd(200),
				m.runCheck("Codecs", "asterisk -rx 'core show translation' | head -5"),
				m.delayCmd(200),
				m.runCheck("Dialplan", "asterisk -rx 'dialplan show' | grep -c 'Context'"),
				m.delayCmd(200),
				m.runCheck("Modules", "asterisk -rx 'module show' | grep -c 'Loaded'"),
				m.delayCmd(200),
				m.runCheck("Network", "ping -c 2 8.8.8.8 | grep 'packet loss' || echo 'Network test failed'"),
				m.delayCmd(200),
				m.runCheck("Ports", "netstat -tlnp | grep -E ':(5060|5038)' | grep LISTEN || echo 'No SIP/AMI ports found'"),
				m.delayCmd(200),
				m.runCheck("System Load", "uptime"),
			)
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
			m.updateContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
	case checkResultMsg:
		m.results = append(m.results, types.CheckResult(msg))
		m.updateContent()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DiagnosticsModel) View() string {
	if !m.ready {
		return "Initializing diagnostics..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

// Messages
type checkResultMsg types.CheckResult

// Command functions
func (m DiagnosticsModel) delayCmd(ms time.Duration) tea.Cmd {
	return tea.Tick(ms*time.Millisecond, func(t time.Time) tea.Msg {
		return nil
	})
}

func (m DiagnosticsModel) runCheck(name, command string) tea.Cmd {
	return func() tea.Msg {
		result := m.monitor.ExecuteCommand(name, command)
		return checkResultMsg(result)
	}
}

func (m DiagnosticsModel) runSIPCheck() tea.Msg {
	online, total := m.monitor.GetSIPPeersCount()
	result := types.CheckResult{
		Name:      "SIP Peers",
		Status:    "success",
		Message:   fmt.Sprintf("%d online out of %d total", online, total),
		Timestamp: time.Now(),
	}
	if online == 0 && total > 0 {
		result.Status = "warning"
		result.Message = fmt.Sprintf("No peers online (total: %d)", total)
	}
	return checkResultMsg(result)
}

func (m DiagnosticsModel) runChannelsCheck() tea.Msg {
	count := m.monitor.GetActiveCallsCount()
	result := types.CheckResult{
		Name:      "Active Channels",
		Status:    "success",
		Message:   fmt.Sprintf("%d active channels", count),
		Timestamp: time.Now(),
	}
	if count > 10 {
		result.Status = "warning"
		result.Message = fmt.Sprintf("High channel count: %d", count)
	}
	return checkResultMsg(result)
}

func (m *DiagnosticsModel) updateContent() {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("🔍 Asterisk Diagnostics"))
	content.WriteString("\n\n")

	if len(m.results) == 0 {
		content.WriteString("No diagnostics run yet.\n\n")
		content.WriteString("Press 'r' for quick check\n")
		content.WriteString("Press 'f' for full diagnostics\n")
		content.WriteString("Press 'c' to clear results\n")
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
		builder.WriteString("\n--- Summary ---\n")
		builder.WriteString(fmt.Sprintf("✅ Success: %d | ⚠️ Warning: %d | ❌ Errors: %d\n",
			successCount, warningCount, errorCount))
	}

	return borderStyle.Render(builder.String())
}

func (m *DiagnosticsModel) footer() string {
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render("Press 'r' for quick check, 'f' for full diagnostics, 'c' to clear, 'q' to quit")
}