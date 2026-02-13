package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch {
		case isShortcut(msg, shortcutClearFilter):
			m.clearFilter()
			m.syncTable()
			return m, nil
		case isShortcut(msg, shortcutOpenCommand):
			return m.enterCommandMode()
		case isShortcut(msg, shortcutApplyFilter):
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

	switch {
	case isShortcut(msg, shortcutQuit):
		return m.openQuitConfirm()
	case isShortcut(msg, shortcutBack):
		return m, m.handleEscape()
	case isShortcut(msg, shortcutCopyImageTag):
		m.copySelectedTagReference()
		return m, nil
	case isShortcut(msg, shortcutOpenFilter):
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.syncTable()
		return m, nil
	case isShortcut(msg, shortcutOpenCommand):
		return m.enterCommandMode()
	case isShortcut(msg, shortcutRefresh):
		return m, m.refreshCurrent()
	case isShortcut(msg, shortcutOpenTagHistory):
		return m, m.handleEnter()
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

	switch {
	case isShortcut(msg, shortcutMoveUp):
		m.table.MoveUp(1)
		return true
	case isShortcut(msg, shortcutMoveDown):
		m.table.MoveDown(1)
		return true
	case isShortcut(msg, shortcutMovePageUp):
		m.table.MoveUp(step)
		return true
	case isShortcut(msg, shortcutMovePageDown):
		m.table.MoveDown(step)
		return true
	case isShortcut(msg, shortcutMoveHalfUp):
		m.table.MoveUp(maxInt(1, step/2))
		return true
	case isShortcut(msg, shortcutMoveHalfDown):
		m.table.MoveDown(maxInt(1, step/2))
		return true
	case isShortcut(msg, shortcutMoveTop):
		m.table.GotoTop()
		return true
	case isShortcut(msg, shortcutMoveBottom):
		m.table.GotoBottom()
		return true
	default:
		return false
	}
}
