package monitor

import (
    "asterisk-monitor/types"
    "fmt"
    "os/exec"
    "strconv"
    "strings"
    "time"
)

type LinuxMonitor struct{}

func NewLinuxMonitor() *LinuxMonitor {
    return &LinuxMonitor{}
}

// GetAsteriskStatus возвращает статус Asterisk
func (m *LinuxMonitor) GetAsteriskStatus() string {
    cmd := exec.Command("sh", "-c", "ps aux | grep -v grep | grep asterisk")
    output, err := cmd.Output()
    
    if err == nil && strings.Contains(string(output), "asterisk") {
        return "running"
    }
    return "stopped"
}

// GetAsteriskPID возвращает PID процесса Asterisk
func (m *LinuxMonitor) GetAsteriskPID() string {
    // Получаем основной PID Asterisk (не safe_asterisk)
    cmd := exec.Command("sh", "-c", "ps aux | grep asterisk | grep -v grep | grep -v safe_asterisk | awk '{print $2}' | head -1")
    output, err := cmd.Output()
    if err != nil {
        return "N/A"
    }
    pid := strings.TrimSpace(string(output))
    if pid == "" {
        return "N/A"
    }
    return pid
}

// GetServiceStatus возвращает статус systemd сервиса
func (m *LinuxMonitor) GetServiceStatus() string {
    cmd := exec.Command("sh", "-c", "systemctl is-active asterisk 2>/dev/null || echo 'unknown'")
    output, err := cmd.Output()
    
    if err != nil {
        return "unknown"
    }
    return strings.TrimSpace(string(output))
}

func (m *LinuxMonitor) GetSIPPeersDetail() string {
    cmd := exec.Command("asterisk", "-rx", "sip show peers")
    output, err := cmd.Output()
    
    if err != nil {
        return "Error getting SIP peers details"
    }
    
    // Возвращаем последние 10 строк для диагностики
    lines := strings.Split(string(output), "\n")
    if len(lines) > 10 {
        return strings.Join(lines[len(lines)-10:], "\n")
    }
    return string(output)
}

// GetSIPPeersCount возвращает количество онлайн и общее число SIP пиров
func (m *LinuxMonitor) GetSIPPeersCount() (int, int) {
    cmd := exec.Command("asterisk", "-rx", "sip show peers")
    output, err := cmd.Output()
    
    if err != nil {
        return 0, 0
    }
    
    outputStr := string(output)
    online := 0
    total := 0
    
    // Разбиваем вывод на строки
    lines := strings.Split(outputStr, "\n")
    
    for _, line := range lines {
        trimmed := strings.TrimSpace(line)
        
        // Пропускаем заголовки и пустые строки
        if strings.Contains(trimmed, "Name/username") || trimmed == "" {
            continue
        }
        
        // Ищем строку с итоговой статистикой
        if strings.Contains(trimmed, "sip peers") {
            // Парсим итоговую строку: "25 sip peers [Monitored: 7 online, 13 offline Unmonitored: 5 online, 0 offline]"
            if strings.Contains(trimmed, "Monitored:") && strings.Contains(trimmed, "Unmonitored:") {
                // Извлекаем числа из строки
                monitoredOnline := extractNumberAfter(trimmed, "Monitored:", "online")
                unmonitoredOnline := extractNumberAfter(trimmed, "Unmonitored:", "online")
                online = monitoredOnline + unmonitoredOnline
                
                // Общее количество - первое число в строке
                total = extractFirstNumber(trimmed)
            }
            break
        }
        
        // Если не нашли итоговую строку, считаем вручную
        if strings.Contains(trimmed, "/") && len(strings.Fields(trimmed)) >= 6 {
            total++
            fields := strings.Fields(trimmed)
            statusField := fields[len(fields)-2] // Предпоследнее поле - статус
            if statusField == "OK" || statusField == "Unmonitored" {
                online++
            }
        }
    }
    
    return online, total
}

func extractNumberAfter(text, after, before string) int {
    startIdx := strings.Index(text, after)
    if startIdx == -1 {
        return 0
    }
    
    endIdx := strings.Index(text[startIdx:], before)
    if endIdx == -1 {
        return 0
    }
    
    numberStr := strings.TrimSpace(text[startIdx+len(after) : startIdx+endIdx])
    number, err := strconv.Atoi(numberStr)
    if err != nil {
        return 0
    }
    
    return number
}

func extractFirstNumber(text string) int {
    fields := strings.Fields(text)
    if len(fields) > 0 {
        number, err := strconv.Atoi(fields[0])
        if err == nil {
            return number
        }
    }
    return 0
}

// GetActiveCallsCount возвращает количество активных вызовов
func (m *LinuxMonitor) GetActiveCallsCount() int {
    cmd := exec.Command("asterisk", "-rx", "core show channels")
    output, err := cmd.Output()
    
    if err != nil {
        return 0
    }
    
    lines := strings.Split(string(output), "\n")
    for _, line := range lines {
        if strings.Contains(line, "active channel") {
            parts := strings.Fields(line)
            if len(parts) > 0 {
                count, _ := strconv.Atoi(parts[0])
                return count
            }
        }
    }
    
    return 0
}

