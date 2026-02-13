package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func newMouseTestModel(t *testing.T) Model {
	t.Helper()
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.width = 120
	m.height = 40
	m.images = []registry.Image{
		{Name: "demo/a"},
		{Name: "demo/b"},
		{Name: "demo/c"},
		{Name: "demo/d"},
		{Name: "demo/e"},
		{Name: "demo/f"},
	}
	m.focus = FocusImages
	m.syncTable()
	return m
}

func TestMouseClickSelectsVisibleTableRow(t *testing.T) {
	m := newMouseTestModel(t)
	region, ok := m.tableMouseRowsRegion()
	if !ok {
		t.Fatalf("expected table mouse region")
	}

	targetRow := 3
	msg := tea.MouseMsg{
		X:      region.x + 1,
		Y:      region.y + targetRow,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonLeft,
	}
	updated, _ := m.Update(msg)
	next := updated.(Model)

	if next.table.Cursor() != targetRow {
		t.Fatalf("expected cursor at row %d, got %d", targetRow, next.table.Cursor())
	}
}

func TestMouseWheelMovesTableSelection(t *testing.T) {
	m := newMouseTestModel(t)
	region, ok := m.tableMouseRowsRegion()
	if !ok {
		t.Fatalf("expected table mouse region")
	}

	down := tea.MouseMsg{
		X:      region.x + 1,
		Y:      region.y,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	}
	updated, _ := m.Update(down)
	next := updated.(Model)
	if next.table.Cursor() != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", next.table.Cursor())
	}

	up := tea.MouseMsg{
		X:      region.x + 1,
		Y:      region.y,
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelUp,
	}
	updated, _ = next.Update(up)
	next = updated.(Model)
	if next.table.Cursor() != 0 {
		t.Fatalf("expected cursor to move up to 0, got %d", next.table.Cursor())
	}
}

func TestMouseWheelDownAtBottomRequestsExternalNextPage(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")
	m.width = 120
	m.height = 40
	m.dockerHubActive = true
	m.dockerHubInputFocus = true
	m.focus = FocusDockerHubTags
	m.dockerHubImage = "library/nginx"
	m.dockerHubNext = "https://hub.docker.com/v2/repositories/library/nginx/tags?page=2"
	m.dockerHubTags = []registry.Tag{
		{Name: "latest"},
		{Name: "alpine"},
		{Name: "1.27"},
	}
	m.syncTable()
	m.tableSetCursor(len(m.table.Rows()) - 1)

	region, ok := m.tableMouseRowsRegion()
	if !ok {
		t.Fatalf("expected table mouse region")
	}

	msg := tea.MouseMsg{
		X:      region.x + 1,
		Y:      region.y + minInt(1, region.height-1),
		Action: tea.MouseActionPress,
		Button: tea.MouseButtonWheelDown,
	}
	updated, cmd := m.Update(msg)
	next := updated.(Model)

	if cmd == nil {
		t.Fatalf("expected next page command when scrolling at bottom")
	}
	if next.dockerHubInputFocus {
		t.Fatalf("expected external search input to blur when table is scrolled")
	}
}
