package tui

import "github.com/charmbracelet/bubbles/textinput"

type contextFormMode int

const (
	contextFormModeAdd contextFormMode = iota
	contextFormModeEdit
)

const (
	contextFormFocusName = iota
	contextFormFocusRegistry
	contextFormFocusKind
	contextFormFocusService
	contextFormFocusAnonymous
	contextFormFocusSecondaryButton
	contextFormFocusPrimaryButton
	contextFormFocusCount
)

func newContextInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 256
	return input
}
