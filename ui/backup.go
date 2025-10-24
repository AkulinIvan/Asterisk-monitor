package ui

import (
	"asterisk-monitor/types"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BackupModel struct {
	monitor      MonitorInterface
	viewport     viewport.Model
	backupInput  textinput.Model
	restoreInput textinput.Model
	results      []types.CheckResult
	backupsList  string
	ready        bool
}

func NewBackupModel(mon MonitorInterface) BackupModel {
	backup := textinput.New()
	backup.Placeholder = "/backups/asterisk"
	backup.SetValue("/tmp/asterisk-backups")

	restore := textinput.New()
	restore.Placeholder = "/path/to/backup.tar.gz"

	vp := viewport.New(80, 20)
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62"))

	return BackupModel{
		monitor:      mon,
		viewport:     vp,
		backupInput:  backup,
		restoreInput: restore,
		results:      []types.CheckResult{},
		backupsList:  "",
		ready:        true, // Сразу готов
	}
}

func (m BackupModel) Init() tea.Cmd {
	m.listBackups()
	m.updateContent()
	return nil
}

func (m BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b", "B":
			m.createBackup()
			return m, nil
		case "r", "R":
			if m.restoreInput.Value() != "" {
				m.restoreBackup()
			}
			return m, nil
		case "l", "L":
			m.listBackups()
			return m, nil
		case "c", "C":
			m.results = []types.CheckResult{}
			m.backupsList = ""
			m.updateContent()
			return m, nil
		case "q", "Q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.backupInput.Focused() {
				m.backupInput.Blur()
				m.restoreInput.Focus()
			} else {
				m.restoreInput.Blur()
				m.backupInput.Focus()
			}
			return m, nil
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-8)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 8
		}
		m.updateContent()
	}

	m.backupInput, _ = m.backupInput.Update(msg)
	m.restoreInput, _ = m.restoreInput.Update(msg)
	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m BackupModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	var controls strings.Builder
	controls.WriteString("Backup Path: " + m.backupInput.View() + "\n")
	controls.WriteString("Restore File: " + m.restoreInput.View() + "\n\n")
	controls.WriteString("Press TAB to switch fields | ")
	controls.WriteString("B: Backup | R: Restore | L: List | C: Clear | Q: Quit")

	return controls.String() + "\n" + m.viewport.View()
}

