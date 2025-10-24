package ui

import (
	"strconv"
	"strings"

	"asterisk-monitor/types"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MonitorInterfaceLogs определяет интерфейс для работы с логами
type MonitorInterfaceLogs interface {
    // GetAsteriskLogs возвращает логи Asterisk с фильтрацией
    GetAsteriskLogs(lines int, level, filter string) string
    // ExecuteCommand выполняет команду Asterisk
    ExecuteCommand(name, command string) types.CheckResult
    // GetActiveCallsCount возвращает количество активных звонков
    GetActiveCallsCount() int
    // Добавляем недостающие методы из основного интерфейса
    GetAsteriskStatus() string
    GetSystemMetrics() types.SystemMetrics
}

type LogsModel struct {
	monitor     MonitorInterfaceLogs
	viewport    viewport.Model
	linesInput  textinput.Model
	levelInput  textinput.Model
	filterInput textinput.Model
	logs        string
	ready       bool
}

func NewLogsModel(mon MonitorInterface) LogsModel {
	lines := textinput.New()
	lines.Placeholder = "100"
	lines.SetValue("50")

	level := textinput.New()
	level.Placeholder = "ERROR"
	level.SetValue("ERROR")

	filter := textinput.New()
	filter.Placeholder = "Filter text..."

	vp := viewport.New(80, 20)

	return LogsModel{
		monitor:     mon,
		viewport:    vp,
		linesInput:  lines,
		levelInput:  level,
		filterInput: filter,
	}
}

func (m LogsModel) Init() tea.Cmd {
	return m.loadLogs
}

func (m LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			return m, m.loadLogs
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-6)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
			m.updateContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}
	case logsLoadedMsg:
		m.logs = string(msg)
		m.updateContent()
	}

	m.linesInput, _ = m.linesInput.Update(msg)
	m.levelInput, _ = m.levelInput.Update(msg)
	m.filterInput, _ = m.filterInput.Update(msg)
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m LogsModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var controls strings.Builder
	controls.WriteString("Lines: " + m.linesInput.View() + " | ")
	controls.WriteString("Level: " + m.levelInput.View() + " | ")
	controls.WriteString("Filter: " + m.filterInput.View() + " | ")
	controls.WriteString("Press ENTER to load")

	return controls.String() + "\n" + m.viewport.View() + "\n" + m.footer()
}

type logsLoadedMsg string

func (m LogsModel) loadLogs() tea.Msg {
	lines, _ := strconv.Atoi(m.linesInput.Value())
	if lines == 0 {
		lines = 50
	}

	logs := m.monitor.GetAsteriskLogs(lines, m.levelInput.Value(), m.filterInput.Value())
	return logsLoadedMsg(logs)
}

func (m *LogsModel) updateContent() {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("📋 Asterisk Logs"))
	content.WriteString("\n\n")

	if m.logs == "" {
		content.WriteString("No logs loaded. Configure filters above and press ENTER.\n")
	} else {
		content.WriteString(m.logs)
	}

	m.viewport.SetContent(content.String())
}

func (m *LogsModel) footer() string {
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render("Press ENTER to load logs | 'q' to quit")
}