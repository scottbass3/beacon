# Beacon

Beacon is a terminal UI for browsing remote Docker registries. This app allow to search remote on repositories and perform common simple like renaming or deleting tags.

## Quick start

```bash
go run ./cmd/beacon --registry https://registry.example.com
```

## Configuration

Beacon reads JSON configuration from `$XDG_CONFIG_HOME/beacon/config.json` (fallback: `~/.config/beacon/config.json`). You can override the path with `--config`.

If multiple contexts exist, you will be prompted to select one when starting the app. If only one context exists, it is used automatically. Passing `--registry` skips context selection and connects directly to that registry.

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
    "name": "staging",
    "registry": "https://staging-registry.example.com",
    "kind": "registry_v2",
    "anonymous": true
  },
  {
    "name": "harbor",
    "registry": "https://reg.cadoles.com",
    "kind": "harbor",
    "anonymous": false,
    "service": "harbor-registry"
  }
]
```

## Debug logging

Use `--debug` to show request logs and headers below the UI.

## Auth cache

Beacon stores the last-used username (and refresh token when `remember` is enabled for `registry_v2`) under `$XDG_CACHE_HOME/beacon/auth.json` (fallback: `~/.cache/beacon/auth.json`).
