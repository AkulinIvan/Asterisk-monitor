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

type ChannelsModel struct {
	monitor  MonitorInterface
	viewport viewport.Model
	channels []types.ChannelInfo
	ready    bool
}

func NewChannelsModel(mon MonitorInterface) ChannelsModel {
	vp := viewport.New(80, 20)
	return ChannelsModel{
		monitor:  mon,
		viewport: vp,
		channels: []types.ChannelInfo{},
	}
}

func (m ChannelsModel) Init() tea.Cmd {
    return m.loadChannels
}

func (m ChannelsModel) loadChannels() tea.Msg {
    time.Sleep(100 * time.Millisecond)
    channels := m.monitor.GetActiveChannels()
    return channelsLoadedMsg{channels: channels}
}

type channelsLoadedMsg struct {
    channels []types.ChannelInfo
}

func (m ChannelsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "r", "R":
            return m, m.loadChannels // Возвращаем команду напрямую
        case "q", "Q", "ctrl+c":
            return m, tea.Quit
        }
    case channelsLoadedMsg:
        m.channels = msg.channels
        m.updateContent()
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
    }

    m.viewport, cmd = m.viewport.Update(msg)
    return m, cmd
}

func (m ChannelsModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

type refreshChannelsMsg struct{}

func (m ChannelsModel) refreshChannels() tea.Cmd {
    return func() tea.Msg {
        time.Sleep(100 * time.Millisecond) // Симуляция задержки получения данных
        return refreshChannelsMsg{}
    }
}

func (m *ChannelsModel) updateContent() {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("📞 Active Channels"))
	content.WriteString("\n\n")

	if len(m.channels) == 0 {
		content.WriteString("No active channels\n")
	} else {
		content.WriteString(m.renderChannels())
	}

	m.viewport.SetContent(content.String())
}

func (m *ChannelsModel) renderChannels() string {
	headers := []string{"Channel", "State", "Duration", "Caller ID"}
	var rows [][]string

	for _, channel := range m.channels {
		rows = append(rows, []string{
			TruncateString(channel.Name, 20),
			FormatStatus(channel.State),
			channel.Duration,
			TruncateString(channel.CallerID, 25),
		})
	}

	return FormatTable(headers, rows)
}

func (m *ChannelsModel) footer() string {
	count := len(m.channels)
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render(fmt.Sprintf("Active channels: %d | Press 'r' to refresh | 'q' to quit", count))
}
