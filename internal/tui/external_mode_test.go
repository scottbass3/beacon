package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestEnterOnExternalTagsOpensHistory(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*Model)
		handleKey func(Model, tea.KeyMsg) (tea.Model, tea.Cmd)
		wantImage string
		wantTag   string
	}{
		{
			name: "dockerhub",
			setup: func(m *Model) {
				m.dockerHubActive = true
				m.focus = FocusDockerHubTags
				m.dockerHubImage = "library/nginx"
				m.dockerHubTags = []registry.Tag{{Name: "alpine"}}
			},
			handleKey: func(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
				return m.handleDockerHubKey(msg)
			},
			wantImage: "library/nginx",
			wantTag:   "alpine",
		},
		{
			name: "github",
			setup: func(m *Model) {
				m.githubActive = true
				m.focus = FocusGitHubTags
				m.githubImage = "org/service"
				m.githubTags = []registry.Tag{{Name: "v1.2.3"}}
			},
			handleKey: func(m Model, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
				return m.handleGitHubKey(msg)
			},
			wantImage: "org/service",
			wantTag:   "v1.2.3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			auth := registry.Auth{Kind: "registry_v2"}
			auth.RegistryV2.Anonymous = true
			m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
			tc.setup(&m)
			m.syncTable()

			updated, cmd := tc.handleKey(m, tea.KeyMsg{Type: tea.KeyEnter})
			next := updated.(Model)

			if cmd == nil {
				t.Fatalf("expected history load command")
			}
			if next.focus != FocusHistory {
				t.Fatalf("expected focus history, got %v", next.focus)
			}
			if !next.hasSelectedImage || next.selectedImage.Name != tc.wantImage {
				t.Fatalf("expected selected image %q, got %#v", tc.wantImage, next.selectedImage)
			}
			if !next.hasSelectedTag || next.selectedTag.Name != tc.wantTag {
				t.Fatalf("expected selected tag %q, got %#v", tc.wantTag, next.selectedTag)
			}
			if !strings.Contains(strings.ToLower(next.status), "history") {
				t.Fatalf("expected history status, got %q", next.status)
			}
		})
	}
}

func TestExternalSearchInputConsumesShortcutKeys(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.dockerHubActive = true
	m.focus = FocusDockerHubTags
	m.dockerHubInputFocus = true
	m.dockerHubInput.Focus()
	m.status = "unchanged"

	updated, _ := m.handleDockerHubKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	next := updated.(Model)
	if next.dockerHubInput.Value() != "r" {
		t.Fatalf("expected input to capture 'r', got %q", next.dockerHubInput.Value())
	}
	if next.commandActive {
		t.Fatalf("command mode should not open while search input is focused")
	}
	if next.status != "unchanged" {
		t.Fatalf("status changed unexpectedly: %q", next.status)
	}

	updated, _ = next.handleDockerHubKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	next = updated.(Model)
	if next.dockerHubInput.Value() != "r:" {
		t.Fatalf("expected ':' to be typed into search input, got %q", next.dockerHubInput.Value())
	}
	if next.commandActive {
		t.Fatalf("command mode should stay closed when typing ':' in search input")
	}
}

func TestHelpShortcutIgnoredWhileExternalInputFocused(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.dockerHubActive = true
	m.dockerHubInputFocus = true
	m.dockerHubInput.Focus()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	next := updated.(Model)

	if next.helpActive {
		t.Fatalf("help page should not open while external search input is focused")
	}
	if next.dockerHubInput.Value() != "?" {
		t.Fatalf("expected '?' to be typed into search input, got %q", next.dockerHubInput.Value())
	}
}

func TestCommandShortcutIgnoredWhileExternalInputFocused(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.dockerHubActive = true
	m.dockerHubInputFocus = true
	m.dockerHubInput.Focus()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	next := updated.(Model)

	if next.commandActive {
		t.Fatalf("command mode should not open while external search input is focused")
	}
	if next.dockerHubInput.Value() != ":" {
		t.Fatalf("expected ':' to be typed into search input, got %q", next.dockerHubInput.Value())
	}
}
