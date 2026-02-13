package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m Model) handleExternalKey(kind externalModeKind, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
			return m, tea.Batch(cmd, m.maybeLoadExternalForFilter(kind))
		}
		return m, cmd
	}

	if m.externalInputFocused(kind) {
		switch {
		case isShortcut(msg, shortcutForceQuit):
			return m.openQuitConfirm()
		case isShortcut(msg, shortcutExitExternalMode):
			return m.exitExternalMode(kind)
		case isShortcut(msg, shortcutSearchExternal):
			query := strings.TrimSpace(m.externalInputValue(kind))
			if query == "" {
				m.status = kind.searchPlaceholder()
				return m, nil
			}
			return m, m.searchExternal(kind, query)
		}
		return m, m.updateExternalInput(kind, msg)
	}

	switch {
	case isShortcut(msg, shortcutQuit):
		return m.openQuitConfirm()
	case isShortcut(msg, shortcutBack):
		if m.focus == FocusHistory {
			return m, m.handleEscape()
		}
		return m.exitExternalMode(kind)
	case isShortcut(msg, shortcutCopyImageTag):
		m.copySelectedTagReference()
		return m, nil
	case isShortcut(msg, shortcutPullImageTag):
		return m, m.pullSelectedTagWithDocker()
	case isShortcut(msg, shortcutOpenCommand):
		return m.enterCommandMode()
	case isShortcut(msg, shortcutOpenExternalTagHistory):
		return m, m.openExternalTagHistory(kind)
	case isShortcut(msg, shortcutFocusExternalSearch):
		m.setExternalInputValue(kind, "")
		m.setExternalInputFocus(kind, true)
		cmd := m.focusExternalInput(kind)
		m.externalInputCursorEnd(kind)
		return m, cmd
	case isShortcut(msg, shortcutOpenFilter):
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.syncTable()
		return m, nil
	case isShortcut(msg, shortcutRefresh):
		return m, m.refreshExternal(kind)
	}
	if m.handleTableNavKey(msg) {
		return m, m.maybeLoadExternalOnBottomKey(kind, msg)
	}

	if len(msg.Runes) > 0 || msg.String() == "backspace" || msg.String() == "delete" {
		m.setExternalInputFocus(kind, true)
		if !m.isExternalInputFocused(kind) {
			return m, m.focusExternalInput(kind)
		}
		return m, m.updateExternalInput(kind, msg)
	}

	return m, nil
}

