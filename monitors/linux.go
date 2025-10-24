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
    if lines == 0 {
        lines = 50
    }
    
    // Используем ротацию логов чтобы обновить файлы
    rotateCmd := "asterisk -rx 'logger rotate' 2>/dev/null"
    m.ExecuteCommand("Rotate Logs", rotateCmd)
    time.Sleep(100 * time.Millisecond)
    
    // Читаем логи из файла
    cmd := fmt.Sprintf("tail -%d /var/log/asterisk/messages 2>/dev/null", lines)
    result := m.ExecuteCommand("Logs", cmd)
    
    // Если доступ запрещен, пробуем с sudo
    if result.Status == "error" || strings.Contains(result.Message, "Permission denied") {
        cmd = fmt.Sprintf("sudo tail -%d /var/log/asterisk/messages 2>/dev/null", lines)
        result = m.ExecuteCommand("Logs Sudo", cmd)
    }
    
    // Если все еще ошибка, показываем информацию о доступных логах
    if result.Status == "error" {
        return m.getLogsDebugInfo()
    }
    
    rawLogs := result.Message
    
    // Если логи пустые, возможно файл не существует
    if strings.TrimSpace(rawLogs) == "" {
        return "Log file is empty or does not exist. Check Asterisk logging configuration."
    }
    
    // Применяем фильтры
    output := rawLogs
    
    if level != "ALL" && level != "" {
        output = m.filterLogsByLevel(output, level)
    }
    
    if filter != "" {
        output = m.filterLogs(output, filter)
    }
    
    // Если после фильтрации ничего не осталось, показываем отладочную информацию
    if strings.TrimSpace(output) == "" {
        return m.getFilterDebugInfo(rawLogs, level, filter)
    }
    
    return output
}

func (m *LinuxMonitor) getLogsDebugInfo() string {
    var debug strings.Builder
    debug.WriteString("=== Logs Debug Information ===\n\n")
    
    // Проверяем доступ к файлам логов
    checks := []struct {
        name string
        cmd  string
    }{
        {"Messages File", "ls -la /var/log/asterisk/messages 2>&1 || echo 'Not found'"},
        {"Full File", "ls -la /var/log/asterisk/full 2>&1 || echo 'Not found'"},
        {"Asterisk Status", "asterisk -rx 'core show version' 2>&1"},
        {"Logger Status", "asterisk -rx 'logger show channels' 2>&1"},
    }
    
    for _, check := range checks {
        result := m.ExecuteCommand(check.name, check.cmd)
        debug.WriteString(fmt.Sprintf("● %s:\n%s\n\n", check.name, result.Message))
    }
    
    debug.WriteString("Try these commands manually:\n")
    debug.WriteString("  tail -50 /var/log/asterisk/messages\n")
    debug.WriteString("  sudo tail -50 /var/log/asterisk/messages\n")
    debug.WriteString("  asterisk -rx 'logger show channels'\n")
    
    return debug.String()
}

func (m *LinuxMonitor) getFilterDebugInfo(rawLogs, level, filter string) string {
    var debug strings.Builder
    debug.WriteString("=== Filter Debug Information ===\n\n")
    
    debug.WriteString(fmt.Sprintf("Level filter: %s\n", level))
    debug.WriteString(fmt.Sprintf("Text filter: %s\n\n", filter))
    
    // Показываем первые несколько строк сырых логов для анализа
    lines := strings.Split(rawLogs, "\n")
    debug.WriteString("First 10 lines of raw logs:\n")
    for i := 0; i < len(lines) && i < 10; i++ {
        if lines[i] != "" {
            debug.WriteString(fmt.Sprintf("%d: %s\n", i+1, lines[i]))
        }
    }
    
    debug.WriteString("\nCommon log patterns to try:\n")
    debug.WriteString("• Level: ALL (show all logs)\n")
    debug.WriteString("• Level: NOTICE (informational messages)\n") 
    debug.WriteString("• Level: WARNING (warning messages)\n")
    debug.WriteString("• Filter: sip (SIP related messages)\n")
    debug.WriteString("• Filter: chan (channel messages)\n")
    
    return debug.String()
}

