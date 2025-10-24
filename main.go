package main

import (
	"asterisk-monitor/config"
	monitor "asterisk-monitor/monitors"
	"asterisk-monitor/ui"
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type appModel struct {
	currentView string
	dashboard   ui.DashboardModel
	diagnostics ui.DiagnosticsModel
	channels    ui.ChannelsModel
	logs        ui.LogsModel
	security    ui.SecurityModel
	backup      ui.BackupModel
	debug       ui.DebugModel
	settings    ui.SettingsModel
	monitor     *monitor.LinuxMonitor
}

func initialAppModel(configManager *config.ConfigManager) appModel {
	mon := monitor.NewLinuxMonitor()

	return appModel{
		currentView: "dashboard",
		dashboard:   ui.NewDashboardModel(mon),
		diagnostics: ui.NewDiagnosticsModel(mon),
		channels:    ui.NewChannelsModel(mon),
		logs:        ui.NewLogsModel(mon),
		security:    ui.NewSecurityModel(mon),
		backup:      ui.NewBackupModel(mon),
		debug:       ui.NewDebugModel(mon),
		settings:    ui.NewSettingsModel(configManager),
		monitor:     mon,
	}
}

func (m appModel) Init() tea.Cmd {
	return m.dashboard.Init()
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "1":
			m.currentView = "dashboard"
			cmd = m.dashboard.Init()
		case "2":
			m.currentView = "diagnostics"
			cmd = m.diagnostics.Init()
		case "3":
			m.currentView = "channels"
			cmd = m.channels.Init()
		case "4":
			m.currentView = "logs"
			cmd = m.logs.Init()
		case "5":
			m.currentView = "security"
			cmd = m.security.Init()
		case "6":
			m.currentView = "backup"
			cmd = m.backup.Init()
		case "7":
			m.currentView = "settings"
			cmd = m.settings.Init()
		case "8":
			m.currentView = "debug"
			cmd = m.debug.Init()
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		}
	}

	switch m.currentView {
	case "dashboard":
		newModel, newCmd := m.dashboard.Update(msg)
		m.dashboard = newModel.(ui.DashboardModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "diagnostics":
		newModel, newCmd := m.diagnostics.Update(msg)
		m.diagnostics = newModel.(ui.DiagnosticsModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "channels":
		newModel, newCmd := m.channels.Update(msg)
		m.channels = newModel.(ui.ChannelsModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "logs":
		newModel, newCmd := m.logs.Update(msg)
		m.logs = newModel.(ui.LogsModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "security":
		newModel, newCmd := m.security.Update(msg)
		m.security = newModel.(ui.SecurityModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "backup":
		newModel, newCmd := m.backup.Update(msg)
		m.backup = newModel.(ui.BackupModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "debug":
		newModel, newCmd := m.debug.Update(msg)
		m.debug = newModel.(ui.DebugModel)
		if newCmd != nil {
			cmd = newCmd
		}
	case "settings":
		newModel, newCmd := m.settings.Update(msg)
		m.settings = newModel.(ui.SettingsModel)
		if newCmd != nil {
			cmd = newCmd
		}
	}

	return m, cmd
}

func (m appModel) View() string {
	var view string

	switch m.currentView {
	case "dashboard":
		view = m.dashboard.View()
	case "diagnostics":
		view = m.diagnostics.View()
	case "channels":
		view = m.channels.View()
	case "logs":
		view = m.logs.View()
	case "security":
		view = m.security.View()
	case "backup":
		view = m.backup.View()
	case "debug":
		view = m.debug.View()
	case "settings":
		view = m.settings.View()
	default:
		view = m.dashboard.View()
	}

	return m.renderHeader() + "\n" + view
}

func (m appModel) renderHeader() string {
	views := []string{
		"1: Dashboard",
		"2: Diagnostics",
		"3: Channels",
		"4: Logs",
		"5: Security",
		"6: Backup",
		"7: Settings",
		"8: Debug",
	}

	var currentViewName string
	switch m.currentView {
	case "dashboard":
		currentViewName = "📊 Dashboard"
	case "diagnostics":
		currentViewName = "🔍 Diagnostics"
	case "channels":
		currentViewName = "📞 Channels"
	case "logs":
		currentViewName = "📋 Logs"
	case "security":
		currentViewName = "🛡️ Security"
	case "backup":
		currentViewName = "💾 Backup"
	case "settings":
		currentViewName = "⚙️ Settings"
	case "debug":
		currentViewName = "🐛 Debug"
	}

	header := fmt.Sprintf("Asterisk Monitor - %s", currentViewName)
	navigation := strings.Join(views, " | ")

	return ui.TitleStyle.Render(header) + "\n" +
		ui.InfoStyle.Render(navigation) + "\n" +
		strings.Repeat("─", 80)
}

func main() {
	// Проверяем, установлен ли Asterisk
	if !isAsteriskInstalled() {
		fmt.Println("❌ Asterisk не установлен или не найден в PATH")
		fmt.Println("Установите Asterisk для Linux:")
		fmt.Println("  Ubuntu/Debian: sudo apt install asterisk")
		fmt.Println("  CentOS/RHEL:   sudo yum install asterisk")
		fmt.Println("  Arch:          sudo pacman -S asterisk")
		os.Exit(1)
	}

	// Проверяем права доступа
	if !hasAsteriskAccess() {
		fmt.Println("⚠️  Предупреждение: возможны проблемы с доступом к Asterisk")
		fmt.Println("Рекомендуется запускать с правами asterisk пользователя или root")
		fmt.Print("Продолжить? (y/n): ")

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			os.Exit(0)
		}
	}

	// Загружаем конфигурацию
	configManager := config.NewConfigManager()
	if err := configManager.Load(); err != nil {
		fmt.Printf("⚠️  Не удалось загрузить конфигурацию: %v\n", err)
		fmt.Println("Будет использована конфигурация по умолчанию")
	}

	fmt.Println("🚀 Запуск Asterisk Monitor...")
	fmt.Println("   Переключение между модулями: 1-5")
	fmt.Println("   Для выхода нажмите Ctrl+C или Q")

	model := initialAppModel(configManager)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Ошибка запуска приложения: %v\n", err)
		os.Exit(1)
	}
}

func isAsteriskInstalled() bool {
	_, err := exec.LookPath("asterisk")
	return err == nil
}

func hasAsteriskAccess() bool {
	cmd := exec.Command("asterisk", "-rx", "core show version")
	err := cmd.Run()
	return err == nil
}
