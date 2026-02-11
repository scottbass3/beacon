package tui

import (
	"fmt"
	"strings"

	lipglossv2 "github.com/charmbracelet/lipgloss/v2"
)

func (m Model) renderAuthModal() string {
	registryHost := strings.TrimSpace(m.registryHost)
	if registryHost == "" {
		registryHost = "-"
	}
	lines := []string{
		modalTitleStyle.Render("Authentication Required"),
		modalLabelStyle.Render(fmt.Sprintf("Registry  %s", registryHost)),
		modalDividerStyle.Render(strings.Repeat("─", 24)),
	}
	if m.authError != "" {
		lines = append(lines, modalErrorStyle.Render(m.authError))
	}

	username := m.usernameInput.View()
	password := m.passwordInput.View()
	if m.authFocus == 0 {
		username = modalInputFocusStyle.Render(username)
	} else {
		username = modalInputStyle.Render(username)
	}
	if m.authFocus == 1 {
		password = modalInputFocusStyle.Render(password)
	} else {
		password = modalInputStyle.Render(password)
	}

	remember := ""
	if m.authUI().ShowRemember {
		remember = "[ ] Remember session"
		if m.remember {
			remember = "[x] Remember session"
		}
	}

	if m.authFocus == 2 && m.authUI().ShowRemember {
		remember = modalFocusStyle.Render(remember)
	} else if m.authUI().ShowRemember {
		remember = modalLabelStyle.Render(remember)
	}

	help := "tab/shift+tab move  enter submit  q quit"
	if m.authUI().ShowRemember {
		help = "tab/shift+tab move  space toggle  enter submit  q quit"
	}

	lines = append(lines,
		"",
		modalLabelStyle.Render("Username"),
		username,
		modalLabelStyle.Render("Password"),
		password,
	)
	if m.authUI().ShowRemember {
		lines = append(lines, remember)
	}
	lines = append(lines,
		"",
		modalHelpStyle.Render(strings.ToUpper(help)),
	)

	return m.renderModalCard(strings.Join(lines, "\n"), 72)
}

func (m Model) renderConfirmModal() string {
	title := strings.TrimSpace(m.confirmTitle)
	if title == "" {
		title = "Confirm action"
	}
	confirmLabel := "Confirm"
	confirmButtonStyle := modalButtonStyle
	confirmButtonFocusStyle := modalButtonFocusStyle
	switch m.confirmAction {
	case confirmActionQuit:
		confirmLabel = "Quit"
		confirmButtonStyle = modalDangerButtonStyle
		confirmButtonFocusStyle = modalDangerFocusStyle
	}

	cancel := "Cancel"
	if m.confirmFocus == 0 {
		cancel = modalButtonFocusStyle.Render(cancel)
	} else {
		cancel = modalButtonStyle.Render(cancel)
	}
	confirm := confirmButtonStyle.Render(confirmLabel)
	if m.confirmFocus == 1 {
		confirm = confirmButtonFocusStyle.Render(confirmLabel)
	}
	buttonRow := lipglossv2.JoinHorizontal(
		lipglossv2.Top,
		lipglossv2.NewStyle().MarginRight(2).Render(cancel),
		confirm,
	)

	lines := []string{
		modalTitleStyle.Render(title),
	}
	if message := strings.TrimSpace(m.confirmMessage); message != "" {
		lines = append(lines, modalLabelStyle.Render(message))
	}
	lines = append(lines,
		"",
		buttonRow,
		"",
		modalHelpStyle.Render("tab/left/right move  enter choose  y/n quick select"),
	)
	return m.renderModalCard(strings.Join(lines, "\n"), 64)
}

func (m Model) renderModal(base, modal string) string {
	width, height := m.modalViewport(base)
	background := lipglossv2.Place(width, height, lipglossv2.Left, lipglossv2.Top, modalBackdropStyle.Render(base))
	canvas := lipglossv2.NewCanvas(lipglossv2.NewLayer(background))
	canvas.AddLayers(
		lipglossv2.NewLayer(modal).
			X(maxInt(0, (width-lipglossv2.Width(modal))/2)).
			Y(maxInt(0, (height-lipglossv2.Height(modal))/2)).
			Z(1),
	)
	return canvas.Render()
}

func (m Model) renderModalCard(content string, maxWidth int) string {
	return modalPanelStyle.Width(m.modalWidth(maxWidth)).Render(content)
}

func (m Model) modalWidth(maxWidth int) int {
	width, _ := m.modalViewport("")
	if width <= 2 {
		return width
	}
	modalWidth := width - 8
	if modalWidth < 24 {
		modalWidth = width - 2
	}
	if maxWidth > 0 && modalWidth > maxWidth {
		modalWidth = maxWidth
	}
	if modalWidth < 12 {
		modalWidth = 12
	}
	return modalWidth
}

func (m Model) modalViewport(base string) (int, int) {
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = maxInt(24, lineCount(base))
	}
	return width, height
}

func (m Model) isContextSelectionActive() bool {
	return m.contextSelectionActive
}

func (m Model) isContextFormActive() bool {
	return m.contextFormActive
}

func (m Model) isAuthModalActive() bool {
	return !m.isContextSelectionActive() && !m.isContextFormActive() && m.authRequired && m.registryClient == nil
}

func (m Model) isConfirmModalActive() bool {
	return m.confirmAction != confirmActionNone
}