func (m Model) handleExternalMouse(kind externalModeKind, msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.handleTableMouse(msg) {
		if m.externalInputFocused(kind) {
			m.setExternalInputFocus(kind, false)
			m.blurExternalInput(kind)
			m.table.Focus()
		}
		if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown {
			return m, m.maybeLoadExternalOnBottom(kind)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) enterExternalMode(kind externalModeKind) (tea.Model, tea.Cmd) {
	other := externalModeDockerHub
	if kind == externalModeDockerHub {
		other = externalModeGitHub
	}

	if m.externalActive(other) {
		m.focus = m.externalPrevFocus(other)
		if prev := m.externalPrevStatus(other); prev != "" {
			m.status = prev
		}
	}

	m.setExternalActive(other, false)
	m.setExternalInputFocus(other, false)
	m.blurExternalInput(other)
	m.setExternalLoading(other, false)

	m.setExternalActive(kind, true)
	m.setExternalPrevFocus(kind, m.focus)
	m.setExternalPrevStatus(kind, m.status)
	m.focus = kind.focus()
	m.status = kind.modeStatus()
	m.setExternalInputFocus(kind, true)
	cmd := m.focusExternalInput(kind)
	m.externalInputCursorEnd(kind)
	m.clearFilter()
	m.syncTable()
	return m, cmd
}

func (m Model) exitExternalMode(kind externalModeKind) (tea.Model, tea.Cmd) {
	m.setExternalActive(kind, false)
	m.setExternalInputFocus(kind, false)
	m.blurExternalInput(kind)
	m.setExternalLoading(kind, false)
	m.focus = m.externalPrevFocus(kind)
	if prev := m.externalPrevStatus(kind); prev != "" {
		m.status = prev
	}
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m *Model) refreshExternal(kind externalModeKind) tea.Cmd {
	query := strings.TrimSpace(m.externalInputValue(kind))
	if query == "" {
		m.status = kind.searchPlaceholder()
		return nil
	}
	return m.searchExternal(kind, query)
}

func (m *Model) searchExternal(kind externalModeKind, query string) tea.Cmd {
	if m.externalLoading(kind) {
		switch kind {
		case externalModeGitHub:
			m.status = "GHCR request already in progress"
		default:
			m.status = "Docker Hub request already in progress"
		}
		return nil
	}

	m.setExternalInputFocus(kind, false)
	m.blurExternalInput(kind)
	m.table.Focus()
	m.status = kind.searchingStatus(query)
	m.setExternalTags(kind, nil)
	m.setExternalImage(kind, "")
	m.setExternalNext(kind, "")
	m.setExternalLoading(kind, true)
	if kind == externalModeDockerHub {
		m.dockerHubRateLimit = registry.DockerHubRateLimit{}
		m.dockerHubRetryUntil = time.Time{}
	}
	m.startLoading()
	m.syncTable()

	switch kind {
	case externalModeGitHub:
		return loadGitHubTagsFirstPageCmd(query, m.logger)
	default:
		return loadDockerHubTagsFirstPageCmd(query, m.logger)
	}
}

func (m *Model) openExternalTagHistory(kind externalModeKind) tea.Cmd {
	if m.focus != kind.focus() {
		return nil
	}

	list := m.listView()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(list.indices) {
		return nil
	}
	index := list.indices[cursor]
	if index < 0 || index >= len(m.externalTags(kind)) {
		return nil
	}
	image := strings.TrimSpace(m.externalImage(kind))
	if image == "" {
		m.status = "Select an image first"
		return nil
	}

	selected := m.externalTags(kind)[index]
	m.selectedImage = registry.Image{Name: image}
	m.hasSelectedImage = true
	m.selectedTag = selected
	m.hasSelectedTag = true
	m.history = nil
	m.focus = FocusHistory
	m.status = kind.loadingHistoryStatus(image, selected.Name)
	m.clearFilter()
	m.syncTable()
	m.startLoading()

	switch kind {
	case externalModeGitHub:
		return loadGitHubHistoryCmd(image, selected.Name, m.logger)
	default:
		return loadDockerHubHistoryCmd(image, selected.Name, m.logger)
	}
}

func (m *Model) maybeLoadExternalOnBottomKey(kind externalModeKind, msg tea.KeyMsg) tea.Cmd {
	switch {
	case isShortcut(msg, shortcutMoveDown),
		isShortcut(msg, shortcutMovePageDown),
		isShortcut(msg, shortcutMoveHalfDown),
		isShortcut(msg, shortcutMoveBottom):
	default:
		return nil
	}
	return m.maybeLoadExternalOnBottom(kind)
}

func (m *Model) maybeLoadExternalOnBottom(kind externalModeKind) tea.Cmd {
	if m.focus != kind.focus() {
		return nil
	}
	rows := m.table.Rows()
	if len(rows) == 0 {
		return nil
	}
	if m.table.Cursor() < len(rows)-1 {
		return nil
	}
	return m.requestNextExternalPage(kind, false)
}

func (m *Model) maybeLoadExternalForFilter(kind externalModeKind) tea.Cmd {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return nil
	}
	if m.focus != kind.focus() {
		return nil
	}
	if len(m.table.Rows()) >= maxInt(1, m.table.Height()) {
		return nil
	}
	return m.requestNextExternalPage(kind, true)
}

func (m *Model) requestNextExternalPage(kind externalModeKind, forFilter bool) tea.Cmd {
	if m.externalLoading(kind) || m.externalNext(kind) == "" || m.externalImage(kind) == "" {
		return nil
	}

	if kind == externalModeDockerHub {
		now := time.Now()
		if !m.dockerHubRetryUntil.IsZero() && now.Before(m.dockerHubRetryUntil) {
			m.status = m.dockerHubRateLimitStatus("Docker Hub rate limit reached")
			return nil
		}
		if m.dockerHubRateLimit.Remaining == 0 && !m.dockerHubRateLimit.ResetAt.IsZero() && now.Before(m.dockerHubRateLimit.ResetAt) {
			m.dockerHubRetryUntil = m.dockerHubRateLimit.ResetAt
			m.status = m.dockerHubRateLimitStatus("Docker Hub rate limit reached")
			return nil
		}
	}

	m.status = kind.loadingMoreStatus(m.externalImage(kind), forFilter)
	m.setExternalLoading(kind, true)
	m.startLoading()

	switch kind {
	case externalModeGitHub:
		return loadGitHubTagsNextPageCmd(m.githubImage, m.githubNext, m.logger)
	default:
		return loadDockerHubTagsNextPageCmd(m.dockerHubImage, m.dockerHubNext, m.logger)
	}
}

func (m Model) externalLoadedStatus(kind externalModeKind) string {
	status := kind.loadedStatus(m.externalImage(kind), len(m.externalTags(kind)), m.externalNext(kind) != "")
	if kind == externalModeDockerHub {
		return status + m.dockerHubRateLimitSuffix()
	}
	return status
}
