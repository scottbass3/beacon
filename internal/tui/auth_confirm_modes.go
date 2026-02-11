package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "tab", "down":
		m.authFocus = (m.authFocus + 1) % m.authFieldCount()
		m.syncAuthFocus()
	case "shift+tab", "up":
		m.authFocus--
		if m.authFocus < 0 {
			m.authFocus = m.authFieldCount() - 1
		}
		m.syncAuthFocus()
	case " ":
		if m.authFocus == 2 && m.authUI().ShowRemember {
			m.remember = !m.remember
		}
	case "enter":
		return m.submitAuth()
	}

	var cmd tea.Cmd
	switch m.authFocus {
	case 0:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	case 1:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "shift+tab":
		m.confirmFocus = 0
	case "right", "l", "tab":
		m.confirmFocus = 1
	case "esc", "n":
		m.clearConfirm()
		return m, nil
	case "y":
		return m.resolveConfirm(true)
	case "enter":
		return m.resolveConfirm(m.confirmFocus == 1)
	case "ctrl+c", "q":
		return m.resolveConfirm(true)
	}
	return m, nil
}

func (m Model) openQuitConfirm() (tea.Model, tea.Cmd) {
	m.confirmAction = confirmActionQuit
	m.confirmTitle = "Quit Beacon?"
	if m.isLoading() {
		m.confirmMessage = "A request is still in progress."
	} else {
		m.confirmMessage = "Close the current session?"
	}
	m.confirmFocus = 0
	return m, nil
}

func (m Model) resolveConfirm(accept bool) (tea.Model, tea.Cmd) {
	action := m.confirmAction
	m.clearConfirm()
	if !accept {
		return m, nil
	}
	switch action {
	case confirmActionQuit:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m *Model) clearConfirm() {
	m.confirmAction = confirmActionNone
	m.confirmTitle = ""
	m.confirmMessage = ""
	m.confirmFocus = 0
}

func (m Model) submitAuth() (tea.Model, tea.Cmd) {
	auth := m.auth
	switch auth.Kind {
	case "registry_v2":
		auth.RegistryV2.Username = strings.TrimSpace(m.usernameInput.Value())
		auth.RegistryV2.Password = m.passwordInput.Value()
		auth.RegistryV2.Remember = m.remember
		if !auth.RegistryV2.Remember {
			auth.RegistryV2.RefreshToken = ""
		}
	case "harbor":
		auth.Harbor.Username = strings.TrimSpace(m.usernameInput.Value())
		auth.Harbor.Password = m.passwordInput.Value()
	}

	client, err := registry.NewClientWithLogger(m.registryHost, auth, m.logger)
	if err != nil {
		m.authError = err.Error()
		return m, nil
	}

	registry.PersistAuthCache(m.registryHost, auth)
	m.auth = auth
	m.registryClient = client
	m.authRequired = false
	m.authError = ""
	return m, m.initialLoadCmd()
}

func (m Model) enterDockerHubMode() (tea.Model, tea.Cmd) {
	return m.enterExternalMode(externalModeDockerHub)
}

func (m Model) exitDockerHubMode() (tea.Model, tea.Cmd) {
	return m.exitExternalMode(externalModeDockerHub)
}

func (m Model) enterGitHubMode() (tea.Model, tea.Cmd) {
	return m.enterExternalMode(externalModeGitHub)
}

func (m Model) exitGitHubMode() (tea.Model, tea.Cmd) {
	return m.exitExternalMode(externalModeGitHub)
}
