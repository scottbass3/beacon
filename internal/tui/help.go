package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) renderHelpSectionBody() string {
	pageTitle := m.helpPageTitle()
	shortcuts := m.currentPageHelpEntries()
	lines := []string{
		helpFooterStyle.Render(fmt.Sprintf("Current page: %s", pageTitle)),
		"",
		helpHeadingStyle.Render("Shortcuts"),
	}
	lines = append(lines, m.renderHelpEntries(shortcuts)...)
	lines = append(lines,
		"",
		helpHeadingStyle.Render("Commands"),
	)
	lines = append(lines, m.renderCommandHelpEntries(availableCommands())...)
	lines = append(lines,
		"",
		helpFooterStyle.Render("Press esc, ?, or f1 to close help."),
	)
	return strings.Join(lines, "\n")
}

func (m Model) renderHelpEntries(entries []helpEntry) []string {
	if len(entries) == 0 {
		return []string{helpFooterStyle.Render("No shortcuts available.")}
	}
	maxKey := 0
	for _, entry := range entries {
		if len(entry.Keys) > maxKey {
			maxKey = len(entry.Keys)
		}
	}
	if maxKey < 8 {
		maxKey = 8
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := fmt.Sprintf("%-*s  %s", maxKey, entry.Keys, entry.Action)
		lines = append(lines, helpItemStyle.Render(line))
	}
	return lines
}

func (m Model) renderCommandHelpEntries(entries []commandHelp) []string {
	if len(entries) == 0 {
		return []string{helpFooterStyle.Render("No commands available.")}
	}
	maxCommand := 0
	for _, entry := range entries {
		if len(entry.Command) > maxCommand {
			maxCommand = len(entry.Command)
		}
	}
	if maxCommand < 12 {
		maxCommand = 12
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := fmt.Sprintf(":%-*s  %s", maxCommand, entry.Command, entry.Usage)
		lines = append(lines, helpItemStyle.Render(line))
	}
	return lines
}

func (m Model) helpPageTitle() string {
	if m.dockerHubActive {
		if m.dockerHubInputFocus {
			return "Docker Hub Search"
		}
		return "Docker Hub Tags"
	}
	if m.githubActive {
		if m.githubInputFocus {
			return "GHCR Search"
		}
		return "GHCR Tags"
	}
	if m.commandActive {
		return "Command Input"
	}
	if m.filterActive {
		return "Filter Input"
	}
	return focusLabel(m.focus)
}

func (m Model) currentPageHelpEntries() []helpEntry {
	entries := []helpEntry{
		{Keys: "?", Action: "Open/close help"},
		{Keys: ":", Action: "Open command input"},
		{Keys: "q / Ctrl+C", Action: "Quit"},
	}

	if m.commandActive {
		entries = append(entries,
			helpEntry{Keys: "Tab", Action: "Autocomplete command"},
			helpEntry{Keys: "Up/Down", Action: "Cycle command suggestions"},
			helpEntry{Keys: "Enter", Action: "Run command"},
			helpEntry{Keys: "Esc", Action: "Close command input"},
		)
		return entries
	}
	if m.filterActive {
		entries = append(entries,
			helpEntry{Keys: "Type", Action: "Set filter text"},
			helpEntry{Keys: "Enter", Action: "Apply and close filter input"},
			helpEntry{Keys: "Esc", Action: "Clear filter"},
		)
		return entries
	}
	if m.dockerHubActive && m.dockerHubInputFocus {
		entries = append(entries,
			helpEntry{Keys: "Type", Action: "Set Docker Hub image query"},
			helpEntry{Keys: "Enter", Action: "Search image tags"},
			helpEntry{Keys: "Esc", Action: "Exit Docker Hub mode"},
		)
		return entries
	}
	if m.githubActive && m.githubInputFocus {
		entries = append(entries,
			helpEntry{Keys: "Type", Action: "Set GHCR image query"},
			helpEntry{Keys: "Enter", Action: "Search image tags"},
			helpEntry{Keys: "Esc", Action: "Exit GHCR mode"},
		)
		return entries
	}

	entries = append(entries,
		helpEntry{Keys: "/", Action: "Filter current list"},
		helpEntry{Keys: "Up/Down, j/k", Action: "Move selection"},
		helpEntry{Keys: "PgUp/PgDn, b/f", Action: "Move one page"},
		helpEntry{Keys: "Ctrl+U/Ctrl+D", Action: "Move half page"},
		helpEntry{Keys: "Home/End, g/G", Action: "Jump to top/bottom"},
		helpEntry{Keys: "r", Action: "Refresh current data"},
	)

	if m.dockerHubActive {
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected tag"},
			helpEntry{Keys: "c", Action: "Copy selected image:tag"},
			helpEntry{Keys: "s", Action: "Focus Docker Hub search input"},
			helpEntry{Keys: "Esc", Action: "Exit Docker Hub mode"},
		)
		return entries
	}
	if m.githubActive {
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected tag"},
			helpEntry{Keys: "c", Action: "Copy selected image:tag"},
			helpEntry{Keys: "s", Action: "Focus GHCR search input"},
			helpEntry{Keys: "Esc", Action: "Exit GHCR mode"},
		)
		return entries
	}

	switch m.focus {
	case FocusProjects:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected project images"},
			helpEntry{Keys: "Esc", Action: "Clear filter / stay on projects"},
		)
	case FocusImages:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected image tags"},
			helpEntry{Keys: "Esc", Action: "Back to projects (when available)"},
		)
	case FocusTags:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected tag history"},
			helpEntry{Keys: "c", Action: "Copy selected image:tag"},
			helpEntry{Keys: "Esc", Action: "Back to images"},
		)
	case FocusHistory:
		entries = append(entries,
			helpEntry{Keys: "Esc", Action: "Back to tags"},
		)
	default:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected item"},
			helpEntry{Keys: "Esc", Action: "Go back one level"},
		)
	}

	return entries
}

func (m Model) openHelp() (tea.Model, tea.Cmd) {
	m.helpActive = true
	return m, nil
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "f1":
		m.helpActive = false
		return m, nil
	case "enter":
		m.helpActive = false
		return m, nil
	case "q", "ctrl+c":
		m.helpActive = false
		return m.openQuitConfirm()
	default:
		return m, nil
	}
}

func isHelpShortcut(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "?", "f1":
		return true
	default:
		return false
	}
}
