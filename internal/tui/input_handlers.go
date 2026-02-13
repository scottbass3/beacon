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
			m.tableSetCursor(0)
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
	case isShortcut(msg, shortcutPullImageTag):
		return m, m.pullSelectedTagWithDocker()
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

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.handleTableMouse(msg) {
		return m, nil
	}
	return m, nil
}

func (m *Model) handleTableMouse(msg tea.MouseMsg) bool {
	switch {
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp:
		if _, ok := m.tableRowAtMouse(msg); !ok {
			return false
		}
		m.tableMoveUp(1)
		return true
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown:
		if _, ok := m.tableRowAtMouse(msg); !ok {
			return false
		}
		m.tableMoveDown(1)
		return true
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		row, ok := m.tableRowAtMouse(msg)
		if !ok {
			return false
		}
		m.tableSetCursor(row)
		return true
	default:
		return false
	}
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
		m.tableMoveUp(1)
		return true
	case isShortcut(msg, shortcutMoveDown):
		m.tableMoveDown(1)
		return true
	case isShortcut(msg, shortcutMovePageUp):
		m.tableMoveUp(step)
		return true
	case isShortcut(msg, shortcutMovePageDown):
		m.tableMoveDown(step)
		return true
	case isShortcut(msg, shortcutMoveHalfUp):
		m.tableMoveUp(maxInt(1, step/2))
		return true
	case isShortcut(msg, shortcutMoveHalfDown):
		m.tableMoveDown(maxInt(1, step/2))
		return true
	case isShortcut(msg, shortcutMoveTop):
		m.tableGotoTop()
		return true
	case isShortcut(msg, shortcutMoveBottom):
		m.tableGotoBottom()
		return true
	default:
		return false
	}
}
