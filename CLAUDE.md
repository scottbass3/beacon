# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go run ./cmd/beacon                                          # run with default config discovery
go run ./cmd/beacon --registry https://registry.example.com # anonymous registry_v2, no config needed
go run ./cmd/beacon --config ~/.config/beacon/config.json   # explicit context file
go run ./cmd/beacon --debug --registry https://registry.example.com # enable request logging
go build ./cmd/beacon                                        # compile binary
go test ./...                                                # run full test suite
go mod tidy                                                  # normalize dependencies
gofmt -w .                                                   # format (required before PR)
```

See `AGENTS.md` for commit style, PR guidelines, and security rules.

## Architecture

Beacon is a Bubble Tea TUI for browsing container image metadata (images → tags → layer history) across multiple registry types.

### Component Map

| Package | Role |
|---|---|
| `cmd/beacon/main.go` | CLI flags, resolves startup registry context, wires up Bubble Tea |
| `internal/tui/` | All UI: model state, views, modals, keyboard/mouse handlers |
| `internal/registry/` | Registry clients, provider pattern, auth, manifest/history parsing |
| `internal/contextstore/` | CRUD for persisted registry contexts, delegates I/O to config |
| `internal/config/` | JSON load/save with XDG path resolution |

### Registry Provider Pattern

`internal/registry/provider.go` defines a `Provider` interface that abstracts registry-specific behavior:
- `Kind()` — identifies the registry type (`registry_v2`, `harbor`, `dockerhub`, `github`)
- `TableSpec()` — declares which columns/capabilities the UI exposes for this type
- `AuthUI()` / `NeedsAuthPrompt()` — drives the auth modal fields shown to the user

Concrete providers: `RegistryV2Client`, `HarborClient`, `DockerHubClient`, `GitHubClient`. The factory in `provider.go` selects the right one from the context's `kind` field.

The `Client` interface defines core operations (`ListImages`, `ListTags`, `ListTagHistory`, `DeleteTag`, `RenameTag`). `ProjectClient` is an optional extension for registries with project namespacing (Harbor).

### TUI State Machine

The single `Model` struct in `internal/tui/` drives all state. Navigation uses a `Focus` enum (`Projects → Images → Tags → History`). Modals (context selection, context form, auth prompt, confirmation) are boolean flags overlaid on the main view.

File naming conventions inside `internal/tui/`:
- `actions_*.go` — async Bubble Tea commands that fetch data and return typed messages
- `*_mode*.go` — input handling for exclusive overlay modes (filter, command, auth)
- `input_handlers.go` / `update_handlers.go` — core key/message dispatch
- `views_*.go` — rendering functions

Async operations (registry fetches, persistence) are dispatched as Bubble Tea `tea.Cmd` and return typed message structs (e.g., `imagesMsg`, `tagsMsg`, `historyMsg`).

### Auth & Config Persistence

- Config: `$XDG_CONFIG_HOME/beacon/config.json` (fallback: `~/.config/beacon/config.json`)
- Auth cache: `$XDG_CACHE_HOME/beacon/auth.json` (fallback: `~/.cache/beacon/auth.json`)
- `registry_v2` persists refresh tokens; Harbor does not cache tokens
- Auth caching prevents re-prompting on startup for remembered sessions

### Startup Flow

1. `main.go` resolves the registry context (from config file or `--registry` flag)
2. TUI initializes: no contexts → creation flow; one → auto-select; many → selection modal
3. If auth is needed and not cached, the auth modal is shown before the main view
4. Navigation: Projects (Harbor only) → Images → Tags → History

## Testing

Use Go's standard `testing` package with table-driven tests. Changes to navigation, auth flows, context editing, or registry clients should include regression tests. No mocking framework — use direct structs and interfaces.
