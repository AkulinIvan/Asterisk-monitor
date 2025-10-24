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
	monitor    MonitorInterface
	viewport   viewport.Model
	debugLogs  string
	filter     string
	isRunning  bool
	debugMode  string // "basic", "audio", "full"
	audioStats string
	ready      bool
}

func NewDebugModel(mon MonitorInterface) DebugModel {
	vp := viewport.New(120, 30)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return DebugModel{
		monitor:    mon,
		viewport:   vp,
		debugLogs:  "",
		filter:     "ERROR|WARNING|failed|reject|timeout|busy|congestion|jitter|packet loss",
		isRunning:  false,
		debugMode:  "basic",
		audioStats: "",
		ready:      true,
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
		case "a", "A":
			m.startAudioDebug()
			return m, nil
		case "c", "C":
			m.debugLogs = ""
			m.audioStats = ""
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
	case audioStatsMsg:
		if m.isRunning && m.debugMode == "audio" {
			m.audioStats = string(msg)
			m.updateContent()

			// Продолжаем сбор аудиостатистики
			if m.isRunning {
				return m, tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
					return m.getAudioStats()
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
type audioStatsMsg string

// Debug functions
func (m *DebugModel) startDebug() {
	if m.isRunning {
		return
	}

	m.isRunning = true
	m.debugMode = "basic"

	// Включаем базовые дебаг режимы
	commands := []string{
		"asterisk -rx 'sip set debug on'",
		"asterisk -rx 'rtp set debug on'",
		"asterisk -rx 'core set debug 1'",
	}

	for _, cmd := range commands {
		m.monitor.ExecuteCommand("Enable Debug", cmd)
	}

	m.debugLogs = "=== BASIC DEBUG MODE STARTED ===\n"
	m.debugLogs += "SIP Debug: ON\n"
	m.debugLogs += "RTP Debug: ON\n"
	m.debugLogs += "Core Debug: Level 1\n"
	m.debugLogs += "Filter: " + m.filter + "\n"
	m.debugLogs += "==============================\n\n"

	m.updateContent()

	// Запускаем получение логов
	go func() {
		time.Sleep(1 * time.Second)
		m.getDebugLogs()
	}()
}

func (m *DebugModel) startAudioDebug() {
	if m.isRunning {
		m.stopDebug()
		time.Sleep(1 * time.Second)
	}

	m.isRunning = true
	m.debugMode = "audio"

	// Включаем расширенные дебаг режимы для аудио проблем
	commands := []string{
		"asterisk -rx 'sip set debug on'",
		"asterisk -rx 'rtp set debug on'",
		"asterisk -rx 'rtcp set debug on'",
		"asterisk -rx 'core set debug 3'",
		"asterisk -rx 'jitterbuffer set debug on'",
	}

	for _, cmd := range commands {
		m.monitor.ExecuteCommand("Enable Audio Debug", cmd)
	}

	m.debugLogs = "=== AUDIO DEBUG MODE STARTED ===\n"
	m.debugLogs += "🔊 Focus: Audio Quality Issues\n"
	m.debugLogs += "SIP Debug: ON\n"
	m.debugLogs += "RTP Debug: ON\n"
	m.debugLogs += "RTCP Debug: ON\n"
	m.debugLogs += "Jitterbuffer Debug: ON\n"
	m.debugLogs += "Core Debug: Level 3\n"
	m.debugLogs += "Filter: " + m.filter + "\n"
	m.debugLogs += "================================\n\n"

	m.updateContent()

	// Запускаем получение логов и статистики
	go func() {
		time.Sleep(1 * time.Second)
		m.getDebugLogs()
		m.getAudioStats()
	}()
}

func (m *DebugModel) stopDebug() {
	if !m.isRunning {
		return
	}

	m.isRunning = false

	// Выключаем все дебаг режимы
	commands := []string{
		"asterisk -rx 'sip set debug off'",
		"asterisk -rx 'rtp set debug off'",
		"asterisk -rx 'rtcp set debug off'",
		"asterisk -rx 'core set debug 0'",
		"asterisk -rx 'jitterbuffer set debug off'",
	}

	for _, cmd := range commands {
		m.monitor.ExecuteCommand("Disable Debug", cmd)
	}

	m.debugLogs += "\n=== DEBUG MODE STOPPED ===\n"
	m.audioStats = ""
	m.updateContent()
}

func (m *DebugModel) refreshDebug() {
	if m.isRunning {
		m.getDebugLogs()
		if m.debugMode == "audio" {
			m.getAudioStats()
		}
	}
}

func (m *DebugModel) toggleFilter() {
	if m.filter == "" {
		m.filter = "ERROR|WARNING|failed|reject|timeout|busy|congestion|jitter|packet loss"
	} else if m.filter == "ERROR|WARNING|failed|reject|timeout|busy|congestion|jitter|packet loss" {
		m.filter = "jitter|packet loss|dropped|out of order|buffer"
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

func (m *DebugModel) getAudioStats() tea.Msg {
	if !m.isRunning || m.debugMode != "audio" {
		return nil
	}

	// Собираем расширенную статистику по аудио проблемам
	commands := []string{
		// Статистика RTP
		"asterisk -rx 'rtp show stats' 2>/dev/null | head -10",
		// Активные RTP сессии
		"asterisk -rx 'rtp show peers' 2>/dev/null | head -10",
		// Проблемы с кодеками
		"asterisk -rx 'core show translation' 2>/dev/null | grep -E '(ulaw|alaw|gsm|g729)'",
		// Статус джиттер-буферов
		"asterisk -rx 'jitterbuffer show' 2>/dev/null | head -5",
		// Сетевые проблемы
		"ping -c 2 8.8.8.8 2>/dev/null | grep 'packet loss' || echo 'Network check failed'",
		// Нагрузка системы
		"top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1",
	}

	var stats strings.Builder
	stats.WriteString("=== AUDIO QUALITY STATS ===\n\n")

	for i, cmd := range commands {
		result := m.monitor.ExecuteCommand(fmt.Sprintf("AudioStat%d", i), cmd)
		if result.Status == "success" && strings.TrimSpace(result.Message) != "" {
			switch i {
			case 0:
				stats.WriteString("📊 RTP Statistics:\n")
			case 1:
				stats.WriteString("\n🔗 RTP Sessions:\n")
			case 2:
				stats.WriteString("\n🎵 Codec Status:\n")
			case 3:
				stats.WriteString("\n📈 Jitter Buffers:\n")
			case 4:
				stats.WriteString("\n🌐 Network:\n")
			case 5:
				stats.WriteString(fmt.Sprintf("\n💻 CPU Load: %s%%\n", result.Message))
				continue
			}
			stats.WriteString(result.Message + "\n")
		}
	}

	return audioStatsMsg(stats.String())
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
			line = m.highlightAudioProblems(line)
			filtered = append(filtered, line)
		}
	}

	return strings.Join(filtered, "\n")
}

func (m *DebugModel) highlightAudioProblems(line string) string {
	// Критические проблемы аудио
	critical := []string{"jitter", "packet loss", "dropped", "out of order", "buffer over", "underrun"}
	// Проблемы среднего уровня
	warnings := []string{"WARNING", "failed", "reject", "timeout", "busy", "congestion"}
	// Информационные события
	info := []string{"RTP", "RTCP", "JitterBuffer", "Codec"}

	for _, problem := range critical {
		if strings.Contains(strings.ToUpper(line), strings.ToUpper(problem)) {
			line = strings.ReplaceAll(line, problem, errorStyle.Render(problem))
		}
	}

	for _, problem := range warnings {
		if strings.Contains(strings.ToUpper(line), strings.ToUpper(problem)) {
			line = strings.ReplaceAll(line, problem, warningStyle.Render(problem))
		}
	}

	for _, problem := range info {
		if strings.Contains(strings.ToUpper(line), strings.ToUpper(problem)) {
			line = strings.ReplaceAll(line, problem, infoStyle.Render(problem))
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

	content.WriteString(TitleStyle.Render("🐛 Asterisk Audio Debug"))
	content.WriteString("\n\n")

	// Статус
	status := "🔴 STOPPED"
	mode := ""
	if m.isRunning {
		status = "🟢 RUNNING"
		mode = " | Mode: " + m.debugMode
	}
	content.WriteString(fmt.Sprintf("Status: %s%s | Filter: %s\n\n", status, mode, m.filter))

	// Показываем аудио статистику если есть
	if m.audioStats != "" {
		content.WriteString(m.audioStats)
		content.WriteString("\n")
	}

	if m.debugLogs == "" {
		content.WriteString("No debug data collected yet.\n\n")
		content.WriteString(m.renderDebugInfo())
	} else {
		content.WriteString("=== REAL-TIME DEBUG LOGS ===\n\n")
		content.WriteString(m.debugLogs)
	}

	m.viewport.SetContent(content.String())
}

func (m *DebugModel) renderDebugInfo() string {
	info := `🎯 AUDIO QUALITY DEBUG MONITOR

Common Audio Issues Detected:
• 🔇 "Bubbling" sounds - Jitter buffer problems
• 🎵 Choppy audio - Packet loss or network issues  
• 🔁 One-way audio - NAT/firewall problems
• 📞 Echo - Acoustic or configuration issues
• ⏱️ Delay - Network latency or codec problems

🎛️ Debug Modes:
• BASIC (S): SIP/RTP errors & general issues
• AUDIO (A): Focus on audio quality problems

🔧 Audio-Specific Checks:
• RTP Packet Loss & Jitter
• Jitter Buffer Performance  
• Codec Compatibility
• Network Latency & Stability
• System Resource Usage

⚡ Commands:
• S - Start basic debug
• A - Start audio quality debug
• X - Stop debugging  
• R - Refresh stats
• F - Toggle filters
• C - Clear logs
• Q - Quit

💡 For Audio Issues:
1. Start AUDIO debug (press A)
2. Make a test call with problems
3. Watch for: jitter, packet loss, buffer issues
4. Check network latency and CPU usage`

	return borderStyle.Render(info)
}

func (m *DebugModel) footer() string {
	status := "STOPPED"
	if m.isRunning {
		status = "RUNNING 🟢"
	}

	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render(fmt.Sprintf("Status: %s | S:Basic A:Audio X:Stop R:Refresh F:Filter C:Clear Q:Quit", status))
}
