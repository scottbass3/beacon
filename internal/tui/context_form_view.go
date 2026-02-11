package tui

import (
	"strings"

	lipglossv2 "github.com/charmbracelet/lipgloss/v2"
)

func (m Model) renderContextFormModal() string {
	title := "Add Context"
	subtitle := "Enter context details."
	if m.contextFormMode == contextFormModeEdit {
		title = "Edit Context"
		subtitle = "Update context details."
	} else if m.contextFormAllowSkip && len(m.contexts) == 0 {
		subtitle = "Add a context now or continue without one."
	}

	name := m.contextFormNameInput.View()
	registryHost := m.contextFormRegistryInput.View()
	kind := m.contextFormKindInput.View()
	service := m.contextFormServiceInput.View()

	if m.contextFormFocus == contextFormFocusName {
		name = modalInputFocusStyle.Render(name)
	} else {
		name = modalInputStyle.Render(name)
	}
	if m.contextFormFocus == contextFormFocusRegistry {
		registryHost = modalInputFocusStyle.Render(registryHost)
	} else {
		registryHost = modalInputStyle.Render(registryHost)
	}
	if m.contextFormFocus == contextFormFocusKind {
		kind = modalInputFocusStyle.Render(kind)
	} else {
		kind = modalInputStyle.Render(kind)
	}
	if m.contextFormFocus == contextFormFocusService {
		service = modalInputFocusStyle.Render(service)
	} else {
		service = modalInputStyle.Render(service)
	}

	anonymous := "[ ] Anonymous"
	if m.contextFormAnonymous {
		anonymous = "[x] Anonymous"
	}
	if m.contextFormFocus == contextFormFocusAnonymous {
		anonymous = modalFocusStyle.Render(anonymous)
	} else {
		anonymous = modalLabelStyle.Render(anonymous)
	}

	secondaryLabel := "Cancel"
	if m.contextFormAllowSkip && len(m.contexts) == 0 {
		secondaryLabel = "Continue without context"
	}
	secondary := modalButtonStyle.Render(secondaryLabel)
	if m.contextFormFocus == contextFormFocusSecondaryButton {
		secondary = modalButtonFocusStyle.Render(secondaryLabel)
	}

	primaryLabel := "Add Context"
	if m.contextFormMode == contextFormModeEdit {
		primaryLabel = "Save Context"
	}
	primary := modalButtonStyle.Render(primaryLabel)
	if m.contextFormFocus == contextFormFocusPrimaryButton {
		primary = modalButtonFocusStyle.Render(primaryLabel)
	}
	leftButton := lipglossv2.NewStyle().MarginRight(2).Render(secondary)
	rightButton := primary
	if m.shouldSwapContextFormActions() {
		leftButton = lipglossv2.NewStyle().MarginRight(2).Render(primary)
		rightButton = secondary
	}
	buttonRow := lipglossv2.JoinHorizontal(
		lipglossv2.Top,
		leftButton,
		rightButton,
	)

	lines := []string{
		modalTitleStyle.Render(title),
		modalLabelStyle.Render(subtitle),
		modalDividerStyle.Render(strings.Repeat("─", 24)),
	}
	if m.contextFormError != "" {
		lines = append(lines, modalErrorStyle.Render(m.contextFormError))
	}
	lines = append(lines,
		"",
		modalLabelStyle.Render("Name"),
		name,
		modalLabelStyle.Render("Registry"),
		registryHost,
		modalLabelStyle.Render("Kind"),
		kind,
		modalLabelStyle.Render("Service"),
		service,
		anonymous,
		"",
		buttonRow,
		"",
		modalHelpStyle.Render("tab/shift+tab move  space toggle anonymous  enter select  esc cancel"),
	)
	return m.renderModalCard(strings.Join(lines, "\n"), 88)
}
