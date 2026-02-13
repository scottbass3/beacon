package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) renderApp() string {
	sections := []string{
		m.renderTopSection(),
		m.renderMainSection(),
	}
	if m.debug {
		sections = append(sections, m.renderLogs())
	}
	return strings.Join(sections, "\n")
}

func (m Model) renderTopSection() string {
	contextName := strings.TrimSpace(m.context)
	if contextName == "" {
		contextName = "-"
	}
	statusValue := strings.TrimSpace(m.status)
	if statusValue == "" {
		statusValue = "-"
	}
	statusLine := statusStyle.Render(statusValue)
	if m.isLoading() {
		statusLine = statusLoadingStyle.Render("Loading")
		if statusValue != "-" {
			statusLine = statusLoadingStyle.Render("Loading " + statusValue)
		}
	}
	pathValue := strings.TrimSpace(m.currentPath())
	if pathValue == "" {
		pathValue = "/"
	}
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("Beacon"), statusLine)
	metaLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		metaLabelStyle.Render("Context"),
		metaValueStyle.Render(contextName),
		metaLabelStyle.Render("Path"),
		metaValueStyle.Render(pathValue),
	)
	lines := []string{
		headerLine,
		metaLine,
	}
	if inputLine := m.renderModeInputLine(); inputLine != "" {
		lines = append(lines, modeInputStyle.Render(inputLine))
	}
	lines = append(lines, shortcutHintStyle.Render(m.renderShortcutHintLine()))
	return topSectionStyle.Width(sectionPanelWidth(m.width)).Render(strings.Join(lines, "\n"))
}

func (m Model) renderMainSection() string {
	panelWidth := sectionPanelWidth(m.width)
	contentWidth := m.mainSectionContentWidth()
	titleLabel := focusLabel(m.focus)
	body := m.renderBody()
	if m.helpActive {
		titleLabel = "Help"
		body = m.renderHelpSectionBody()
	}
	title := mainSectionTitleStyle.Render(strings.ToUpper(titleLabel))
	titleLine := mainSectionTitleLine.
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(title)
	content := strings.Join([]string{
		titleLine,
		body,
	}, "\n")
	return mainSectionStyle.Width(panelWidth).Render(content)
}

func sectionPanelWidth(width int) int {
	if width <= 0 {
		width = defaultRenderWidth
	}
	panelWidth := width - 2
	if panelWidth < 24 {
		panelWidth = width
	}
	if panelWidth < 1 {
		panelWidth = 1
	}
	return panelWidth
}

func (m Model) mainSectionContentWidth() int {
	contentWidth := sectionPanelWidth(m.width) - mainSectionHChromeChars
	if contentWidth < 1 {
		return 1
	}
	return contentWidth
}

func (m Model) renderModeInputLine() string {
	if m.commandActive {
		return m.commandInput.View()
	}
	if m.filterActive {
		return m.filterInput.View()
	}
	if value := strings.TrimSpace(m.filterInput.Value()); value != "" {
		return m.filterInput.Prompt + value
	}
	if !m.dockerHubActive {
		if !m.githubActive {
			return ""
		}
		if m.githubInputFocus {
			return m.githubInput.View()
		}
		if value := strings.TrimSpace(m.githubInput.Value()); value != "" {
			return "Search: " + value
		}
		return ""
	}
	if m.dockerHubInputFocus {
		return m.dockerHubInput.View()
	}
	if value := strings.TrimSpace(m.dockerHubInput.Value()); value != "" {
		return "Search: " + value
	}
	return ""
}

func (m Model) renderShortcutHintLine() string {
	return m.shortcutHintLine()
}

func (m Model) renderBody() string {
	view := m.table.View()
	if len(m.table.Rows()) == 0 {
		return view + "\n" + emptyStyle.Render(m.emptyBodyMessage())
	}
	return view
}

func (m Model) currentPath() string {
	if m.dockerHubActive {
		if m.dockerHubImage != "" {
			return "dockerhub/" + m.dockerHubImage
		}
		return "dockerhub"
	}
	if m.githubActive {
		if m.githubImage != "" {
			return "ghcr/" + m.githubImage
		}
		return "ghcr"
	}
	if path := m.breadcrumb(); path != "" {
		return path
	}
	return "/"
}
