package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DebugModel struct {
	monitor   MonitorInterface
	viewport  viewport.Model
	debugLogs string
	filter    string
	isRunning bool
	ready     bool
}

func NewDebugModel(mon MonitorInterface) DebugModel {
	vp := viewport.New(120, 30)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return DebugModel{
		monitor:   mon,
		viewport:  vp,
		debugLogs: "",
		filter:    "ERROR|WARNING|failed|reject|timeout|busy|congestion",
		isRunning: false,
		ready:     true,
	}
}

func (m DebugModel) Init() tea.Cmd {
	m.updateContent()
	return nil
}

func (m DebugModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "s", "S":
			m.startDebug()
			return m, nil
		case "x", "X":
			m.stopDebug()
			return m, nil
		case "c", "C":
			m.debugLogs = ""
			m.updateContent()
			return m, nil
		case "f", "F":
			m.toggleFilter()
			return m, nil
		case "r", "R":
			m.refreshDebug()
			return m, nil
		case "q", "Q", "ctrl+c":
			m.stopDebug()
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}
		m.updateContent()
	case debugUpdateMsg:
		if m.isRunning {
			newLogs := string(msg)
			if m.filter != "" {
				newLogs = m.filterDebugLogs(newLogs)
			}
			if strings.TrimSpace(newLogs) != "" {
				m.debugLogs = newLogs + "\n" + m.debugLogs
				// Ограничиваем размер логов
				lines := strings.Split(m.debugLogs, "\n")
				if len(lines) > 100 {
					m.debugLogs = strings.Join(lines[:100], "\n")
				}
				m.updateContent()
			}
			// Продолжаем обновление если все еще запущено
			if m.isRunning {
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return m.getDebugLogs()
				})
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DebugModel) View() string {
	if !m.ready {
		return "Initializing debug..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

// Messages
type debugUpdateMsg string

// Debug functions
func (m *DebugModel) startDebug() {
	if m.isRunning {
		return
	}

	m.isRunning = true

	// Включаем дебаг режимы
	commands := []string{
		"asterisk -rx 'sip set debug on'",
		"asterisk -rx 'rtp set debug on'",
		"asterisk -rx 'core set debug 1'",
	}

	for _, cmd := range commands {
		m.monitor.ExecuteCommand("Enable Debug", cmd)
	}

	m.debugLogs = "=== DEBUG MODE STARTED ===\n"
	m.debugLogs += "SIP Debug: ON\n"
	m.debugLogs += "RTP Debug: ON\n"
	m.debugLogs += "Core Debug: Level 1\n"
	m.debugLogs += "Filter: " + m.filter + "\n"
	m.debugLogs += "========================\n\n"

	m.updateContent()

	// Запускаем получение логов
	go func() {
		time.Sleep(1 * time.Second)
		m.getDebugLogs()
	}()
}

func (m *DebugModel) stopDebug() {
	if !m.isRunning {
		return
	}

	m.isRunning = false

	// Выключаем дебаг режимы
	commands := []string{
		"asterisk -rx 'sip set debug off'",
		"asterisk -rx 'rtp set debug off'",
		"asterisk -rx 'core set debug 0'",
	}

	for _, cmd := range commands {
		m.monitor.ExecuteCommand("Disable Debug", cmd)
	}

	m.debugLogs += "\n=== DEBUG MODE STOPPED ===\n"
	m.updateContent()
}

func (m *DebugModel) refreshDebug() {
	if m.isRunning {
		m.getDebugLogs()
	}
}

func (m *DebugModel) toggleFilter() {
	if m.filter == "" {
		m.filter = "ERROR|WARNING|failed|reject|timeout|busy|congestion|INVITE|BYE|REGISTER"
	} else if m.filter == "ERROR|WARNING|failed|reject|timeout|busy|congestion|INVITE|BYE|REGISTER" {
		m.filter = "ERROR|WARNING|failed"
	} else {
		m.filter = ""
	}
	m.updateContent()
}

func (m *DebugModel) getDebugLogs() tea.Msg {
	if !m.isRunning {
		return nil
	}

	// Получаем логи с фильтрацией проблем
	cmd := fmt.Sprintf(
		"timeout 5 asterisk -rvvv 2>&1 | grep -E '%s' | head -20 || echo 'No debug output'",
		m.filter,
	)

	result := m.monitor.ExecuteCommand("Debug Logs", cmd)

	if result.Status == "success" && strings.TrimSpace(result.Message) != "" {
		return debugUpdateMsg(result.Message)
	}

	return debugUpdateMsg("... waiting for debug events ...")
}

func (m *DebugModel) filterDebugLogs(logs string) string {
	if m.filter == "" {
		return logs
	}

	lines := strings.Split(logs, "\n")
	var filtered []string

	for _, line := range lines {
		if containsAny(line, strings.Split(m.filter, "|")) {
			// Подсвечиваем ключевые слова
			line = m.highlightProblems(line)
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}

func (m *DebugModel) highlightProblems(line string) string {
	problems := []string{
		"ERROR", "WARNING", "failed", "reject", "timeout",
		"busy", "congestion", "INVITE", "BYE", "REGISTER",
	}

	for _, problem := range problems {
		if strings.Contains(strings.ToUpper(line), strings.ToUpper(problem)) {
			var style lipgloss.Style
			switch {
			case strings.Contains(strings.ToUpper(problem), "ERROR"):
				style = errorStyle
			case strings.Contains(strings.ToUpper(problem), "WARNING"):
				style = warningStyle
			case strings.Contains(strings.ToUpper(problem), "FAILED"):
				style = errorStyle
			case strings.Contains(strings.ToUpper(problem), "REJECT"):
				style = warningStyle
			case strings.Contains(strings.ToUpper(problem), "TIMEOUT"):
				style = warningStyle
			default:
				style = infStyle
			}
			line = strings.ReplaceAll(line, problem, style.Render(problem))
		}
	}

	return line
}

func containsAny(text string, keywords []string) bool {
	textUpper := strings.ToUpper(text)
	for _, keyword := range keywords {
		if strings.Contains(textUpper, strings.ToUpper(keyword)) {
			return true
		}
	}
	return false
}

func (m *DebugModel) updateContent() {
	if !m.ready {
		return
	}

	var content strings.Builder

	content.WriteString(TitleStyle.Render("🐛 Asterisk Real-time Debug"))
	content.WriteString("\n\n")

	// Статус
	status := "🔴 STOPPED"
	if m.isRunning {
		status = "🟢 RUNNING"
	}
	content.WriteString(fmt.Sprintf("Status: %s | Filter: %s\n\n", status, m.filter))

	if m.debugLogs == "" {
		content.WriteString("No debug data collected yet.\n\n")
		content.WriteString(m.renderDebugInfo())
	} else {
		content.WriteString(m.debugLogs)
	}

	m.viewport.SetContent(content.String())
}

func (m *DebugModel) renderDebugInfo() string {
	info := `Real-time Debug Monitor:

This module enables Asterisk debug modes and shows only problematic events:

🔍 Monitored Problems:
• SIP Errors & Warnings
• RTP Issues  
• Call Failures
• Registration Problems
• Timeouts & Rejects
• Busy/Congestion

🎯 Debug Modes Enabled:
• SIP Debug: Detailed SIP messaging
• RTP Debug: RTP/audio stream issues  
• Core Debug: General Asterisk problems

⚡ Commands:
• S - Start debugging
• X - Stop debugging  
• R - Refresh logs
• F - Toggle filters
• C - Clear logs
• Q - Quit

💡 Tips:
• Start debug when you have issues
• Watch for colored problem keywords
• Use filters to focus on specific issues`

	return borderStyle.Render(info)
}

func (m *DebugModel) footer() string {
	status := "STOPPED"
	if m.isRunning {
		status = "RUNNING 🟢"
	}

	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render(fmt.Sprintf("Status: %s | S:Start X:Stop R:Refresh F:Filter C:Clear Q:Quit", status))
}
