package tui

import (
	"strings"
)

func (m Model) renderLogs() string {
	panelWidth := sectionPanelWidth(m.width)
	contentWidth := maxInt(10, panelWidth-6)

	lines := []string{logTitleStyle.Render("Requests")}
	visible := m.visibleLogs()
	if len(visible) == 0 {
		lines = append(lines, emptyStyle.Render("(no requests yet)"))
		for i := 1; i < maxVisibleLogs; i++ {
			lines = append(lines, "")
		}
	} else {
		start := 0
		if len(visible) > maxVisibleLogs {
			start = len(visible) - maxVisibleLogs
		}
		for _, entry := range visible[start:] {
			lines = append(lines, truncateLogLine(entry, contentWidth))
		}
		for len(lines) < maxVisibleLogs+1 {
			lines = append(lines, "")
		}
	}
	return logBoxStyle.Width(panelWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) visibleLogs() []string {
	if len(m.logs) == 0 {
		return nil
	}
	count := minInt(len(m.logs), maxVisibleLogs)
	return m.logs[len(m.logs)-count:]
}
