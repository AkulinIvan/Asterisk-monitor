package ui

import (
	"asterisk-monitor/types"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ConfigManagerInterface определяет интерфейс для работы с конфигурацией
type ConfigManagerInterface interface {
	Get() *types.Config
	Update(config *types.Config) error
	CreateDefault() error
}

// SettingsModel manages the settings view
type SettingsModel struct {
	config     ConfigManagerInterface
	viewport   viewport.Model
	inputs     []textinput.Model
	focusIndex int
	savedMsg   string
	ready      bool
}

// NewSettingsModel creates a new settings model
func NewSettingsModel(cfg ConfigManagerInterface) SettingsModel {
	currentConfig := cfg.Get()

	// Create input fields
	inputs := make([]textinput.Model, 9)

	// Asterisk Settings
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "localhost"
	inputs[0].SetValue(currentConfig.Asterisk.Host)
	inputs[0].Prompt = "Asterisk Host: "

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "5038"
	inputs[1].SetValue(currentConfig.Asterisk.AMIPort)
	inputs[1].Prompt = "AMI Port: "

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "admin"
	inputs[2].SetValue(currentConfig.Asterisk.Username)
	inputs[2].Prompt = "Username: "

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "password"
	inputs[3].SetValue(currentConfig.Asterisk.Password)
	inputs[3].EchoMode = textinput.EchoPassword
	inputs[3].EchoCharacter = '•'
	inputs[3].Prompt = "Password: "

	// Monitoring Settings
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "30"
	inputs[4].SetValue(fmt.Sprintf("%d", currentConfig.Monitoring.RefreshInterval))
	inputs[4].Prompt = "Refresh Interval (sec): "

	inputs[5] = textinput.New()
	inputs[5].Placeholder = "30"
	inputs[5].SetValue(fmt.Sprintf("%d", currentConfig.Monitoring.LogRetention))
	inputs[5].Prompt = "Log Retention (days): "

	// Security Settings
	inputs[6] = textinput.New()
	inputs[6].Placeholder = "true/false"
	inputs[6].SetValue(fmt.Sprintf("%t", currentConfig.Security.CheckFirewall))
	inputs[6].Prompt = "Check Firewall: "

	inputs[7] = textinput.New()
	inputs[7].Placeholder = "true/false"
	inputs[7].SetValue(fmt.Sprintf("%t", currentConfig.Security.CheckPasswords))
	inputs[7].Prompt = "Check Passwords: "

	inputs[8] = textinput.New()
	inputs[8].Placeholder = "true/false"
	inputs[8].SetValue(fmt.Sprintf("%t", currentConfig.Security.CheckSSL))
	inputs[8].Prompt = "Check SSL: "

	// Set focus on first field
	inputs[0].Focus()

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return SettingsModel{
		config:     cfg,
		viewport:   vp,
		inputs:     inputs,
		focusIndex: 0,
		savedMsg:   "",
		ready:      true, // Сразу готов
	}
}

func (m SettingsModel) Init() tea.Cmd {
	m.updateContent()
	return textinput.Blink
}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab", "shift+tab", "enter", "up", "down":
			// Navigation
			s := msg.String()

			if s == "enter" && m.focusIndex == len(m.inputs) {
				m.saveSettings()
				return m, nil
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i < len(m.inputs); i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(ColorBlue())
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(ColorGray())
			}

			m.updateContent()
			return m, tea.Batch(cmds...)
		case "s", "S":
			m.saveSettings()
			return m, nil
		case "r", "R":
			m.resetToDefaults()
			return m, nil
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

	// Handle text input
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	}

	m.viewport, _ = m.viewport.Update(msg)
	return m, cmd
}

