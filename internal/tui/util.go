package tui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"github.com/scottbass3/beacon/internal/registry"
)

func dockerHubErrorMsg(err error) dockerHubTagsMsg {
	msg := dockerHubTagsMsg{err: err}
	var rateErr *registry.DockerHubRateLimitError
	if errors.As(err, &rateErr) {
		msg.rateLimit = rateErr.RateLimit
		msg.retryAfter = rateErr.RetryAfter
	}
	return msg
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) authUI() registry.AuthUI {
	if m.provider == nil {
		return registry.AuthUI{}
	}
	return m.provider.AuthUI(m.auth)
}

func (m Model) authFieldCount() int {
	ui := m.authUI()
	if ui.ShowRemember {
		return 3
	}
	if ui.ShowUsername || ui.ShowPassword {
		return 2
	}
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func lineCount(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func truncateLogLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	line := strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if len(line) <= width {
		return line
	}
	if width <= 3 {
		return line[:width]
	}
	return line[:width-3] + "..."
}

func equalTableColumns(a, b []table.Column) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Width != b[i].Width {
			return false
		}
	}
	return true
}

func equalTableRows(a, b []table.Row) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
