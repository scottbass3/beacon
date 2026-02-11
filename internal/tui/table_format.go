package tui

import (
	"fmt"
	"strings"
	"time"
)

func formatCount(value int) string {
	if value < 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04")
}

func formatSize(sizeBytes int64) string {
	if sizeBytes < 0 {
		return "-"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(sizeBytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d B", sizeBytes)
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func formatHistoryCommand(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func firstNonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
