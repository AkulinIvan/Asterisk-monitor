package monitor

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LogProblemCall записывает информацию о проблемном вызове в лог
func (m *LinuxMonitor) LogProblemCall(severity, channel, problem, details string) error {
	logDir := "/var/log/asterisk-monitor"
	logFile := filepath.Join(logDir, "problem-calls.log")
	
	// Создаем директорию если не существует
	os.MkdirAll(logDir, 0755)
	
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logEntry := fmt.Sprintf("[%s] [%s] Channel: %s | Problem: %s | Details: %s\n", 
		timestamp, severity, channel, problem, details)
	
	_, err = file.WriteString(logEntry)
	return err
}

// GetProblemCallLogs возвращает последние записи из лога проблемных вызовов
func (m *LinuxMonitor) GetProblemCallLogs(lines int) string {
	if lines == 0 {
		lines = 50
	}
	
	logFile := "/var/log/asterisk-monitor/problem-calls.log"
	cmd := fmt.Sprintf("tail -%d %s 2>/dev/null || echo 'No problem call logs found'", lines, logFile)
	
	result := m.ExecuteCommand("Problem Logs", cmd)
	return result.Message
}

// ClearProblemCallLogs очищает лог проблемных вызовов
func (m *LinuxMonitor) ClearProblemCallLogs() string {
	logFile := "/var/log/asterisk-monitor/problem-calls.log"
	cmd := fmt.Sprintf("echo '' > %s", logFile)
	
	result := m.ExecuteCommand("Clear Logs", cmd)
	if result.Status == "success" {
		return "Problem call logs cleared successfully"
	}
	return "Failed to clear problem call logs"
}