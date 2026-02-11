package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleContextFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m.openQuitConfirm()
	case "q":
		return m.openQuitConfirm()
	case "esc":
		return m.cancelContextForm()
	case "tab", "down":
		m.contextFormFocus = m.nextContextFormFocus(m.contextFormFocus)
		return m, m.syncContextFormFocus()
	case "shift+tab", "up":
		m.contextFormFocus = m.prevContextFormFocus(m.contextFormFocus)
		return m, m.syncContextFormFocus()
	case "left", "h":
		if m.shouldSwapContextFormActions() {
			if m.contextFormFocus == contextFormFocusSecondaryButton {
				m.contextFormFocus = contextFormFocusPrimaryButton
				return m, m.syncContextFormFocus()
			}
		} else {
			if m.contextFormFocus == contextFormFocusPrimaryButton {
				m.contextFormFocus = contextFormFocusSecondaryButton
				return m, m.syncContextFormFocus()
			}
		}
	case "right", "l":
		if m.shouldSwapContextFormActions() {
			if m.contextFormFocus == contextFormFocusPrimaryButton {
				m.contextFormFocus = contextFormFocusSecondaryButton
				return m, m.syncContextFormFocus()
			}
		} else {
			if m.contextFormFocus == contextFormFocusSecondaryButton {
				m.contextFormFocus = contextFormFocusPrimaryButton
				return m, m.syncContextFormFocus()
			}
		}
	case " ":
		if m.contextFormFocus == contextFormFocusAnonymous {
			m.contextFormAnonymous = !m.contextFormAnonymous
			return m, nil
		}
	case "enter":
		switch m.contextFormFocus {
		case contextFormFocusSecondaryButton:
			return m.cancelContextForm()
		case contextFormFocusPrimaryButton:
			return m.submitContextForm()
		case contextFormFocusAnonymous:
			m.contextFormAnonymous = !m.contextFormAnonymous
			return m, nil
		default:
			m.contextFormFocus = m.nextContextFormFocus(m.contextFormFocus)
			return m, m.syncContextFormFocus()
		}
	}

	var cmd tea.Cmd
	switch m.contextFormFocus {
	case contextFormFocusName:
		m.contextFormNameInput, cmd = m.contextFormNameInput.Update(msg)
	case contextFormFocusRegistry:
		m.contextFormRegistryInput, cmd = m.contextFormRegistryInput.Update(msg)
	case contextFormFocusKind:
		m.contextFormKindInput, cmd = m.contextFormKindInput.Update(msg)
	case contextFormFocusService:
		m.contextFormServiceInput, cmd = m.contextFormServiceInput.Update(msg)
	}
	return m, cmd
}

func (m Model) shouldSwapContextFormActions() bool {
	return m.contextFormAllowSkip && len(m.contexts) == 0 && m.contextFormMode == contextFormModeAdd
}

func (m Model) nextContextFormFocus(current int) int {
	if !m.shouldSwapContextFormActions() {
		return (current + 1) % contextFormFocusCount
	}
	switch current {
	case contextFormFocusName:
		return contextFormFocusRegistry
	case contextFormFocusRegistry:
		return contextFormFocusKind
	case contextFormFocusKind:
		return contextFormFocusService
	case contextFormFocusService:
		return contextFormFocusAnonymous
	case contextFormFocusAnonymous:
		return contextFormFocusPrimaryButton
	case contextFormFocusPrimaryButton:
		return contextFormFocusSecondaryButton
	case contextFormFocusSecondaryButton:
		return contextFormFocusName
	default:
		return contextFormFocusName
	}
}

func (m Model) prevContextFormFocus(current int) int {
	if !m.shouldSwapContextFormActions() {
		current--
		if current < 0 {
			return contextFormFocusCount - 1
		}
		return current
	}
	switch current {
	case contextFormFocusName:
		return contextFormFocusSecondaryButton
	case contextFormFocusRegistry:
		return contextFormFocusName
	case contextFormFocusKind:
		return contextFormFocusRegistry
	case contextFormFocusService:
		return contextFormFocusKind
	case contextFormFocusAnonymous:
		return contextFormFocusService
	case contextFormFocusPrimaryButton:
		return contextFormFocusAnonymous
	case contextFormFocusSecondaryButton:
		return contextFormFocusPrimaryButton
	default:
		return contextFormFocusName
	}
}
