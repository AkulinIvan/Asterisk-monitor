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
	// Инициализируем viewport с минимальными размерами
	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))
	
	return DiagnosticsModel{
		monitor:  mon,
		viewport: vp,
		results:  []types.CheckResult{},
		ready:    false, // все равно ждем WindowSizeMsg для точных размеров
	}
}

func (m DiagnosticsModel) Init() tea.Cmd {
	// Сразу обновляем контент при инициализации
	return m.initializeContent
}

func (m DiagnosticsModel) initializeContent() tea.Msg {
	// Просто сообщение для обновления контента
	return contentInitializedMsg{}
}

type contentInitializedMsg struct{}

func (m DiagnosticsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			// Быстрая диагностика
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, tea.Sequence(
				m.runCheck("Service Status", "systemctl is-active asterisk"),
				m.delay(300),
				m.runCheck("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1"),
				m.delay(300),
				m.runSIPCheckCmd(),
				m.delay(300),
				m.runChannelsCheckCmd(),
				m.delay(300),
				m.runCheck("Version Info", "asterisk -rx 'core show version' | head -1"),
			)
		case "f", "F":
			// Полная диагностика
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, tea.Sequence(
				m.runCheck("Service Status", "systemctl is-active asterisk"),
				m.delay(200),
				m.runCheck("Asterisk Process", "ps aux | grep -v grep | grep asterisk | head -1"),
				m.delay(200),
				m.runSIPCheckCmd(),
				m.delay(200),
				m.runChannelsCheckCmd(),
				m.delay(200),
				m.runCheck("Version Info", "asterisk -rx 'core show version' | head -1"),
				m.delay(200),
				m.runCheck("Codecs", "asterisk -rx 'core show translation' | head -5"),
				m.delay(200),
				m.runCheck("Dialplan", "asterisk -rx 'dialplan show' | grep -c 'Context'"),
				m.delay(200),
				m.runCheck("Modules", "asterisk -rx 'module show' | grep -c 'Loaded'"),
				m.delay(200),
				m.runCheck("Network", "ping -c 2 8.8.8.8 | grep 'packet loss' || echo 'Network test failed'"),
				m.delay(200),
				m.runCheck("Ports", "netstat -tlnp | grep -E ':(5060|5038)' | grep LISTEN || echo 'No SIP/AMI ports found'"),
				m.delay(200),
				m.runCheck("System Load", "uptime"),
			)
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
			return m, nil
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			// Первая инициализация с реальными размерами окна
			m.viewport = viewport.New(msg.Width, msg.Height-4)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
		} else {
			// Обновление размеров
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 4
		}
		m.updateContent()
		return m, nil
	case checkResultMsg:
		m.results = append(m.results, types.CheckResult(msg))
		m.updateContent()
		return m, nil
	case contentInitializedMsg:
		m.updateContent()
		return m, nil
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
func (m DiagnosticsModel) delay(ms int) tea.Cmd {
	return tea.Tick(time.Duration(ms)*time.Millisecond, func(t time.Time) tea.Msg {
		return nil
	})
}

func (m DiagnosticsModel) runCheck(name, command string) tea.Cmd {
	return func() tea.Msg {
		result := m.monitor.ExecuteCommand(name, command)
		return checkResultMsg(result)
	}
}

func (m DiagnosticsModel) runSIPCheckCmd() tea.Cmd {
	return func() tea.Msg {
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
}

func (m DiagnosticsModel) runChannelsCheckCmd() tea.Cmd {
	return func() tea.Msg {
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