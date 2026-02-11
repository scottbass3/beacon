package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	lipglossv2 "github.com/charmbracelet/lipgloss/v2"
)

func (m Model) contextSelectionHelpText() string {
	if m.contextSelectionRequired {
		return "up/down move  enter select  a add context  q quit"
	}
	return "up/down move  enter select  a add context  esc close  q quit"
}

func (m Model) openContextSelection(required bool) (tea.Model, tea.Cmd) {
	m.contextSelectionActive = true
	m.contextSelectionRequired = required
	m.contextSelectionError = ""
	if len(m.contexts) == 0 {
		m.contextSelectionIndex = 0
		m.status = "No contexts configured"
		m.syncTable()
		return m, nil
	}
	if current := m.currentContextIndex(); current >= 0 {
		m.contextSelectionIndex = current
	}
	m.syncTable()
	return m, nil
}

func (m Model) closeContextSelection() (tea.Model, tea.Cmd) {
	m.contextSelectionActive = false
	m.contextSelectionRequired = false
	m.contextSelectionError = ""
	m.syncTable()
	return m, nil
}

func (m Model) runContextCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		return m.openContextSelection(false)
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "add":
		if len(args) != 1 {
			m.status = "Usage: :context add"
			return m, nil
		}
		return m.openContextFormAdd(false, false)
	case "remove", "rm", "delete":
		if len(args) < 2 {
			m.status = "Usage: :context remove <name>"
			return m, nil
		}
		return m.removeContextByName(strings.Join(args[1:], " "))
	case "edit":
		if len(args) < 2 {
			m.status = "Usage: :context edit <name>"
			return m, nil
		}
		return m.openContextFormEditByName(strings.Join(args[1:], " "))
	default:
		return m.switchContext(strings.Join(args, " "))
	}
}

func (m Model) handleContextSelectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.contexts) == 0 {
		switch msg.String() {
		case "ctrl+c":
			return m.openQuitConfirm()
		case "q":
			return m.openQuitConfirm()
		case "esc":
			if m.contextSelectionRequired {
				return m.openQuitConfirm()
			}
			return m.closeContextSelection()
		case "a":
			return m.openContextFormAdd(true, false)
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m.openQuitConfirm()
	case "q":
		return m.openQuitConfirm()
	case "esc":
		if m.contextSelectionRequired {
			return m.openQuitConfirm()
		}
		return m.closeContextSelection()
	case "up", "k", "shift+tab":
		m.contextSelectionIndex--
		if m.contextSelectionIndex < 0 {
			m.contextSelectionIndex = len(m.contexts) - 1
		}
		m.contextSelectionError = ""
		return m, nil
	case "down", "j", "tab":
		m.contextSelectionIndex = (m.contextSelectionIndex + 1) % len(m.contexts)
		m.contextSelectionError = ""
		return m, nil
	case "home", "g":
		m.contextSelectionIndex = 0
		m.contextSelectionError = ""
		return m, nil
	case "end", "G":
		m.contextSelectionIndex = len(m.contexts) - 1
		m.contextSelectionError = ""
		return m, nil
	case "a":
		return m.openContextFormAdd(true, false)
	case "enter":
		selected := clampInt(m.contextSelectionIndex, 0, len(m.contexts)-1)
		return m.switchContextAt(selected)
	}

	return m, nil
}

func (m Model) renderContextSelectionModal() string {
	lines := []string{
		modalTitleStyle.Render("Select Context"),
		modalLabelStyle.Render("Choose a registry context to continue."),
		modalDividerStyle.Render(strings.Repeat("─", 24)),
	}
	if m.contextSelectionError != "" {
		lines = append(lines, modalErrorStyle.Render(m.contextSelectionError))
	}
	if len(m.contexts) == 0 {
		lines = append(lines,
			modalErrorStyle.Render("No contexts configured."),
			"",
			modalHelpStyle.Render("a add context  esc close  q quit"),
		)
		return m.renderModalCard(strings.Join(lines, "\n"), 84)
	}

	selected := clampInt(m.contextSelectionIndex, 0, len(m.contexts)-1)
	for i, ctx := range m.contexts {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}

		name := contextDisplayName(ctx, i)
		host := strings.TrimSpace(ctx.Host)
		hostLabel := modalOptionMutedStyle.Render(host)
		if host == "" {
			hostLabel = modalOptionErrorStyle.Render("(no registry configured)")
		}

		row := prefix + lipglossv2.JoinHorizontal(
			lipglossv2.Top,
			name,
			"  ",
			hostLabel,
		)

		style := modalOptionStyle
		if i == selected {
			style = modalOptionFocusStyle
		}
		lines = append(lines, style.Render(row))
	}
	lines = append(lines,
		"",
		modalHelpStyle.Render(m.contextSelectionHelpText()),
	)
	return m.renderModalCard(strings.Join(lines, "\n"), 84)
}
