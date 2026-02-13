package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestCopySelectedTagReference(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true

	tests := []struct {
		name     string
		setup    func(*Model)
		handle   func(Model) (tea.Model, tea.Cmd)
		wantCopy string
	}{
		{
			name: "registry tags",
			setup: func(m *Model) {
				m.focus = FocusTags
				m.hasSelectedImage = true
				m.selectedImage = registry.Image{Name: "team/service"}
				m.tags = []registry.Tag{{Name: "v1.2.3"}}
				m.syncTable()
			},
			handle: func(m Model) (tea.Model, tea.Cmd) {
				return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
			wantCopy: "team/service:v1.2.3",
		},
		{
			name: "dockerhub tags",
			setup: func(m *Model) {
				m.dockerHubActive = true
				m.focus = FocusDockerHubTags
				m.dockerHubImage = "library/nginx"
				m.dockerHubTags = []registry.Tag{{Name: "alpine"}}
				m.syncTable()
			},
			handle: func(m Model) (tea.Model, tea.Cmd) {
				return m.handleDockerHubKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
			wantCopy: "library/nginx:alpine",
		},
		{
			name: "github tags",
			setup: func(m *Model) {
				m.githubActive = true
				m.focus = FocusGitHubTags
				m.githubImage = "org/service"
				m.githubTags = []registry.Tag{{Name: "latest"}}
				m.syncTable()
			},
			handle: func(m Model) (tea.Model, tea.Cmd) {
				return m.handleGitHubKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
			wantCopy: "org/service:latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
			tc.setup(&m)

			var copied string
			writeClipboard = func(value string) error {
				copied = value
				return nil
			}
			t.Cleanup(func() {
				writeClipboard = clipboardWriteAll
			})

			updated, _ := tc.handle(m)
			next := updated.(Model)

			if copied != tc.wantCopy {
				t.Fatalf("expected copied value %q, got %q", tc.wantCopy, copied)
			}
			if !strings.Contains(next.status, tc.wantCopy) {
				t.Fatalf("expected status to include copied value, got %q", next.status)
			}
		})
	}
}

func TestCopySelectedTagReferenceClipboardError(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.focus = FocusTags
	m.hasSelectedImage = true
	m.selectedImage = registry.Image{Name: "team/service"}
	m.tags = []registry.Tag{{Name: "v1.2.3"}}
	m.syncTable()

	writeClipboard = func(string) error {
		return errors.New("clipboard unavailable")
	}
	t.Cleanup(func() {
		writeClipboard = clipboardWriteAll
	})

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	next := updated.(Model)
	if !strings.Contains(next.status, "Failed to copy") {
		t.Fatalf("expected copy error status, got %q", next.status)
	}
}

func TestCopySelectedTagReferenceWithoutSelection(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.focus = FocusTags
	m.hasSelectedImage = true
	m.selectedImage = registry.Image{Name: "team/service"}

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	next := updated.(Model)
	if next.status != "No tag selected to copy" {
		t.Fatalf("expected no selection status, got %q", next.status)
	}
}