func (m *BackupModel) createBackup() {
	backupPath := m.backupInput.Value()
	if backupPath == "" {
		backupPath = "/tmp/asterisk-backups"
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	backupFile := fmt.Sprintf("%s/asterisk-backup-%s.tar.gz", backupPath, timestamp)
	backupDir := fmt.Sprintf("/tmp/asterisk-backup-%s", timestamp)

	// Добавляем сообщение о начале бэкапа
	m.results = append(m.results, types.CheckResult{
		Name:      "Backup Started",
		Status:    "info",
		Message:   fmt.Sprintf("Creating backup to: %s", backupFile),
		Timestamp: time.Now(),
	})
	m.updateContent()

	commands := []string{
		fmt.Sprintf("mkdir -p %s", backupPath),
		fmt.Sprintf("mkdir -p %s", backupDir),
		"cp -r /etc/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /etc/asterisk access'",
		"cp -r /var/lib/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /var/lib/asterisk access'",
		"cp -r /var/spool/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /var/spool/asterisk access'",
		"cp -r /var/log/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /var/log/asterisk access'",
		fmt.Sprintf("tar -czf %s -C %s . 2>/dev/null || echo 'Tar failed'", backupFile, backupDir),
		fmt.Sprintf("rm -rf %s", backupDir),
		fmt.Sprintf("chmod 644 %s 2>/dev/null || true", backupFile),
	}

	// Выполняем команды синхронно
	for i, cmd := range commands {
		result := m.monitor.ExecuteCommand(fmt.Sprintf("Backup Step %d", i+1), cmd)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(500 * time.Millisecond) // Задержка для визуального эффекта

		// Если ошибка, прерываем
		if result.Status == "error" {
			// Cleanup on error
			m.monitor.ExecuteCommand("Cleanup", fmt.Sprintf("rm -rf %s %s", backupDir, backupFile))
			return
		}
	}

	// Verify backup
	verifyCmd := fmt.Sprintf("test -f %s && tar -tzf %s | wc -l || echo '0'", backupFile, backupFile)
	verifyResult := m.monitor.ExecuteCommand("Verify Backup", verifyCmd)
	
	if verifyResult.Status == "success" && verifyResult.Message != "0" {
		fileCount := strings.TrimSpace(verifyResult.Message)
		m.results = append(m.results, types.CheckResult{
			Name:      "Backup Completed",
			Status:    "success",
			Message:   fmt.Sprintf("Backup created successfully: %s (%s files)", backupFile, fileCount),
			Timestamp: time.Now(),
		})
	} else {
		m.results = append(m.results, types.CheckResult{
			Name:      "Backup Completed",
			Status:    "warning",
			Message:   fmt.Sprintf("Backup created but verification failed: %s", backupFile),
			Timestamp: time.Now(),
		})
	}

	m.updateContent()
	m.listBackups() // Обновляем список бэкапов
}

func (m *BackupModel) restoreBackup() {
	backupFile := m.restoreInput.Value()
	if backupFile == "" {
		m.results = append(m.results, types.CheckResult{
			Name:      "Restore Error",
			Status:    "error",
			Message:   "No backup file specified",
			Timestamp: time.Now(),
		})
		m.updateContent()
		return
	}

	// Добавляем сообщение о начале восстановления
	m.results = append(m.results, types.CheckResult{
		Name:      "Restore Started",
		Status:    "info",
		Message:   fmt.Sprintf("Starting restore from: %s", backupFile),
		Timestamp: time.Now(),
	})
	m.updateContent()

	// Check if backup file exists
	checkCmd := fmt.Sprintf("test -f %s && echo 'exists' || echo 'not found'", backupFile)
	checkResult := m.monitor.ExecuteCommand("Check Backup", checkCmd)
	if !strings.Contains(checkResult.Message, "exists") {
		m.results = append(m.results, types.CheckResult{
			Name:      "Restore Error",
			Status:    "error",
			Message:   fmt.Sprintf("Backup file not found: %s", backupFile),
			Timestamp: time.Now(),
		})
		m.updateContent()
		return
	}

	// Create restore directory
	restoreDir := fmt.Sprintf("/tmp/asterisk-restore-%d", time.Now().Unix())

	commands := []string{
		fmt.Sprintf("mkdir -p %s", restoreDir),
		fmt.Sprintf("tar -xzf %s -C %s", backupFile, restoreDir),

		// Stop Asterisk before restore
		"systemctl stop asterisk",

		// Backup current configuration
		fmt.Sprintf("cp -r /etc/asterisk /etc/asterisk.backup.%d", time.Now().Unix()),

		// Restore files
		fmt.Sprintf("cp -r %s/etc/asterisk/* /etc/asterisk/ 2>/dev/null || echo 'No config files to restore'", restoreDir),
		fmt.Sprintf("cp -r %s/var/lib/asterisk/* /var/lib/asterisk/ 2>/dev/null || echo 'No lib files to restore'", restoreDir),
		fmt.Sprintf("cp -r %s/var/spool/asterisk/* /var/spool/asterisk/ 2>/dev/null || echo 'No spool files to restore'", restoreDir),

		// Fix permissions
		"chown -R asterisk:asterisk /etc/asterisk/ 2>/dev/null || true",
		"chown -R asterisk:asterisk /var/lib/asterisk/ 2>/dev/null || true",
		"chown -R asterisk:asterisk /var/spool/asterisk/ 2>/dev/null || true",

		// Start Asterisk
		"systemctl start asterisk",

		// Cleanup
		fmt.Sprintf("rm -rf %s", restoreDir),
	}

	// Выполняем команды синхронно
	for i, cmd := range commands {
		result := m.monitor.ExecuteCommand(fmt.Sprintf("Restore Step %d", i+1), cmd)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(500 * time.Millisecond)

		// Если ошибка, пытаемся восстановить
		if result.Status == "error" {
			// Emergency restore
			m.monitor.ExecuteCommand("Emergency Restore",
				fmt.Sprintf("cp -r /etc/asterisk.backup.%d/* /etc/asterisk/ && systemctl start asterisk", time.Now().Unix()))
			return
		}
	}

	m.results = append(m.results, types.CheckResult{
		Name:      "Restore Completed",
		Status:    "success",
		Message:   fmt.Sprintf("Backup restored successfully from: %s", backupFile),
		Timestamp: time.Now(),
	})
	m.updateContent()
}

func (m *BackupModel) listBackups() {
	backupPath := m.backupInput.Value()
	if backupPath == "" {
		backupPath = "/tmp/asterisk-backups"
	}

	cmd := fmt.Sprintf("ls -la %s/asterisk-backup-*.tar.gz 2>/dev/null | head -20 || echo 'No backups found'", backupPath)
	result := m.monitor.ExecuteCommand("List Backups", cmd)

	if result.Status == "success" && !strings.Contains(result.Message, "No backups found") {
		// Get backup sizes
		sizeCmd := fmt.Sprintf("du -h %s/asterisk-backup-*.tar.gz 2>/dev/null | sort -hr || echo 'No size info'", backupPath)
		sizeResult := m.monitor.ExecuteCommand("Backup Sizes", sizeCmd)

		m.backupsList = "Available Backups:\n" + result.Message
		if sizeResult.Status == "success" {
			m.backupsList += "\n\nSizes:\n" + sizeResult.Message
		}
	} else {
		m.backupsList = "No backups found in " + backupPath
	}

	m.updateContent()
}

func (m *BackupModel) updateContent() {
	if !m.ready {
		return
	}

	var content strings.Builder

	content.WriteString(TitleStyle.Render("💾 Asterisk Backup & Restore"))
	content.WriteString("\n\n")

	if len(m.results) == 0 && m.backupsList == "" {
		content.WriteString("No backup operations performed yet.\n\n")
		content.WriteString(m.renderBackupInfo())
	} else {
		if m.backupsList != "" {
			content.WriteString(m.backupsList)
			content.WriteString("\n\n")
		}

		if len(m.results) > 0 {
			content.WriteString(m.renderResults())
		}
	}

	m.viewport.SetContent(content.String())
}

func (m *BackupModel) renderResults() string {
	var builder strings.Builder

	builder.WriteString("Recent Operations:\n")
	for _, result := range m.results {
		var statusIcon string
		switch result.Status {
		case "success":
			statusIcon = "✅"
		case "error":
			statusIcon = "❌"
		case "warning":
			statusIcon = "⚠️"
		case "info":
			statusIcon = "ℹ️"
		default:
			statusIcon = "🔍"
		}

		timestamp := FormatTimestamp(result.Timestamp)
		builder.WriteString(fmt.Sprintf("%s [%s] %s: %s\n",
			statusIcon, timestamp, result.Name, result.Message))
	}

	return borderStyle.Render(builder.String())
}

func (m *BackupModel) renderBackupInfo() string {
	info := `Backup Includes:
• /etc/asterisk/ - Configuration files
• /var/lib/asterisk/ - Library files
• /var/spool/asterisk/ - Spool directories
• /var/log/asterisk/ - Log files

Backup Commands:
• Press 'B' to create backup
• Press 'R' to restore from specified file
• Press 'L' to list available backups
• Press 'C' to clear results

Important Notes:
• Backups are compressed .tar.gz files
• Restore stops Asterisk temporarily
• Original configs are backed up automatically
• Verify backups regularly`

	return borderStyle.Render(info)
}

// Helper function to get backup statistics
func (m *BackupModel) getBackupStats() string {
	backupPath := m.backupInput.Value()
	if backupPath == "" {
		backupPath = "/tmp/asterisk-backups"
	}

	statsCmd := fmt.Sprintf(`
        count=$(find %s -name "asterisk-backup-*.tar.gz" -type f 2>/dev/null | wc -l)
        total_size=$(du -ch %s/asterisk-backup-*.tar.gz 2>/dev/null | grep total | cut -f1)
        latest=$(ls -t %s/asterisk-backup-*.tar.gz 2>/dev/null | head -1)
        if [ -n "$latest" ]; then
            latest_date=$(stat -c %%y "$latest" 2>/dev/null | cut -d' ' -f1)
        else
            latest_date="N/A"
        fi
        echo "Backups: $count | Total Size: $total_size | Latest: $latest_date"
    `, backupPath, backupPath, backupPath)

	result := m.monitor.ExecuteCommand("Backup Stats", statsCmd)
	return result.Message
}