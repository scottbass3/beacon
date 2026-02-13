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
		helpFooterStyle.Render("Press esc, ?, f1, or enter to close help."),
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
	return m.shortcutPageTitle(false)
}

func (m Model) openHelp() (tea.Model, tea.Cmd) {
	m.helpActive = true
	return m, nil
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isShortcut(msg, shortcutCloseHelp):
		m.helpActive = false
		return m, nil
	case isShortcut(msg, shortcutQuit):
		m.helpActive = false
		return m.openQuitConfirm()
	default:
		return m, nil
	}
}

func isHelpShortcut(msg tea.KeyMsg) bool {
	return isShortcut(msg, shortcutOpenHelp)
}
