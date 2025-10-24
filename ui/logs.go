package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LogsModel struct {
	monitor     MonitorInterface
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
	level.Placeholder = "ALL"
	level.SetValue("ALL")

	filter := textinput.New()
	filter.Placeholder = "Filter text..."

	vp := viewport.New(100, 100)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return LogsModel{
		monitor:     mon,
		viewport:    vp,
		linesInput:  lines,
		levelInput:  level,
		filterInput: filter,
		ready:       true,
	}
}

func (m LogsModel) Init() tea.Cmd {
	m.updateContent()
	return nil
}

func (m LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.loadLogs()
			m.updateContent()
			return m, nil
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
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 6
		}
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

func (m *LogsModel) loadLogs() {
	lines, _ := strconv.Atoi(m.linesInput.Value())
	if lines == 0 {
		lines = 50
	}

	m.logs = m.monitor.GetAsteriskLogs(lines, m.levelInput.Value(), m.filterInput.Value())
}

func (m *LogsModel) updateContent() {
	if !m.ready {
		return
	}

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