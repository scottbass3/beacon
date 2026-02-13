# Beacon

[![Tests](https://github.com/scottbass3/beacon/actions/workflows/tests.yml/badge.svg)](https://github.com/scottbass3/beacon/actions/workflows/tests.yml)
[![Release](https://github.com/scottbass3/beacon/actions/workflows/release.yml/badge.svg)](https://github.com/scottbass3/beacon/actions/workflows/release.yml)
[![Deps Up-to-Date](https://img.shields.io/github/actions/workflow/status/scottbass3/beacon/dependencies.yml?branch=main&event=push&label=deps%20up-to-date)](https://github.com/scottbass3/beacon/actions/workflows/dependencies.yml)
[![Latest Release](https://img.shields.io/github/v/release/scottbass3/beacon?display_name=tag)](https://github.com/scottbass3/beacon/releases/latest)

Beacon is a terminal UI for exploring container image metadata across registries.

Current scope:
- Browse images, tags, and layer history for a selected registry context.
- Support registry providers: `registry_v2` and `harbor`.
- Support external tag search modes: Docker Hub and GitHub Container Registry (`ghcr.io`).
- Manage contexts from inside the UI (`:context`, `:context add`, `:context edit`, `:context remove`).

Not yet implemented in the UI:
- Tag rename/delete workflows, even though client interfaces already expose those methods.

## Quick start
Basic run :
```bash
go run ./cmd/beacon
```

Run with a direct registry URL (anonymous `registry_v2`):

```bash
go run ./cmd/beacon --registry https://registry.example.com
```

Run with a config file (contexts):

```bash
go run ./cmd/beacon --config ~/.config/beacon/config.json
```

Enable request logging:

```bash
go run ./cmd/beacon --debug --registry https://registry.example.com
```

## Configuration

Beacon reads JSON config from:
- `$XDG_CONFIG_HOME/beacon/config.json`
- fallback: `~/.config/beacon/config.json`

`--config` overrides the config path.

The config root can be either:
- an array of contexts, or
- an object with a `contexts` field.

Each context supports:
- `name`: display name
- `registry`: registry base URL
- `kind`: `registry_v2` or `harbor`
- `anonymous`: whether credentials are required
- `service`: optional auth service override

Example:

```json
[
  {
    "name": "prod",
    "registry": "https://registry.example.com",
    "kind": "registry_v2",
    "anonymous": false,
    "service": "registry.example.com"
  },
  {
    "name": "harbor",
    "registry": "https://harbor.example.com",
    "kind": "harbor",
    "anonymous": false,
    "service": "harbor-registry"
  }
]
```

Startup behavior:
- no contexts: opens context creation flow
- one context: auto-selects it
- multiple contexts: opens context selection modal
- `--registry`: skips context selection and uses that host directly

## Commands and navigation

In-app command mode (`:`):
- `:help`
- `:context`, `:context add`, `:context edit <name>`, `:context remove <name>`, `:context <name>`
- `:dockerhub [image]`
- `:github [owner/image]` (alias: `:ghcr`)

Core keys:
- `Enter`: drill down (projects/images -> tags -> history)
- `Esc`: go back one level
- `/`: filter current list
- `r`: refresh current view
- `c`: copy selected `image:tag` (when browsing tags)
- `p`: pull selected `image:tag` with Docker (when browsing tags)
- Mouse: click a row to select it, use scroll wheel to move up/down in tables
- `?` or `F1`: help

## Debug logging

Use `--debug` to stream request logs under the UI.

## Auth cache

Beacon stores cached auth metadata in:
- `$XDG_CACHE_HOME/beacon/auth.json`
- fallback: `~/.cache/beacon/auth.json`

For `registry_v2`, when `remember` is enabled, refresh token data is persisted there.

## Project layout

- `cmd/beacon/`: CLI entrypoint
- `internal/tui/`: Bubble Tea model, views, actions, modes, context/auth UX
- `internal/contextstore/`: context CRUD and persistence orchestration
- `internal/registry/`: registry providers, clients, auth flow, history resolution
  - `internal/registry/history.go`: shared manifest/config history structures

## Development

```bash
go build ./cmd/beacon
go test ./...
```
