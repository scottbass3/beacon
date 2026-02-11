package contextstore

import (
	"fmt"
	"strings"

	"github.com/scottbass3/beacon/internal/registry"
)

// Service contains pure context CRUD and validation logic.
type Service struct {
	store Store
}

func NewService(path string) Service {
	return Service{store: New(path)}
}

func (s Service) Save(contexts []Context) error {
	return s.store.Save(contexts)
}

func (s Service) Add(existing []Context, candidate Context) ([]Context, int, error) {
	normalized, err := normalizeContext(candidate)
	if err != nil {
		return nil, -1, err
	}
	if err := ensureUniqueName(existing, normalized.Name, -1); err != nil {
		return nil, -1, err
	}
	updated := append(append([]Context{}, existing...), normalized)
	return updated, len(updated) - 1, nil
}

func (s Service) Edit(existing []Context, index int, candidate Context) ([]Context, error) {
	if index < 0 || index >= len(existing) {
		return nil, fmt.Errorf("invalid context selection")
	}
	normalized, err := normalizeContext(candidate)
	if err != nil {
		return nil, err
	}
	if err := ensureUniqueName(existing, normalized.Name, index); err != nil {
		return nil, err
	}
	updated := append([]Context{}, existing...)
	updated[index] = normalized
	return updated, nil
}

func (s Service) RemoveByName(existing []Context, name string) ([]Context, Context, int, error) {
	index, ok := ResolveByName(existing, name)
	if !ok {
		return nil, Context{}, -1, fmt.Errorf("unknown context: %s", strings.TrimSpace(name))
	}
	removed := existing[index]
	updated := make([]Context, 0, len(existing)-1)
	updated = append(updated, existing[:index]...)
	updated = append(updated, existing[index+1:]...)
	return updated, removed, index, nil
}

func ResolveByName(contexts []Context, name string) (int, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return 0, false
	}
	for i, ctx := range contexts {
		if strings.EqualFold(strings.TrimSpace(ctx.Name), trimmed) {
			return i, true
		}
	}
	for i, ctx := range contexts {
		if strings.EqualFold(strings.TrimSpace(ctx.Host), trimmed) {
			return i, true
		}
	}
	return 0, false
}

func NormalizeKindInput(input string) (string, bool) {
	kind := strings.ToLower(strings.TrimSpace(input))
	switch kind {
	case "registry", "v2", "registry_v2":
		return "registry_v2", true
	case "harbor":
		return "harbor", true
	default:
		return "", false
	}
}

func normalizeContext(candidate Context) (Context, error) {
	name := strings.TrimSpace(candidate.Name)
	host := strings.TrimSpace(candidate.Host)
	if name == "" {
		return Context{}, fmt.Errorf("context name is required")
	}
	if host == "" {
		return Context{}, fmt.Errorf("registry is required")
	}
	kind, ok := NormalizeKindInput(candidate.Auth.Kind)
	if !ok {
		return Context{}, fmt.Errorf("kind must be registry_v2 or harbor")
	}
	auth := registry.Auth{Kind: kind}
	switch kind {
	case "harbor":
		auth.Harbor.Anonymous = candidate.Auth.Harbor.Anonymous
		auth.Harbor.Service = strings.TrimSpace(candidate.Auth.Harbor.Service)
	default:
		auth.RegistryV2.Anonymous = candidate.Auth.RegistryV2.Anonymous
		auth.RegistryV2.Service = strings.TrimSpace(candidate.Auth.RegistryV2.Service)
	}
	auth.Normalize()
	return Context{Name: name, Host: host, Auth: auth}, nil
}

func ensureUniqueName(existing []Context, name string, skip int) error {
	needle := strings.ToLower(strings.TrimSpace(name))
	for i, ctx := range existing {
		if i == skip {
			continue
		}
		if strings.ToLower(strings.TrimSpace(ctx.Name)) == needle {
			return fmt.Errorf("context %q already exists", name)
		}
	}
	return nil
}