// GetActiveChannels возвращает список активных каналов
func (m *LinuxMonitor) GetActiveChannels() []types.ChannelInfo {
    cmd := exec.Command("asterisk", "-rx", "core show channels")
    output, err := cmd.Output()
    
    if err != nil {
        return []types.ChannelInfo{}
    }
    
    var channels []types.ChannelInfo
    lines := strings.Split(string(output), "\n")
    
    for _, line := range lines {
        if strings.Contains(line, "/") && !strings.Contains(line, "active channel") {
            parts := strings.Fields(line)
            if len(parts) >= 4 {
                channel := types.ChannelInfo{
                    Name:        parts[0],
                    State:       parts[1],
                    Duration:    parts[2],
                    CallerID:    strings.Join(parts[3:], " "),
                    Application: "unknown",
                }
                channels = append(channels, channel)
            }
        }
    }
    
    return channels
}

// GetAsteriskUptime возвращает время работы Asterisk
func (m *LinuxMonitor) GetAsteriskUptime() string {
    cmd := exec.Command("asterisk", "-rx", "core show uptime")
    output, err := cmd.Output()
    
    if err != nil {
        return "unknown"
    }
    
    lines := strings.Split(string(output), "\n")
    for _, line := range lines {
        if strings.Contains(line, "System uptime") {
            return strings.TrimSpace(strings.TrimPrefix(line, "System uptime:"))
        }
    }
    
    return "unknown"
}

// GetSystemLoad возвращает нагрузку системы
func (m *LinuxMonitor) GetSystemLoad() string {
    cmd := exec.Command("sh", "-c", "uptime | awk -F'load average:' '{print $2}'")
    output, err := cmd.Output()
    
    if err != nil {
        return "unknown"
    }
    
    return strings.TrimSpace(string(output))
}

// GetCPUUsage возвращает использование CPU
func (m *LinuxMonitor) GetCPUUsage() float64 {
    cmd := exec.Command("sh", "-c", "top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1")
    output, err := cmd.Output()
    
    if err != nil {
        return 0
    }
    
    usageStr := strings.TrimSpace(string(output))
    usage, err := strconv.ParseFloat(usageStr, 64)
    if err != nil {
        return 0
    }
    
    return usage
}

// GetMemoryUsage возвращает использование памяти
func (m *LinuxMonitor) GetMemoryUsage() float64 {
    cmd := exec.Command("sh", "-c", "free | grep Mem | awk '{printf \"%.1f\", $3/$2 * 100.0}'")
    output, err := cmd.Output()
    
    if err != nil {
        return 0
    }
    
    usageStr := strings.TrimSpace(string(output))
    usage, err := strconv.ParseFloat(usageStr, 64)
    if err != nil {
        return 0
    }
    
    return usage
}

// GetDiskUsage возвращает использование диска
func (m *LinuxMonitor) GetDiskUsage() float64 {
    cmd := exec.Command("sh", "-c", "df / | awk 'NR==2 {print $5}' | sed 's/%//'")
    output, err := cmd.Output()
    
    if err != nil {
        return 0
    }
    
    usageStr := strings.TrimSpace(string(output))
    usage, err := strconv.ParseFloat(usageStr, 64)
    if err != nil {
        return 0
    }
    
    return usage
}

// ExecuteCommand выполняет команду Asterisk
func (m *LinuxMonitor) ExecuteCommand(name, command string) types.CheckResult {
    cmd := exec.Command("sh", "-c", command)
    output, err := cmd.Output()
    
    if err != nil {
        return types.CheckResult{
            Name:      name,
            Status:    "error",
            Message:   fmt.Sprintf("Command failed: %s", command),
            Error:     err.Error(),
            Timestamp: time.Now(),
        }
    }
    
    return types.CheckResult{
        Name:      name,
        Status:    "success",
        Message:   strings.TrimSpace(string(output)),
        Timestamp: time.Now(),
    }
}

// GetAsteriskLogs возвращает логи Asterisk
func (m *LinuxMonitor) GetAsteriskLogs(lines int, level, filter string) string {
    cmd := fmt.Sprintf("sudo tail -%d /var/log/asterisk/messages", lines)
    if level != "ALL" {
        cmd += fmt.Sprintf(" | grep -i %s", level)
    }
    if filter != "" {
        cmd += fmt.Sprintf(" | grep -i \"%s\"", filter)
    }
    
    result := m.ExecuteCommand("Logs", cmd)
    return result.Message
}

// GetSystemMetrics возвращает полные системные метрики
func (m *LinuxMonitor) GetSystemMetrics() types.SystemMetrics {
    return types.SystemMetrics{
        CPUUsage:     m.GetCPUUsage(),
        MemoryUsage:  m.GetMemoryUsage(),
        DiskUsage:    m.GetDiskUsage(),
        ActiveCalls:  m.GetActiveCallsCount(),
        OnlinePeers:  func() int { o, _ := m.GetSIPPeersCount(); return o }(),
        TotalPeers:   func() int { _, t := m.GetSIPPeersCount(); return t }(),
        Uptime:       m.GetAsteriskUptime(),
        LoadAverage:  m.GetSystemLoad(),
        AsteriskPID:  m.GetAsteriskPID(),
        ServiceState: m.GetServiceStatus(),
    }
}