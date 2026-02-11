package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.clearFilter()
			m.syncTable()
			return m, nil
		case ":":
			return m.enterCommandMode()
		case "enter":
			m.stopFilterEditing()
			m.syncTable()
			return m, nil
		}
		before := m.filterInput.Value()
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		if m.filterInput.Value() != before {
			m.table.SetCursor(0)
			m.syncTable()
		}
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "esc":
		return m, m.handleEscape()
	case "/":
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.syncTable()
		return m, nil
	case ":":
		return m.enterCommandMode()
	case "r":
		return m, m.refreshCurrent()
	case "enter":
		return m, m.handleEnter()
	}

	if len(msg.Runes) == 1 && msg.Runes[0] == ':' {
		return m.enterCommandMode()
	}
	if m.handleTableNavKey(msg) {
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleDockerHubKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleExternalKey(externalModeDockerHub, msg)
}

func (m Model) handleGitHubKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleExternalKey(externalModeGitHub, msg)
}

func (m *Model) handleTableNavKey(msg tea.KeyMsg) bool {
	rowCount := len(m.table.Rows())
	if rowCount == 0 {
		return false
	}
	step := maxInt(1, m.table.Height())

	switch msg.String() {
	case "up", "k":
		m.table.MoveUp(1)
		return true
	case "down", "j":
		m.table.MoveDown(1)
		return true
	case "pgup", "b":
		m.table.MoveUp(step)
		return true
	case "pgdown", "f", " ":
		m.table.MoveDown(step)
		return true
	case "ctrl+u", "u":
		m.table.MoveUp(maxInt(1, step/2))
		return true
	case "ctrl+d", "d":
		m.table.MoveDown(maxInt(1, step/2))
		return true
	case "home", "g":
		m.table.GotoTop()
		return true
	case "end", "G":
		m.table.GotoBottom()
		return true
	default:
		return false
	}
}
