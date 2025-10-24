package ui

import (
	"asterisk-monitor/types"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardModel struct {
	monitor    MonitorInterface
	viewport   viewport.Model
	metrics    types.SystemMetrics
	lastUpdate time.Time
	alerts     []string
	ready      bool
}

func NewDashboardModel(mon MonitorInterface) DashboardModel {
	vp := viewport.New(80, 20)
	return DashboardModel{
		monitor:    mon,
		viewport:   vp,
		metrics:    mon.GetSystemMetrics(),
		lastUpdate: time.Now(),
		alerts:     []string{},
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			m.refreshData()
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
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 2
		}
		m.updateContent()
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m DashboardModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

func (m *DashboardModel) refreshData() {
	m.metrics = m.monitor.GetSystemMetrics()
	m.lastUpdate = time.Now()
	m.updateContent()

	// Check for alerts
	if m.metrics.ActiveCalls > 10 {
		m.addAlert(fmt.Sprintf("High call volume: %d active calls", m.metrics.ActiveCalls))
	}
	if m.metrics.CPUUsage > 80 {
		m.addAlert(fmt.Sprintf("High CPU usage: %.1f%%", m.metrics.CPUUsage))
	}
}

func (m *DashboardModel) updateContent() {
	var content strings.Builder

	// Header
	content.WriteString(TitleStyle.Render("📊 Asterisk Monitor Dashboard"))
	content.WriteString("\n\n")

	// System Status
	content.WriteString(m.renderSystemStatus())
	content.WriteString("\n\n")

	// Metrics
	content.WriteString(m.renderMetrics())
	content.WriteString("\n\n")

	// SIP Peers
	content.WriteString(m.renderSIPPeers())
	content.WriteString("\n\n")

	// Recent Alerts
	if len(m.alerts) > 0 {
		content.WriteString(m.renderAlerts())
		content.WriteString("\n\n")
	}

	m.viewport.SetContent(content.String())
}

func (m *DashboardModel) renderSystemStatus() string {
	status := m.monitor.GetAsteriskStatus()
	serviceStatus := m.metrics.ServiceState

	return borderStyle.Render(
		"System Status:\n" +
			FormatStatus("Asterisk Process: "+status) + "\n" +
			FormatStatus("Systemd Service: "+serviceStatus) + "\n" +
			FormatMetric("PID", m.metrics.AsteriskPID) + "\n" +
			FormatMetric("Uptime", m.metrics.Uptime) + "\n" +
			FormatMetric("Load Average", m.metrics.LoadAverage),
	)
}

func (m *DashboardModel) renderMetrics() string {
	onlineStr := strconv.Itoa(m.metrics.OnlinePeers)
	totalStr := strconv.Itoa(m.metrics.TotalPeers)
	callsStr := strconv.Itoa(m.metrics.ActiveCalls)

	peersStatus := fmt.Sprintf("%s/%s", onlineStr, totalStr)
	if m.metrics.OnlinePeers == m.metrics.TotalPeers && m.metrics.TotalPeers > 0 {
		peersStatus = successStyle.Render(peersStatus)
	} else if m.metrics.OnlinePeers > 0 {
		peersStatus = warningStyle.Render(peersStatus)
	} else {
		peersStatus = errorStyle.Render(peersStatus)
	}

	return borderStyle.Render(
		"Performance Metrics:\n" +
			FormatMetric("CPU Usage", fmt.Sprintf("%.1f%%", m.metrics.CPUUsage)) + " " +
			ProgressBar(20, m.metrics.CPUUsage) + "\n" +
			FormatMetric("Memory Usage", fmt.Sprintf("%.1f%%", m.metrics.MemoryUsage)) + " " +
			ProgressBar(20, m.metrics.MemoryUsage) + "\n" +
			FormatMetric("Disk Usage", fmt.Sprintf("%.1f%%", m.metrics.DiskUsage)) + " " +
			ProgressBar(20, m.metrics.DiskUsage) + "\n" +
			FormatMetric("Active Calls", callsStr) + "\n" +
			FormatMetric("SIP Peers", peersStatus),
	)
}

func (m *DashboardModel) renderSIPPeers() string {
	online, total := m.monitor.GetSIPPeersCount()
	status := "Healthy"
	style := successStyle

	if online == 0 && total > 0 {
		status = "Critical"
		style = errorStyle
	} else if online < total {
		status = "Warning"
		style = warningStyle
	}

	return borderStyle.Render(
		"SIP Status:\n" +
			FormatMetric("Online/Total", fmt.Sprintf("%d/%d", online, total)) + "\n" +
			FormatMetric("Status", style.Render(status)),
	)
}

func (m *DashboardModel) renderAlerts() string {
	var alertsStr strings.Builder
	alertsStr.WriteString("Recent Alerts:\n")

	for i, alert := range m.alerts {
		if i >= 5 { // Show only last 5 alerts
			break
		}
		alertsStr.WriteString("⚠️  " + alert + "\n")
	}

	return borderStyle.Render(alertsStr.String())
}

func (m *DashboardModel) addAlert(alert string) {
	timestamp := FormatTimestamp(time.Now())
	m.alerts = append([]string{timestamp + " - " + alert}, m.alerts...)
	if len(m.alerts) > 10 {
		m.alerts = m.alerts[:10]
	}
}

func (m *DashboardModel) footer() string {
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render(fmt.Sprintf("Last update: %s | Press 'r' to refresh | 'q' to quit",
			FormatTimestamp(m.lastUpdate)))
}
