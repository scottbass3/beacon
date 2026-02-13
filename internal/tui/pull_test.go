package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestPullSelectedTagWithDocker(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true

	tests := []struct {
		name     string
		setup    func(*Model)
		handle   func(Model) (tea.Model, tea.Cmd)
		wantPull string
	}{
		{
			name: "registry tags",
			setup: func(m *Model) {
				m.focus = FocusTags
				m.selectedProject = "team"
				m.hasSelectedProject = true
				m.hasSelectedImage = true
				m.selectedImage = registry.Image{Name: "team/service"}
				m.tags = []registry.Tag{{Name: "v1.2.3"}}
				m.syncTable()
			},
			handle: func(m Model) (tea.Model, tea.Cmd) {
				return m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
			},
			wantPull: "registry.example.com/team/service:v1.2.3",
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
				return m.handleDockerHubKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
			},
			wantPull: "library/nginx:alpine",
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
				return m.handleGitHubKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
			},
			wantPull: "org/service:latest",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
			tc.setup(&m)

			var pulled string
			runDockerPull = func(reference string) error {
				pulled = reference
				return nil
			}
			t.Cleanup(func() {
				runDockerPull = dockerPull
			})

			updated, cmd := tc.handle(m)
			next := updated.(Model)

			if cmd == nil {
				t.Fatalf("expected pull command")
			}
			if next.status != "Pulling "+tc.wantPull+"..." {
				t.Fatalf("expected pulling status, got %q", next.status)
			}
			if pulled != "" {
				t.Fatalf("pull should run in command, got %q", pulled)
			}

			msg := cmd()
			pullMsg, ok := msg.(dockerPullMsg)
			if !ok {
				t.Fatalf("expected dockerPullMsg, got %T", msg)
			}
			if pullMsg.reference != tc.wantPull {
				t.Fatalf("expected pulled reference %q, got %q", tc.wantPull, pullMsg.reference)
			}
			if pulled != tc.wantPull {
				t.Fatalf("expected pull command reference %q, got %q", tc.wantPull, pulled)
			}

			finalModel, _ := next.Update(msg)
			final := finalModel.(Model)
			if final.status != "Pulled "+tc.wantPull {
				t.Fatalf("expected pulled status, got %q", final.status)
			}
		})
	}
}

func TestPullSelectedTagWithDockerError(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.focus = FocusTags
	m.hasSelectedImage = true
	m.selectedImage = registry.Image{Name: "team/service"}
	m.tags = []registry.Tag{{Name: "v1.2.3"}}
	m.syncTable()

	runDockerPull = func(string) error {
		return errors.New("docker unavailable")
	}
	t.Cleanup(func() {
		runDockerPull = dockerPull
	})

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	next := updated.(Model)
	if cmd == nil {
		t.Fatalf("expected pull command")
	}

	finalModel, _ := next.Update(cmd())
	final := finalModel.(Model)
	if !strings.Contains(final.status, "Failed to pull") {
		t.Fatalf("expected failure status, got %q", final.status)
	}
}

func TestPullSelectedTagWithDockerWithoutSelection(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.focus = FocusTags
	m.hasSelectedImage = true
	m.selectedImage = registry.Image{Name: "team/service"}

	updated, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	next := updated.(Model)
	if cmd != nil {
		t.Fatalf("did not expect pull command when no tag is selected")
	}
	if next.status != "No tag selected to pull" {
		t.Fatalf("expected no selection status, got %q", next.status)
	}
}
