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

	return BackupModel{
		monitor:      mon,
		viewport:     vp,
		backupInput:  backup,
		restoreInput: restore,
		results:      []types.CheckResult{},
		backupsList:  "",
	}
}

func (m BackupModel) Init() tea.Cmd {
	return m.listBackups
}

func (m BackupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "b", "B":
			return m, m.createBackup
		case "r", "R":
			if m.restoreInput.Value() != "" {
				return m, m.restoreBackup
			}
		case "l", "L":
			return m, m.listBackups
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
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
		}
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-8)
			m.viewport.Style = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))
			m.ready = true
			m.updateContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 8
		}
	case backupCreatedMsg:
		m.results = append(m.results, types.CheckResult{
			Name:      "Backup Created",
			Status:    "success",
			Message:   string(msg),
			Timestamp: time.Now(),
		})
		m.updateContent()
		return m, m.listBackups
	case backupRestoredMsg:
		m.results = append(m.results, types.CheckResult{
			Name:      "Backup Restored",
			Status:    "success",
			Message:   string(msg),
			Timestamp: time.Now(),
		})
		m.updateContent()
	case backupsListedMsg:
		m.backupsList = string(msg)
		m.updateContent()
	case backupErrorMsg:
		m.results = append(m.results, types.CheckResult{
			Name:      "Backup Error",
			Status:    "error",
			Message:   string(msg),
			Timestamp: time.Now(),
		})
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

// Messages
type backupCreatedMsg string
type backupRestoredMsg string
type backupsListedMsg string
type backupErrorMsg string

// Commands
func (m BackupModel) createBackup() tea.Msg {
	backupPath := m.backupInput.Value()
	if backupPath == "" {
		backupPath = "/tmp/asterisk-backups"
	}

	timestamp := time.Now().Format("2006-01-02-150405")
	backupFile := fmt.Sprintf("%s/asterisk-backup-%s.tar.gz", backupPath, timestamp)
	backupDir := fmt.Sprintf("/tmp/asterisk-backup-%s", timestamp)

	commands := []string{
		fmt.Sprintf("sudo mkdir -p %s", backupPath),
		fmt.Sprintf("sudo mkdir -p %s", backupDir),
		"sudo cp -r /etc/asterisk/ " + backupDir + "/",
		"sudo cp -r /var/lib/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /var/lib/asterisk found'",
		"sudo cp -r /var/spool/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /var/spool/asterisk found'",
		"sudo cp -r /var/log/asterisk/ " + backupDir + "/ 2>/dev/null || echo 'No /var/log/asterisk found'",
		fmt.Sprintf("sudo tar -czf %s -C %s .", backupFile, backupDir),
		fmt.Sprintf("sudo rm -rf %s", backupDir),
		fmt.Sprintf("sudo chmod 644 %s", backupFile),
	}

	var results []string
	for i, cmd := range commands {
		result := m.monitor.ExecuteCommand(fmt.Sprintf("Backup Step %d", i+1), cmd)
		if result.Status == "error" {
			// Cleanup on error
			m.monitor.ExecuteCommand("Cleanup", fmt.Sprintf("sudo rm -rf %s %s", backupDir, backupFile))
			return backupErrorMsg(fmt.Sprintf("Backup failed at step %d: %s", i+1, result.Error))
		}
		results = append(results, result.Message)
	}

	// Verify backup
	verifyCmd := fmt.Sprintf("sudo tar -tzf %s | wc -l", backupFile)
	verifyResult := m.monitor.ExecuteCommand("Verify Backup", verifyCmd)
	if verifyResult.Status == "success" {
		fileCount := strings.TrimSpace(verifyResult.Message)
		return backupCreatedMsg(fmt.Sprintf("Backup created successfully: %s (%s files)", backupFile, fileCount))
	}

	return backupCreatedMsg(fmt.Sprintf("Backup created: %s", backupFile))
}

