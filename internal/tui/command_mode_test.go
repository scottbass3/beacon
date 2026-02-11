package tui

import (
	"testing"

	"github.com/scottbass3/beacon/internal/registry"
)

func TestParseCommand(t *testing.T) {
	name, args := parseCommand("github owner/image")
	if name != "github" {
		t.Fatalf("expected github, got %q", name)
	}
	if len(args) != 1 || args[0] != "owner/image" {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestResolveCommandAlias(t *testing.T) {
	descriptor, ok := resolveCommand("ghcr")
	if !ok {
		t.Fatalf("expected alias to resolve")
	}
	if descriptor.Name != "github" {
		t.Fatalf("expected github descriptor, got %q", descriptor.Name)
	}
}

func TestRunCommandHelpAndUnknown(t *testing.T) {
	auth := registry.Auth{Kind: "registry_v2"}
	auth.RegistryV2.Anonymous = true
	m := NewModel("https://registry.example.com", auth, nil, false, nil, nil, "", "")

	m.commandInput.SetValue("help")
	updated, _ := m.runCommand()
	next := updated.(Model)
	if !next.helpActive {
		t.Fatalf("expected help to be active after :help")
	}

	next.commandInput.SetValue("does-not-exist")
	updated, _ = next.runCommand()
	next = updated.(Model)
	if next.status == "" {
		t.Fatalf("expected status message for unknown command")
	}
}
