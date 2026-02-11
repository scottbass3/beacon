package tui

import (
	"testing"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestSwitchContextAt(t *testing.T) {
	authA := registry.Auth{Kind: "registry_v2"}
	authA.RegistryV2.Anonymous = true
	authB := registry.Auth{Kind: "harbor"}
	authB.Harbor.Anonymous = true

	contexts := []ContextOption{
		{Name: "prod", Host: "https://registry.example.com", Auth: authA},
		{Name: "harbor", Host: "https://harbor.example.com", Auth: authB},
	}

	m := NewModel("", registry.Auth{}, nil, false, nil, contexts, "prod", "/tmp/beacon-config.json")
	updated, cmd := m.switchContextAt(1)
	next := updated.(Model)

	if next.context != "harbor" {
		t.Fatalf("expected context harbor, got %q", next.context)
	}
	if next.registryHost != "https://harbor.example.com" {
		t.Fatalf("unexpected registry host: %s", next.registryHost)
	}
	if cmd == nil {
		t.Fatalf("expected init command after context switch")
	}
}