func (m BackupModel) restoreBackup() tea.Msg {
	backupFile := m.restoreInput.Value()
	if backupFile == "" {
		return backupErrorMsg("No backup file specified")
	}

	// Check if backup file exists
	checkCmd := fmt.Sprintf("sudo test -f %s && echo 'exists' || echo 'not found'", backupFile)
	checkResult := m.monitor.ExecuteCommand("Check Backup", checkCmd)
	if !strings.Contains(checkResult.Message, "exists") {
		return backupErrorMsg(fmt.Sprintf("Backup file not found: %s", backupFile))
	}

	// Create restore directory
	restoreDir := fmt.Sprintf("/tmp/asterisk-restore-%d", time.Now().Unix())

	commands := []string{
		fmt.Sprintf("sudo mkdir -p %s", restoreDir),
		fmt.Sprintf("sudo tar -xzf %s -C %s", backupFile, restoreDir),

		// Stop Asterisk before restore
		"sudo systemctl stop asterisk",

		// Backup current configuration
		fmt.Sprintf("sudo cp -r /etc/asterisk /etc/asterisk.backup.%d", time.Now().Unix()),

		// Restore files
		fmt.Sprintf("sudo cp -r %s/etc/asterisk/* /etc/asterisk/", restoreDir),
		fmt.Sprintf("sudo cp -r %s/var/lib/asterisk/* /var/lib/asterisk/ 2>/dev/null || echo 'No lib files to restore'", restoreDir),
		fmt.Sprintf("sudo cp -r %s/var/spool/asterisk/* /var/spool/asterisk/ 2>/dev/null || echo 'No spool files to restore'", restoreDir),

		// Fix permissions
		"sudo chown -R asterisk:asterisk /etc/asterisk/",
		"sudo chown -R asterisk:asterisk /var/lib/asterisk/ 2>/dev/null || true",
		"sudo chown -R asterisk:asterisk /var/spool/asterisk/ 2>/dev/null || true",

		// Start Asterisk
		"sudo systemctl start asterisk",

		// Cleanup
		fmt.Sprintf("sudo rm -rf %s", restoreDir),
	}

	for i, cmd := range commands {
		result := m.monitor.ExecuteCommand(fmt.Sprintf("Restore Step %d", i+1), cmd)
		if result.Status == "error" {
			// Restore original backup and start Asterisk
			m.monitor.ExecuteCommand("Emergency Restore",
				fmt.Sprintf("sudo cp -r /etc/asterisk.backup.%d/* /etc/asterisk/ && sudo systemctl start asterisk", time.Now().Unix()))
			return backupErrorMsg(fmt.Sprintf("Restore failed at step %d: %s", i+1, result.Error))
		}
	}

	return backupRestoredMsg(fmt.Sprintf("Backup restored successfully from: %s", backupFile))
}

func (m BackupModel) listBackups() tea.Msg {
	backupPath := m.backupInput.Value()
	if backupPath == "" {
		backupPath = "/tmp/asterisk-backups"
	}

	cmd := fmt.Sprintf("sudo ls -la %s/asterisk-backup-*.tar.gz 2>/dev/null | head -20", backupPath)
	result := m.monitor.ExecuteCommand("List Backups", cmd)

	if result.Status == "error" || strings.Contains(result.Message, "No such file") {
		return backupsListedMsg("No backups found in " + backupPath)
	}

	// Get backup sizes
	sizeCmd := fmt.Sprintf("sudo du -h %s/asterisk-backup-*.tar.gz 2>/dev/null | sort -hr", backupPath)
	sizeResult := m.monitor.ExecuteCommand("Backup Sizes", sizeCmd)

	list := "Available Backups:\n" + result.Message
	if sizeResult.Status == "success" {
		list += "\n\nSizes:\n" + sizeResult.Message
	}

	return backupsListedMsg(list)
}

func (m *BackupModel) updateContent() {
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
        count=$(sudo find %s -name "asterisk-backup-*.tar.gz" -type f 2>/dev/null | wc -l)
        total_size=$(sudo du -ch %s/asterisk-backup-*.tar.gz 2>/dev/null | grep total | cut -f1)
        latest=$(sudo ls -t %s/asterisk-backup-*.tar.gz 2>/dev/null | head -1)
        if [ -n "$latest" ]; then
            latest_date=$(sudo stat -c %%y "$latest" 2>/dev/null | cut -d' ' -f1)
        else
            latest_date="N/A"
        fi
        echo "Backups: $count | Total Size: $total_size | Latest: $latest_date"
    `, backupPath, backupPath, backupPath)

	result := m.monitor.ExecuteCommand("Backup Stats", statsCmd)
	return result.Message
}
