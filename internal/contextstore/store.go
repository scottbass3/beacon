package contextstore

import (
	"strings"

	"github.com/scottbass3/beacon/internal/config"
	"github.com/scottbass3/beacon/internal/registry"
)

// Context is the app-level context configuration persisted to disk.
type Context struct {
	Name string
	Host string
	Auth registry.Auth
}

// Store persists registry contexts in the Beacon config file.
type Store struct {
	path string
}

func DefaultPath() string {
	return config.DefaultPath()
}

func New(path string) Store {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = config.DefaultPath()
	}
	return Store{path: trimmed}
}

func (s Store) Path() string {
	return s.path
}

func (s Store) Ensure() ([]Context, error) {
	cfg, err := config.Ensure(s.path)
	if err != nil {
		return nil, err
	}
	return contextsFromConfig(cfg.Contexts), nil
}

func (s Store) Save(contexts []Context) error {
	cfg := config.Config{Contexts: make([]config.Context, 0, len(contexts))}
	for _, ctx := range contexts {
		cfg.Contexts = append(cfg.Contexts, toConfigContext(ctx))
	}
	return config.Save(s.path, cfg)
}

func contextsFromConfig(configContexts []config.Context) []Context {
	if len(configContexts) == 0 {
		return nil
	}
	out := make([]Context, 0, len(configContexts))
	for _, ctx := range configContexts {
		out = append(out, fromConfigContext(ctx))
	}
	return out
}

func fromConfigContext(ctx config.Context) Context {
	kind := normalizeKind(ctx.Kind)
	auth := registry.Auth{Kind: kind}
	switch kind {
	case "harbor":
		auth.Harbor.Anonymous = ctx.Anonymous
		auth.Harbor.Service = strings.TrimSpace(ctx.Service)
	default:
		auth.RegistryV2.Anonymous = ctx.Anonymous
		auth.RegistryV2.Service = strings.TrimSpace(ctx.Service)
	}
	auth.Normalize()
	return Context{
		Name: strings.TrimSpace(ctx.Name),
		Host: strings.TrimSpace(ctx.Registry),
		Auth: auth,
	}
}

func toConfigContext(ctx Context) config.Context {
	kind := normalizeKind(ctx.Auth.Kind)
	out := config.Context{
		Name:     strings.TrimSpace(ctx.Name),
		Registry: strings.TrimSpace(ctx.Host),
		Kind:     kind,
	}
	switch kind {
	case "harbor":
		out.Anonymous = ctx.Auth.Harbor.Anonymous
		out.Service = strings.TrimSpace(ctx.Auth.Harbor.Service)
	default:
		out.Anonymous = ctx.Auth.RegistryV2.Anonymous
		out.Service = strings.TrimSpace(ctx.Auth.RegistryV2.Service)
	}
	return out
}

func normalizeKind(value string) string {
	kind := strings.ToLower(strings.TrimSpace(value))
	switch kind {
	case "harbor":
		return "harbor"
	case "registry", "v2", "registry_v2":
		return "registry_v2"
	default:
		return "registry_v2"
	}
}
