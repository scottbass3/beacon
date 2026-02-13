package tui

import (
	"strings"
	"testing"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestCurrentPageHelpEntriesArePageScoped(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Model)
		wantIn     []string
		wantAbsent []string
	}{
		{
			name: "registry tags",
			setup: func(m *Model) {
				m.focus = FocusTags
			},
			wantIn: []string{
				"Open selected tag history",
				"Copy selected image:tag",
			},
			wantAbsent: []string{
				"Focus search input",
				"Exit external mode",
			},
		},
		{
			name: "external history",
			setup: func(m *Model) {
				m.dockerHubActive = true
				m.focus = FocusHistory
			},
			wantIn: []string{
				"Go back one level",
				"Focus search input",
			},
			wantAbsent: []string{
				"Copy selected image:tag",
				"Exit external mode",
			},
		},
		{
			name: "filter input",
			setup: func(m *Model) {
				m.filterActive = true
			},
			wantIn: []string{
				"Set filter text",
				"Apply and close filter input",
				"Open command input",
			},
			wantAbsent: []string{
				"Open help",
				"Refresh current data",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := testModelForShortcuts()
			tc.setup(&m)

			entries := m.currentPageHelpEntries()
			actions := map[string]bool{}
			for _, entry := range entries {
				actions[entry.Action] = true
			}

			for _, action := range tc.wantIn {
				if !actions[action] {
					t.Fatalf("expected help entry action %q", action)
				}
			}
			for _, action := range tc.wantAbsent {
				if actions[action] {
					t.Fatalf("did not expect help entry action %q", action)
				}
			}
		})
	}
}

func TestRenderShortcutHintLineUsesCurrentPage(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*Model)
		wantIn     []string
		wantAbsent []string
	}{
		{
			name: "registry tags",
			setup: func(m *Model) {
				m.focus = FocusTags
			},
			wantIn: []string{
				"Shortcuts:",
				"c copy",
				"esc back",
			},
			wantAbsent: []string{
				"s search",
			},
		},
		{
			name: "dockerhub tags",
			setup: func(m *Model) {
				m.dockerHubActive = true
				m.focus = FocusDockerHubTags
			},
			wantIn: []string{
				"s search",
				"c copy",
				"esc exit",
			},
		},
		{
			name: "filter input",
			setup: func(m *Model) {
				m.filterActive = true
			},
			wantIn: []string{
				"Filter:",
				"type text",
				": command",
			},
			wantAbsent: []string{
				"? help",
			},
		},
		{
			name: "help overlay",
			setup: func(m *Model) {
				m.helpActive = true
				m.focus = FocusTags
			},
			wantIn: []string{
				"Help:",
				"esc/? close",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := testModelForShortcuts()
			tc.setup(&m)

			line := m.renderShortcutHintLine()
			for _, part := range tc.wantIn {
				if !strings.Contains(line, part) {
					t.Fatalf("expected hint %q to contain %q", line, part)
				}
			}
			for _, part := range tc.wantAbsent {
				if strings.Contains(line, part) {
					t.Fatalf("did not expect hint %q to contain %q", line, part)
				}
			}
		})
	}
}

func TestHelpPageTitleIgnoresHelpOverlay(t *testing.T) {
	m := testModelForShortcuts()
	m.helpActive = true
	m.focus = FocusTags

	if got := m.helpPageTitle(); got != "Tags" {
		t.Fatalf("expected underlying page title Tags, got %q", got)
	}
}

func testModelForShortcuts() Model {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	return NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
}
