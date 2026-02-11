package tui

import (
	"testing"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestHandleEscapeFromHistoryInDockerHub(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.dockerHubActive = true
	m.focus = FocusHistory
	m.history = []registry.HistoryEntry{{CreatedBy: "RUN echo hi"}}
	m.hasSelectedTag = true
	m.selectedTag = registry.Tag{Name: "latest"}

	m.handleEscape()

	if m.focus != FocusDockerHubTags {
		t.Fatalf("expected focus to return to DockerHub tags, got %v", m.focus)
	}
	if m.hasSelectedTag {
		t.Fatalf("expected selected tag to be cleared")
	}
	if len(m.history) != 0 {
		t.Fatalf("expected history to be cleared")
	}
}

func TestHandleEscapeFromImagesWithProjects(t *testing.T) {
	auth := registry.Auth{Kind: "harbor"}
	auth.Harbor.Anonymous = true
	m := NewModel("https://harbor.example.com", auth, nil, false, nil, nil, "", "")
	m.focus = FocusImages
	m.hasSelectedProject = true
	m.selectedProject = "prod"

	m.handleEscape()

	if m.focus != FocusProjects {
		t.Fatalf("expected focus projects, got %v", m.focus)
	}
	if m.hasSelectedProject {
		t.Fatalf("expected selected project to be cleared")
	}
}
