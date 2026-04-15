package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	lipglossv1 "github.com/charmbracelet/lipgloss"
)

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
	applyInputTheme(&input)
	return input
}

// applyInputTheme sets explicit background colors on a textinput so its
// rendered cells are opaque in the lipgloss v2 compositor. Without this,
// ANSI reset codes within the textinput output create transparent cells that
// show the layer below when the input is rendered inside a modal overlay.
func applyInputTheme(input *textinput.Model) {
	bg := lipglossv1.Color(rawColorSurface2)
	textStyle := lipglossv1.NewStyle().Foreground(lipglossv1.Color(rawColorTitleText)).Background(bg)
	input.TextStyle = textStyle
	input.Cursor.TextStyle = textStyle
	input.PlaceholderStyle = lipglossv1.NewStyle().Foreground(lipglossv1.Color(rawColorMuted)).Background(bg)
}
