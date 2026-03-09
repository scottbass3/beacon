package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func printableASCII() string {
	var b strings.Builder
	for r := rune(32); r <= rune(126); r++ {
		b.WriteRune(r)
	}
	return b.String()
}

func TestAuthInputAllowsAllPrintableASCIIInUsernameAndPassword(t *testing.T) {
	tests := []struct {
		name      string
		focus     int
		inputName string
		value     func(Model) string
	}{
		{
			name:      "username",
			focus:     0,
			inputName: "username",
			value: func(m Model) string {
				return m.usernameInput.Value()
			},
		},
		{
			name:      "password",
			focus:     1,
			inputName: "password",
			value: func(m Model) string {
				return m.passwordInput.Value()
			},
		},
	}
	want := printableASCII()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth := registry.Auth{Kind: "registry_v2"}
			m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
			if !m.isAuthModalActive() {
				t.Fatalf("expected auth modal to be active")
			}
			m.authFocus = tc.focus
			m.syncAuthFocus()

			for _, r := range want {
				updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
				m = updated.(Model)
				if m.isConfirmModalActive() {
					t.Fatalf("did not expect quit confirmation when typing %q in %s", r, tc.inputName)
				}
			}

			if got := tc.value(m); got != want {
				t.Fatalf("expected %s input to capture all printable ASCII chars, got %q", tc.inputName, got)
			}
		})
	}
}
