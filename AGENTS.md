# Repository Guidelines

## Project Structure & Module Organization
- `cmd/beacon/`: entrypoint for the Bubble Tea TUI (`main.go`).
- `internal/tui/`: UI state, view rendering, and interaction logic.
- `internal/registry/`: domain types and registry-facing interfaces/helpers.
- `README.md`: usage notes and current status.
- Tests are not present yet; when added, keep them near the code (e.g., `internal/tui/model_test.go`).

## Build, Test, and Development Commands
- `go run ./cmd/beacon --registry https://registry.example.com`: run the TUI locally during development.
- `go run ./cmd/beacon --config ~/.config/beacon/config.json`: run using a config file with contexts.
- `go run ./cmd/beacon --debug --registry https://registry.example.com`: run with request logging.
- `go build ./cmd/beacon`: compile a local binary for quick manual testing.
- `go test ./...`: run all Go tests (useful once tests are added).
- `go mod tidy`: normalize module dependencies after adding/removing packages.

## Coding Style & Naming Conventions
- Go formatting is required: run `gofmt` on all `.go` files (use `gofmt -w .`).
- Package names: short, lowercase (e.g., `tui`, `registry`).
- Exported identifiers: `PascalCase` (e.g., `PullCommand`).
- Unexported identifiers: `camelCase`.
- Keep Bubble Tea `Model` methods (`Init`, `Update`, `View`) cohesive and side-effect light; push I/O to dedicated helpers or services.
- UI elements are done using Bubbletea, Bubbles and Lip Gloss libraries

## Testing Guidelines
- Testing framework: Go’s standard `testing` package.
- Naming: `*_test.go`, table-driven tests where practical.
- Add tests alongside the code they validate, especially for registry helpers and view logic.
- There is no coverage target yet; prioritize core registry behaviors and TUI actions as they are implemented.

## Commit & Pull Request Guidelines
- Git history currently contains a single “Initial commit”; no established convention exists yet.
- Suggested convention: short, imperative subject line (e.g., “Add registry client stub”), optional body for rationale.
- PRs should include: summary, testing notes (commands run), and screenshots for TUI changes when applicable.

## Configuration & Security
- Registry credentials are not wired yet; when added, prefer environment variables or a local config file excluded by `.gitignore`.
- Avoid committing tokens or credentials to the repository.
