package ui

import (
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss"

	"asterisk-monitor/types"
)

type MonitorInterface interface {
    GetAsteriskStatus() string
    GetAsteriskPID() string
    GetServiceStatus() string
    GetSIPPeersCount() (int, int)
    GetActiveCallsCount() int
    GetActiveChannels() []types.ChannelInfo
    GetAsteriskUptime() string
    GetSystemLoad() string
    GetCPUUsage() float64
    GetMemoryUsage() float64
    GetDiskUsage() float64
    ExecuteCommand(name, command string) types.CheckResult
    GetAsteriskLogs(lines int, level, filter string) string
    GetSystemMetrics() types.SystemMetrics
}

var (
    // Colors
    colorGreen    = lipgloss.Color("10")
    colorRed      = lipgloss.Color("9")
    colorYellow   = lipgloss.Color("11")
    colorBlue     = lipgloss.Color("12")
    colorPurple   = lipgloss.Color("13")
    colorGray     = lipgloss.Color("8")
    colorDarkGray = lipgloss.Color("235")

    // Styles
    TitleStyle = lipgloss.NewStyle().
            Foreground(colorBlue).
            Bold(true).
            Padding(0, 1)

    InfoStyle = lipgloss.NewStyle().
            Foreground(colorGray)

    statusStyle = lipgloss.NewStyle().
            Bold(true).
            Padding(0, 1)

    successStyle = statusStyle.Copy().Foreground(colorGreen)
    warningStyle = statusStyle.Copy().Foreground(colorYellow)
    errorStyle   = statusStyle.Copy().Foreground(colorRed)
    infoStyle    = statusStyle.Copy().Foreground(colorBlue)

    borderStyle = lipgloss.NewStyle().
            BorderStyle(lipgloss.RoundedBorder()).
            BorderForeground(colorGray).
            Padding(0, 1)

    metricStyle = lipgloss.NewStyle().
            Foreground(colorPurple).
            Bold(true)

    labelStyle = lipgloss.NewStyle().
            Foreground(colorGray)

    logStyle = lipgloss.NewStyle().
            Foreground(colorDarkGray)
)

// Export style getters
func SuccessStyle() lipgloss.Style {
    return successStyle
}

func WarningStyle() lipgloss.Style {
    return warningStyle
}

func ErrorStyle() lipgloss.Style {
    return errorStyle
}




func FormatStatus(status string) string {
	switch status {
	case "running", "active", "success":
		return successStyle.Render("● " + status)
	case "stopped", "inactive", "failed", "error":
		return errorStyle.Render("● " + status)
	case "warning":
		return warningStyle.Render("● " + status)
	default:
		return infoStyle.Render("● " + status)
	}
}

func FormatMetric(label, value string) string {
	return labelStyle.Render(label) + ": " + metricStyle.Render(value)
}

func FormatLogEntry(timestamp, message string) string {
	return logStyle.Render(timestamp) + " " + message
}

func ProgressBar(width int, percent float64) string {
    // Валидация входных параметров
    if width < 0 {
        width = 0
    }
    if width == 0 {
        return ""
    }
    
    // Ограничиваем процент в диапазоне 0-100
    if percent < 0 {
        percent = 0
    }
    if percent > 100 {
        percent = 100
    }
    
    // Рассчитываем количество заполненных блоков
    filled := int(float64(width) * percent / 100)
    if filled > width {
        filled = width
    }
    empty := width - filled

    // Выбираем цвет в зависимости от процента
    var color lipgloss.Color
    switch {
    case percent >= 80:
        color = colorRed
    case percent >= 60:
        color = colorYellow
    default:
        color = colorGreen
    }

    barStyle := lipgloss.NewStyle().Foreground(color)

    // Создаем строку прогресс-бара
    filledPart := strings.Repeat("█", filled)
    emptyPart := strings.Repeat("░", empty)
    
    return barStyle.Render(filledPart) + emptyPart
}

func ProgressBarVisualLength(width int, percent float64) int {
    if width <= 0 {
        return 0
    }
    
    if percent < 0 {
        percent = 0
    }
    if percent > 100 {
        percent = 100
    }
    
    filled := int(float64(width) * percent / 100)
    if filled > width {
        filled = width
    }
    empty := width - filled
    
    // Каждый символ занимает 1 визуальную позицию, независимо от байтов
    return filled + empty
}

func FormatTable(headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return "No data available"
	}

	// Calculate column widths
	colWidths := make([]int, len(headers))
	for i, header := range headers {
		colWidths[i] = len(header)
	}

	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	// Build table
	var builder strings.Builder

	// Headers
	for i, header := range headers {
		builder.WriteString(lipgloss.NewStyle().
			Foreground(colorBlue).
			Bold(true).
			Width(colWidths[i]).
			Render(header))
		if i < len(headers)-1 {
			builder.WriteString(" │ ")
		}
	}
	builder.WriteString("\n")

	// Separator
	for i, width := range colWidths {
		builder.WriteString(strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			builder.WriteString("─┼─")
		}
	}
	builder.WriteString("\n")

	// Rows
	for _, row := range rows {
		for i, cell := range row {
			builder.WriteString(lipgloss.NewStyle().
				Width(colWidths[i]).
				Render(cell))
			if i < len(row)-1 {
				builder.WriteString(" │ ")
			}
		}
		builder.WriteString("\n")
	}

	return borderStyle.Render(builder.String())
}

func FormatTimestamp(t time.Time) string {
	return t.Format("15:04:05")
}

func TruncateString(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    
    // Для Unicode строк корректно обрезаем
    runes := []rune(s)
    if len(runes) <= maxLen {
        return s
    }
    
    return string(runes[:maxLen-3]) + "..."
}

// Экспортируемые функции для границ
func BorderStyle() lipgloss.Style {
	return borderStyle
}

// Экспортируемые цветовые константы
func ColorBlue() lipgloss.Color {
	return colorBlue
}

func ColorGray() lipgloss.Color {
	return colorGray
}

func ColorRed() lipgloss.Color {
	return colorRed
}

func ColorYellow() lipgloss.Color {
	return colorYellow
}

func ColorGreen() lipgloss.Color {
	return colorGreen
}
