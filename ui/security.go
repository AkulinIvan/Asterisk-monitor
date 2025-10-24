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

type SecurityModel struct {
	monitor  MonitorInterface
	viewport viewport.Model
	results  []types.CheckResult
	ready    bool
}

func NewSecurityModel(mon MonitorInterface) SecurityModel {
	vp := viewport.New(80, 20)
	return SecurityModel{
		monitor:  mon,
		viewport: vp,
		results:  []types.CheckResult{},
	}
}

func (m SecurityModel) Init() tea.Cmd {
	return nil
}

func (m SecurityModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "r", "R":
			m.runQuickSecurityScan()
		case "f", "F":
			m.runFullSecurityScan()
		case "c", "C":
			m.results = []types.CheckResult{}
			m.updateContent()
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

func (m SecurityModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	return m.viewport.View() + "\n" + m.footer()
}

func (m *SecurityModel) runQuickSecurityScan() {
	m.results = []types.CheckResult{}
	m.updateContent()

	checks := []struct {
		name string
		cmd  string
	}{
		{"Open SIP Ports", "netstat -tlnp | grep -E ':(5060|5061|5062)' | grep LISTEN"},
		{"Open AMI Port", "netstat -tlnp | grep ':5038' | grep LISTEN"},
		{"Fail2Ban Status", "systemctl is-active fail2ban"},
		{"Firewall Status", "systemctl is-active ufw 2>/dev/null || systemctl is-active firewalld 2>/dev/null || echo 'No firewall detected'"},
		{"Asterisk Process User", "ps aux | grep asterisk | grep -v grep | awk '{print $1}' | head -1"},
	}

	for _, check := range checks {
		result := m.monitor.ExecuteCommand(check.name, check.cmd)
		m.analyzeSecurityResult(&result)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(300 * time.Millisecond)
	}
}

func (m *SecurityModel) runFullSecurityScan() {
	m.results = []types.CheckResult{}
	m.updateContent()

	checks := []struct {
		name string
		cmd  string
	}{
		// Network Security
		{"Open Network Ports", "netstat -tlnp | grep -E ':(5060|5061|5062|5038|10000)'"},
		{"SIP Port Exposure", "ss -tlnp | grep -E ':(5060|5061)' | awk '{print $4}'"},
		{"AMI Port Exposure", "ss -tlnp | grep ':5038' | awk '{print $4}'"},

		// Service Security
		{"Fail2Ban Status", "systemctl is-active fail2ban"},
		{"Firewall Status", "systemctl is-active ufw 2>/dev/null || systemctl is-active firewalld 2>/dev/null || echo 'No firewall detected'"},
		{"SELinux Status", "getenforce 2>/dev/null || echo 'SELinux not available'"},

		// File Permissions
		{"Asterisk Config Permissions", "find /etc/asterisk -type f -perm /o+rw -ls | wc -l"},
		{"Asterisk Directory Permissions", "find /etc/asterisk -type d -perm /o+rwx -ls | wc -l"},
		{"Asterisk File Ownership", "find /etc/asterisk ! -user asterisk -type f | wc -l"},

		// Process Security
		{"Asterisk Process User", "ps aux | grep asterisk | grep -v grep | awk '{print $1}' | head -1"},
		{"Asterisk Running as Root", "ps aux | grep asterisk | grep -v grep | grep root | wc -l"},

		// SSL/TLS Security
		{"SSL Certificate Check", "find /etc/asterisk -name '*.pem' -exec openssl x509 -checkend 86400 -in {} \\; 2>/dev/null | grep -c 'will expire' || echo 'No SSL certificates found'"},
		{"TLS Configuration", "grep -r 'tls' /etc/asterisk/*.conf 2>/dev/null | grep -v '^#' | wc -l"},

		// Authentication Security
		{"Default Passwords Check", "grep -r 'password' /etc/asterisk/sip.conf 2>/dev/null | grep -v '^#' | grep -v '^;' | head -5"},
		{"AMI Authentication", "grep -r 'secret\\|password' /etc/asterisk/manager.conf 2>/dev/null | grep -v '^#' | grep -v '^;' | head -3"},

		// Logging Security
		{"Log File Permissions", "ls -la /var/log/asterisk/ 2>/dev/null | head -5"},
		{"Debug Mode Check", "grep -r 'debug' /etc/asterisk/logger.conf 2>/dev/null | grep -v '^#' | grep -v 'off' | wc -l"},
	}

	for _, check := range checks {
		result := m.monitor.ExecuteCommand(check.name, check.cmd)
		m.analyzeSecurityResult(&result)
		m.results = append(m.results, result)
		m.updateContent()
		time.Sleep(200 * time.Millisecond)
	}
}

func (m *SecurityModel) analyzeSecurityResult(result *types.CheckResult) {
	// Анализируем результат и устанавливаем соответствующий статус
	switch result.Name {
	case "Open SIP Ports":
		if strings.Contains(result.Message, "0.0.0.0") || strings.Contains(result.Message, ":::") {
			result.Status = "warning"
			result.Message += " ⚠️  SIP ports exposed to all interfaces"
		} else if result.Message == "" {
			result.Status = "success"
			result.Message = "No SIP ports open to public"
		}

	case "Open AMI Port":
		if strings.Contains(result.Message, "0.0.0.0") || strings.Contains(result.Message, ":::") {
			result.Status = "error"
			result.Message += " ❌ AMI port exposed to all interfaces - SECURITY RISK!"
		} else if result.Message == "" {
			result.Status = "success"
			result.Message = "AMI port not exposed to public"
		}

	case "Fail2Ban Status":
		if result.Message != "active" {
			result.Status = "warning"
			result.Message += " ⚠️  Fail2Ban not active"
		}

	case "Firewall Status":
		if strings.Contains(result.Message, "inactive") || strings.Contains(result.Message, "No firewall") {
			result.Status = "warning"
			result.Message += " ⚠️  Firewall not active"
		}

	case "Asterisk Config Permissions":
		if result.Message != "0" {
			result.Status = "error"
			result.Message += " ❌ World-writable config files found"
		}

	case "Asterisk Process User":
		if result.Message == "root" {
			result.Status = "warning"
			result.Message += " ⚠️  Running as root - not recommended"
		}

	case "Asterisk Running as Root":
		if result.Message != "0" {
			result.Status = "error"
			result.Message += " ❌ Asterisk should not run as root"
		}

	case "SSL Certificate Check":
		if result.Message != "No SSL certificates found" && result.Message != "0" {
			result.Status = "warning"
			result.Message += " ⚠️  SSL certificates expiring soon"
		}

	case "Default Passwords Check":
		if result.Message != "" {
			result.Status = "warning"
			result.Message += " ⚠️  Check for default passwords"
		}
	}

	// Если статус еще не установлен, устанавливаем по умолчанию
	if result.Status == "success" && result.Error == "" {
		result.Status = "success"
	} else if result.Error != "" {
		result.Status = "error"
	}
}

func (m *SecurityModel) updateContent() {
	var content strings.Builder

	content.WriteString(TitleStyle.Render("🛡️ Asterisk Security Scan"))
	content.WriteString("\n\n")

	if len(m.results) == 0 {
		content.WriteString("No security scan performed yet.\n")
		content.WriteString("Press 'r' for quick scan or 'f' for full security audit.\n\n")
		content.WriteString(m.renderSecurityTips())
	} else {
		content.WriteString(m.renderSecurityResults())
		content.WriteString("\n\n")
		content.WriteString(m.renderSecuritySummary())
	}

	m.viewport.SetContent(content.String())
}

func (m *SecurityModel) renderSecurityResults() string {
	var builder strings.Builder

	for _, result := range m.results {
		var statusIcon string
		switch result.Status {
		case "success":
			statusIcon = "✅"
		case "warning":
			statusIcon = "⚠️"
		case "error":
			statusIcon = "❌"
		default:
			statusIcon = "🔍"
		}

		builder.WriteString(fmt.Sprintf("%s %s\n", statusIcon, result.Name))
		builder.WriteString(fmt.Sprintf("   %s\n", result.Message))
		if result.Error != "" {
			builder.WriteString(fmt.Sprintf("   Error: %s\n", result.Error))
		}
		builder.WriteString("\n")
	}

	return borderStyle.Render(builder.String())
}

func (m *SecurityModel) renderSecuritySummary() string {
	criticalCount := 0
	warningCount := 0
	successCount := 0

	for _, result := range m.results {
		switch result.Status {
		case "error":
			criticalCount++
		case "warning":
			warningCount++
		case "success":
			successCount++
		}
	}

	var summary strings.Builder
	summary.WriteString("Security Summary:\n")

	if criticalCount > 0 {
		summary.WriteString(errorStyle.Render(fmt.Sprintf("❌ Critical Issues: %d - Immediate attention required!\n", criticalCount)))
	}
	if warningCount > 0 {
		summary.WriteString(warningStyle.Render(fmt.Sprintf("⚠️  Warnings: %d - Review recommended\n", warningCount)))
	}
	if successCount > 0 {
		summary.WriteString(successStyle.Render(fmt.Sprintf("✅ Passed Checks: %d\n", successCount)))
	}

	// Общая оценка безопасности
	totalChecks := criticalCount + warningCount + successCount
	if totalChecks > 0 {
		score := (successCount * 100) / totalChecks
		summary.WriteString(fmt.Sprintf("\nSecurity Score: %d%%\n", score))

		if score >= 80 {
			summary.WriteString(successStyle.Render("Overall: Good security posture ✓"))
		} else if score >= 60 {
			summary.WriteString(warningStyle.Render("Overall: Needs improvement ⚠️"))
		} else {
			summary.WriteString(errorStyle.Render("Overall: Poor security posture ❌"))
		}
	}

	return borderStyle.Render(summary.String())
}

func (m *SecurityModel) renderSecurityTips() string {
	tips := []string{
		"🔒 Always run Asterisk as non-root user",
		"🚫 Restrict AMI (5038) to localhost only",
		"🛡️ Enable fail2ban for SIP authentication",
		"🔑 Use strong passwords for SIP accounts",
		"📝 Regularly update SSL certificates",
		"🌐 Configure firewall to restrict SIP ports",
		"📊 Monitor logs for suspicious activity",
		"🔄 Keep Asterisk and system updated",
		"🔍 Regular security scans recommended",
		"📚 Review Asterisk security best practices",
	}

	var builder strings.Builder
	builder.WriteString("Security Best Practices:\n")
	for _, tip := range tips {
		builder.WriteString("• " + tip + "\n")
	}

	return borderStyle.Render(builder.String())
}

func (m *SecurityModel) footer() string {
	return lipgloss.NewStyle().
		Foreground(colorGray).
		Render("Press 'r' for quick scan, 'f' for full audit, 'c' to clear, 'q' to quit")
}