func (m *LinuxMonitor) getSystemInfoInstead(lines int) string {
    if lines == 0 {
        lines = 20
    }
    
    var output strings.Builder
    output.WriteString("=== System Information (Logs unavailable) ===\n\n")
    
    // Получаем системную информацию через доступные команды
    commands := []struct {
        name string
        cmd  string
    }{
        {"Uptime", "asterisk -rx 'core show uptime'"},
        {"Version", "asterisk -rx 'core show version'"},
        {"Channels", "asterisk -rx 'core show channels count'"},
        {"Calls", "asterisk -rx 'core show calls'"},
        {"SIP Peers", "asterisk -rx 'sip show peers' | head -10"},
        {"System Load", "uptime"},
    }
    
    for _, item := range commands {
        result := m.ExecuteCommand(item.name, item.cmd)
        if result.Status == "success" {
            output.WriteString(fmt.Sprintf("● %s: %s\n", item.name, strings.TrimSpace(result.Message)))
        }
    }
    
    output.WriteString("\nTo access logs, run:\n")
    output.WriteString("  sudo tail -f /var/log/asterisk/messages\n")
    output.WriteString("Or check permissions on /var/log/asterisk/ directory\n")
    
    return output.String()
}

func (m *LinuxMonitor) filterLogsByLevel(logs, level string) string {
    levelLower := strings.ToLower(level)
    lines := strings.Split(logs, "\n")
    var filtered []string
    
    for _, line := range lines {
        lineLower := strings.ToLower(line)
        
        switch levelLower {
        case "error":
            // Разные варианты обозначения ошибок в Asterisk
            if strings.Contains(lineLower, "error") || 
               strings.Contains(lineLower, "err[") ||
               strings.Contains(lineLower, ".err]") ||
               strings.Contains(lineLower, "failed") ||
               strings.Contains(lineLower, "failure") ||
               strings.Contains(lineLower, "reject") ||
               strings.Contains(lineLower, "unable") ||
               strings.Contains(lineLower, "invalid") {
                filtered = append(filtered, line)
            }
        case "warning", "warn":
            if strings.Contains(lineLower, "warning") || 
               strings.Contains(lineLower, "warn[") ||
               strings.Contains(lineLower, ".wrn]") ||
               strings.Contains(lineLower, "deprecated") ||
               strings.Contains(lineLower, "not found") {
                filtered = append(filtered, line)
            }
        case "notice":
            if strings.Contains(lineLower, "notice") || 
               strings.Contains(lineLower, "ntc[") ||
               strings.Contains(lineLower, ".ntc]") ||
               strings.Contains(lineLower, "registered") ||
               strings.Contains(lineLower, "unregistered") ||
               strings.Contains(lineLower, "connected") ||
               strings.Contains(lineLower, "destroyed") {
                filtered = append(filtered, line)
            }
        case "debug":
            if strings.Contains(lineLower, "debug") || 
               strings.Contains(lineLower, "dbg[") ||
               strings.Contains(lineLower, ".dbg]") {
                filtered = append(filtered, line)
            }
        case "verbose":
            if strings.Contains(lineLower, "verbose") || 
               strings.Contains(lineLower, "verb[") ||
               strings.Contains(lineLower, ".verb]") {
                filtered = append(filtered, line)
            }
        case "all", "":
            // Без фильтра - все строки
            filtered = append(filtered, line)
        default:
            // Для неизвестных уровней ищем точное совпадение
            if strings.Contains(lineLower, levelLower) {
                filtered = append(filtered, line)
            }
        }
    }
    
    return strings.Join(filtered, "\n")
}

// filterLogs фильтрует логи по тексту
func (m *LinuxMonitor) filterLogs(logs, filter string) string {
    lines := strings.Split(logs, "\n")
    var filtered []string
    filterLower := strings.ToLower(filter)
    
    for _, line := range lines {
        if strings.Contains(strings.ToLower(line), filterLower) {
            filtered = append(filtered, line)
        }
    }
    
    return strings.Join(filtered, "\n")
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