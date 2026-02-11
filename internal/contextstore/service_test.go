package contextstore

import (
	"testing"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestServiceAddEditRemove(t *testing.T) {
	svc := NewService("/tmp/beacon-contextstore-test.json")

	addAuth := registry.Auth{Kind: "registry_v2"}
	addAuth.RegistryV2.Anonymous = true

	contexts, index, err := svc.Add(nil, Context{
		Name: "prod",
		Host: "https://registry.example.com",
		Auth: addAuth,
	})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if index != 0 {
		t.Fatalf("unexpected index: %d", index)
	}
	if len(contexts) != 1 {
		t.Fatalf("expected 1 context, got %d", len(contexts))
	}

	_, _, err = svc.Add(contexts, Context{
		Name: "PROD",
		Host: "https://another.example.com",
		Auth: addAuth,
	})
	if err == nil {
		t.Fatalf("expected duplicate add to fail")
	}

	editAuth := registry.Auth{Kind: "harbor"}
	editAuth.Harbor.Anonymous = false
	editAuth.Harbor.Service = "harbor-registry"

	contexts, err = svc.Edit(contexts, 0, Context{
		Name: "prod",
		Host: "https://harbor.example.com",
		Auth: editAuth,
	})
	if err != nil {
		t.Fatalf("edit failed: %v", err)
	}
	if contexts[0].Host != "https://harbor.example.com" {
		t.Fatalf("unexpected host after edit: %s", contexts[0].Host)
	}
	if contexts[0].Auth.Kind != "harbor" {
		t.Fatalf("unexpected kind after edit: %s", contexts[0].Auth.Kind)
	}

	contexts, removed, removedIndex, err := svc.RemoveByName(contexts, "prod")
	if err != nil {
		t.Fatalf("remove failed: %v", err)
	}
	if removedIndex != 0 {
		t.Fatalf("unexpected removed index: %d", removedIndex)
	}
	if removed.Name != "prod" {
		t.Fatalf("unexpected removed context: %+v", removed)
	}
	if len(contexts) != 0 {
		t.Fatalf("expected empty contexts after remove, got %d", len(contexts))
	}
}

func TestResolveByName(t *testing.T) {
	contexts := []Context{
		{Name: "prod", Host: "https://registry.example.com"},
		{Name: "staging", Host: "https://staging.example.com"},
	}

	index, ok := ResolveByName(contexts, "PROD")
	if !ok || index != 0 {
		t.Fatalf("expected to resolve by name, got ok=%v index=%d", ok, index)
	}

	index, ok = ResolveByName(contexts, "https://staging.example.com")
	if !ok || index != 1 {
		t.Fatalf("expected to resolve by host, got ok=%v index=%d", ok, index)
	}
}
