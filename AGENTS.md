# Repository Guidelines

## Project Structure & Module Organization
- `cmd/beacon/` — CLI entrypoint for the Bubble Tea application.
- `internal/tui/` — model state, views, shortcuts, and modal flows.
- `internal/registry/` — registry clients, providers, auth, and pull/history logic.
- `internal/contextstore/` — persisted registry contexts.
- `internal/config/` — config file loading.

Keep new tests next to the code they cover (e.g., `internal/tui/*_test.go`).

## Build, Test, and Development Commands
- `go run ./cmd/beacon`: start the TUI with default config discovery.
- `go run ./cmd/beacon --registry https://registry.example.com`: connect directly to a registry without a config file.
- `go run ./cmd/beacon --config ~/.config/beacon/config.json`: run against an explicit context file.
- `go run ./cmd/beacon --debug --registry https://registry.example.com`: enable request logging under the UI.
- `go build ./cmd/beacon`: compile the local binary.
- `go test ./...`: run the full Go test suite.
- `go mod tidy`: normalize module dependencies after package changes.

## Coding Style & Naming Conventions
Format all Go files with `gofmt -w .` before submitting. Use short lowercase package names, `PascalCase` for exported identifiers, and `camelCase` for internal helpers. Keep Bubble Tea `Init`, `Update`, and `View` methods focused on state transitions and rendering; push registry and persistence I/O into dedicated helpers or services. Follow existing Bubble Tea, Bubbles, and Lip Gloss patterns rather than introducing parallel UI conventions.

## Testing Guidelines
This repository uses Go’s standard `testing` package. Name files `*_test.go` and prefer table-driven tests for registry parsing, context persistence, and TUI state transitions. Run `go test ./...` before opening a PR. No fixed coverage target exists, but changes to navigation, auth flows, context editing, or registry clients should include regression tests.

## Commit & Pull Request Guidelines
Recent commits use short, imperative subjects such as `Simplify release workflow to build and upload binaries directly`. Match that style and keep the first line specific. Pull requests should include a brief summary, test notes with commands run, and screenshots or terminal captures for visible TUI changes.

## Configuration & Security
Do not commit registry credentials, tokens, or local config files. Prefer environment variables or local JSON config under `$XDG_CONFIG_HOME/beacon/config.json` (fallback: `~/.config/beacon/config.json`). Cached auth data may be written under `$XDG_CACHE_HOME/beacon/auth.json` (fallback: `~/.cache/beacon/auth.json`), so avoid using real production credentials in shared environments.
