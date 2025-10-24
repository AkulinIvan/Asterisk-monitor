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
	return DiagnosticsModel{
		monitor:  mon,
		viewport: viewport.New(0, 0), // временные размеры
		results:  []types.CheckResult{},
		ready:    false,
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
			// Быстрая диагностика
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, tea.Batch(
				m.runCheckCmd("Service Status", "systemctl is-active asterisk"),
				tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1")
				}),
				tea.Tick(600*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runSIPCheck()
				}),
				tea.Tick(900*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runChannelsCheck()
				}),
				tea.Tick(1200*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Version Info", "asterisk -rx 'core show version' | head -1")
				}),
			)
		case "f", "F":
			// Полная диагностика
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, tea.Batch(
				m.runCheckCmd("Service Status", "systemctl is-active asterisk"),
				tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1")
				}),
				tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runSIPCheck()
				}),
				tea.Tick(600*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runChannelsCheck()
				}),
				tea.Tick(800*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Version Info", "asterisk -rx 'core show version' | head -1")
				}),
				tea.Tick(1000*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Codecs", "asterisk -rx 'core show translation' | head -5")
				}),
				tea.Tick(1200*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Dialplan", "asterisk -rx 'dialplan show' | grep -c 'Context'")
				}),
				tea.Tick(1400*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Modules", "asterisk -rx 'module show' | grep -c 'Loaded'")
				}),
				tea.Tick(1600*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Network", "ping -c 2 8.8.8.8 | grep 'packet loss' || echo 'Network test failed'")
				}),
				tea.Tick(1800*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("Ports", "netstat -tlnp | grep -E ':(5060|5038)' | grep LISTEN || echo 'No SIP/AMI ports found'")
				}),
				tea.Tick(2000*time.Millisecond, func(t time.Time) tea.Msg {
					return m.runCheck("System Load", "uptime")
				}),
			)
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
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
	case checkResultMsg:
		m.results = append(m.results, types.CheckResult(msg))
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

// Messages
type checkResultMsg types.CheckResult

// Command functions
func (m DiagnosticsModel) runCheckCmd(name, command string) tea.Cmd {
	return func() tea.Msg {
		result := m.monitor.ExecuteCommand(name, command)
		return checkResultMsg(result)
	}
}

func (m DiagnosticsModel) runCheck(name, command string) checkResultMsg {
	result := m.monitor.ExecuteCommand(name, command)
	return checkResultMsg(result)
}

func (m DiagnosticsModel) runSIPCheck() checkResultMsg {
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

func (m DiagnosticsModel) runChannelsCheck() checkResultMsg {
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