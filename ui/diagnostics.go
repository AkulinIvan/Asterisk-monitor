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
	// Возвращаем команду, а не функцию
	return m.startQuickDiagnostics
}

func (m DiagnosticsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			return m, m.startQuickDiagnostics
		case "f", "F":
			return m, m.startFullDiagnostics
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, nil
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
	case diagnosticsResultMsg:
		m.results = append(m.results, types.CheckResult(msg))
		m.updateContent()
	case startDiagnosticsMsg:
		// Начинаем диагностику
		return m, m.runDiagnosticsChecks(msg.checks)
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
type diagnosticsResultMsg types.CheckResult

type startDiagnosticsMsg struct {
	checks []diagnosticsCheck
}

type diagnosticsCheck struct {
	name string
	cmd  func() tea.Msg
}

// Command functions
func (m DiagnosticsModel) startQuickDiagnostics() tea.Msg {
	m.results = []types.CheckResult{}
	m.updateContent()
	
	checks := []diagnosticsCheck{
		{"Service Status", m.checkServiceStatus},
		{"Asterisk Process", m.checkAsteriskProcess},
		{"SIP Peers", m.checkSIPPeers},
		{"Active Channels", m.checkActiveChannels},
		{"Version Info", m.checkVersion},
	}
	
	return startDiagnosticsMsg{checks: checks}
}

func (m DiagnosticsModel) startFullDiagnostics() tea.Msg {
	m.results = []types.CheckResult{}
	m.updateContent()
	
	checks := []diagnosticsCheck{
		{"Service Status", m.checkServiceStatus},
		{"Asterisk Process", m.checkAsteriskProcess},
		{"SIP Peers", m.checkSIPPeers},
		{"Active Channels", m.checkActiveChannels},
		{"Version Info", m.checkVersion},
		{"Codecs", m.checkCodecs},
		{"Dialplan", m.checkDialplan},
		{"Modules", m.checkModules},
		{"Network", m.checkNetwork},
		{"Ports", m.checkPorts},
		{"System Load", m.checkSystemLoad},
	}
	
	return startDiagnosticsMsg{checks: checks}
}

func (m DiagnosticsModel) runDiagnosticsChecks(checks []diagnosticsCheck) tea.Cmd {
	if len(checks) == 0 {
		return nil
	}
	
	// Берем первую проверку
	check := checks[0]
	remaining := checks[1:]
	
	// Выполняем проверку и планируем следующую
	return tea.Sequence(
		func() tea.Msg {
			return check.cmd()
		},
		func() tea.Msg {
			time.Sleep(300 * time.Millisecond) // Задержка между проверками
			return startDiagnosticsMsg{checks: remaining}
		},
	)
}

// Check functions (возвращают tea.Msg)
func (m DiagnosticsModel) checkServiceStatus() tea.Msg {
	result := m.monitor.ExecuteCommand("Service Status", "systemctl is-active asterisk")
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkAsteriskProcess() tea.Msg {
	result := m.monitor.ExecuteCommand("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1")
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkSIPPeers() tea.Msg {
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
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkActiveChannels() tea.Msg {
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
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkVersion() tea.Msg {
	result := m.monitor.ExecuteCommand("Version Info", "asterisk -rx 'core show version' | head -1")
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkCodecs() tea.Msg {
	result := m.monitor.ExecuteCommand("Codecs", "asterisk -rx 'core show translation' | head -5")
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkDialplan() tea.Msg {
	result := m.monitor.ExecuteCommand("Dialplan", "asterisk -rx 'dialplan show' | grep -c 'Context'")
	if result.Status == "success" {
		result.Message = fmt.Sprintf("%s contexts found", strings.TrimSpace(result.Message))
	}
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkModules() tea.Msg {
	result := m.monitor.ExecuteCommand("Modules", "asterisk -rx 'module show' | grep -c 'Loaded'")
	if result.Status == "success" {
		result.Message = fmt.Sprintf("%s modules loaded", strings.TrimSpace(result.Message))
	}
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkNetwork() tea.Msg {
	result := m.monitor.ExecuteCommand("Network", "ping -c 2 8.8.8.8 | grep 'packet loss' || echo 'Network test failed'")
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkPorts() tea.Msg {
	result := m.monitor.ExecuteCommand("Ports", "netstat -tlnp | grep -E ':(5060|5038)' | grep LISTEN || echo 'No SIP/AMI ports found'")
	return diagnosticsResultMsg(result)
}

func (m DiagnosticsModel) checkSystemLoad() tea.Msg {
	result := m.monitor.ExecuteCommand("System Load", "uptime")
	return diagnosticsResultMsg(result)
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