func (m SettingsModel) View() string {
	if !m.ready {
		return "Initializing settings..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

func (m *SettingsModel) saveSettings() {
	newConfig := &types.Config{}

	// Parse Asterisk settings
	newConfig.Asterisk.Host = m.inputs[0].Value()
	newConfig.Asterisk.AMIPort = m.inputs[1].Value()
	newConfig.Asterisk.Username = m.inputs[2].Value()
	newConfig.Asterisk.Password = m.inputs[3].Value()

	// Parse Monitoring settings
	if interval, err := strconv.Atoi(m.inputs[4].Value()); err == nil {
		newConfig.Monitoring.RefreshInterval = interval
	} else {
		newConfig.Monitoring.RefreshInterval = 30
	}

	if retention, err := strconv.Atoi(m.inputs[5].Value()); err == nil {
		newConfig.Monitoring.LogRetention = retention
	} else {
		newConfig.Monitoring.LogRetention = 30
	}

	newConfig.Monitoring.EnableAlerts = true

	// Parse Security settings
	newConfig.Security.CheckFirewall = m.inputs[6].Value() == "true"
	newConfig.Security.CheckPasswords = m.inputs[7].Value() == "true"
	newConfig.Security.CheckSSL = m.inputs[8].Value() == "true"

	// Validate settings
	if newConfig.Asterisk.Host == "" {
		m.savedMsg = errorStyle.Render("Host cannot be empty")
		m.updateContent()
		return
	}
	if newConfig.Asterisk.AMIPort == "" {
		m.savedMsg = errorStyle.Render("AMI Port cannot be empty")
		m.updateContent()
		return
	}
	if newConfig.Monitoring.RefreshInterval < 5 {
		m.savedMsg = errorStyle.Render("Refresh interval must be at least 5 seconds")
		m.updateContent()
		return
	}

	// Save configuration
	if err := m.config.Update(newConfig); err != nil {
		m.savedMsg = errorStyle.Render(fmt.Sprintf("Failed to save settings: %v", err))
	} else {
		m.savedMsg = successStyle.Render("✅ Settings saved successfully!")
	}

	m.updateContent()
}

func (m *SettingsModel) resetToDefaults() {
	if err := m.config.CreateDefault(); err != nil {
		m.savedMsg = errorStyle.Render(fmt.Sprintf("Failed to reset settings: %v", err))
		m.updateContent()
		return
	}

	// Reload inputs with default values
	defaultConfig := m.config.Get()
	m.inputs[0].SetValue(defaultConfig.Asterisk.Host)
	m.inputs[1].SetValue(defaultConfig.Asterisk.AMIPort)
	m.inputs[2].SetValue(defaultConfig.Asterisk.Username)
	m.inputs[3].SetValue(defaultConfig.Asterisk.Password)
	m.inputs[4].SetValue(fmt.Sprintf("%d", defaultConfig.Monitoring.RefreshInterval))
	m.inputs[5].SetValue(fmt.Sprintf("%d", defaultConfig.Monitoring.LogRetention))
	m.inputs[6].SetValue(fmt.Sprintf("%t", defaultConfig.Security.CheckFirewall))
	m.inputs[7].SetValue(fmt.Sprintf("%t", defaultConfig.Security.CheckPasswords))
	m.inputs[8].SetValue(fmt.Sprintf("%t", defaultConfig.Security.CheckSSL))

	m.savedMsg = successStyle.Render("✅ Settings reset to defaults!")
	m.updateContent()
}

func (m *SettingsModel) updateContent() {
	if !m.ready {
		return
	}

	var content strings.Builder

	content.WriteString(TitleStyle.Render("⚙️ Asterisk Monitor Settings"))
	content.WriteString("\n\n")

	if m.savedMsg != "" {
		content.WriteString(m.savedMsg)
		content.WriteString("\n\n")
	}

	// Asterisk Settings
	content.WriteString(BorderStyle().Render("Asterisk Configuration:\n"))
	for i := 0; i < 4; i++ {
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Monitoring Settings
	content.WriteString(BorderStyle().Render("Monitoring Configuration:\n"))
	for i := 4; i < 6; i++ {
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Security Settings
	content.WriteString(BorderStyle().Render("Security Configuration:\n"))
	content.WriteString("Note: Use 'true' or 'false' for security settings\n")
	for i := 6; i < 9; i++ {
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Current configuration info
	content.WriteString(m.renderCurrentConfig())

	m.viewport.SetContent(content.String())
}

func (m *SettingsModel) renderCurrentConfig() string {
	cfg := m.config.Get()

	info := fmt.Sprintf(`Current Configuration:
• Host: %s
• AMI Port: %s
• Username: %s
• Refresh Interval: %d seconds
• Log Retention: %d days
• Security Checks: Firewall=%t, Passwords=%t, SSL=%t`,
		cfg.Asterisk.Host,
		cfg.Asterisk.AMIPort,
		cfg.Asterisk.Username,
		cfg.Monitoring.RefreshInterval,
		cfg.Monitoring.LogRetention,
		cfg.Security.CheckFirewall,
		cfg.Security.CheckPasswords,
		cfg.Security.CheckSSL)

	return BorderStyle().Render(info)
}

func (m *SettingsModel) footer() string {
	var focusName string
	switch m.focusIndex {
	case 0:
		focusName = "Asterisk Host"
	case 1:
		focusName = "AMI Port"
	case 2:
		focusName = "Username"
	case 3:
		focusName = "Password"
	case 4:
		focusName = "Refresh Interval"
	case 5:
		focusName = "Log Retention"
	case 6:
		focusName = "Check Firewall"
	case 7:
		focusName = "Check Passwords"
	case 8:
		focusName = "Check SSL"
	default:
		focusName = "Save Button"
	}

	help := fmt.Sprintf("Focus: %s | Tab: Navigate | Enter: Save | S: Save | R: Reset | Q: Quit", focusName)

	return lipgloss.NewStyle().
		Foreground(ColorGray()).
		Render(help)
}

func (m *SettingsModel) GetContentForTesting() string {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("⚙️ Asterisk Monitor Settings"))
	content.WriteString("\n\n")

	if m.savedMsg != "" {
		content.WriteString(m.savedMsg)
		content.WriteString("\n\n")
	}

	// Asterisk Settings
	content.WriteString(BorderStyle().Render("Asterisk Configuration:\n"))
	for i := 0; i < 4; i++ {
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Monitoring Settings  
	content.WriteString(BorderStyle().Render("Monitoring Configuration:\n"))
	for i := 4; i < 6; i++ {
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Security Settings
	content.WriteString(BorderStyle().Render("Security Configuration:\n"))
	content.WriteString("Note: Use 'true' or 'false' for security settings\n")
	for i := 6; i < 9; i++ {
		content.WriteString(m.inputs[i].View())
		content.WriteString("\n")
	}
	content.WriteString("\n")

	// Current configuration info
	content.WriteString(m.renderCurrentConfig())

	return content.String()
}