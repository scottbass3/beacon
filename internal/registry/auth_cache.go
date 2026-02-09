package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type authCacheEntry struct {
	Username     string `json:"username,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func ApplyAuthCache(auth *Auth, host string) {
	if auth == nil || auth.Kind == "" || auth.Kind == "none" || host == "" {
		return
	}
	entries, err := loadAuthCache()
	if err != nil {
		return
	}
	entry, ok := entries[cacheKey(host, auth.Kind)]
	if !ok {
		return
	}

	switch auth.Kind {
	case "registry_v2":
		if auth.RegistryV2.Username == "" && entry.Username != "" {
			auth.RegistryV2.Username = entry.Username
		}
		if auth.RegistryV2.Remember && auth.RegistryV2.RefreshToken == "" && entry.RefreshToken != "" {
			auth.RegistryV2.RefreshToken = entry.RefreshToken
		}
	case "harbor":
		if auth.Harbor.Username == "" && entry.Username != "" {
			auth.Harbor.Username = entry.Username
		}
	}
}

func PersistAuthCache(host string, auth Auth) {
	if auth.Kind == "" || auth.Kind == "none" || host == "" {
		return
	}
	entries, err := loadAuthCache()
	if err != nil {
		return
	}

	key := cacheKey(host, auth.Kind)
	entry := entries[key]
	switch auth.Kind {
	case "registry_v2":
		if auth.RegistryV2.Username != "" {
			entry.Username = auth.RegistryV2.Username
		}
		if auth.RegistryV2.Remember {
			if auth.RegistryV2.RefreshToken != "" {
				entry.RefreshToken = auth.RegistryV2.RefreshToken
			}
		} else {
			entry.RefreshToken = ""
		}
	case "harbor":
		if auth.Harbor.Username != "" {
			entry.Username = auth.Harbor.Username
		}
		entry.RefreshToken = ""
	default:
		return
	}

	if entry.Username == "" && entry.RefreshToken == "" {
		delete(entries, key)
	} else {
		entries[key] = entry
	}
	_ = saveAuthCache(entries)
}

func cacheKey(host, kind string) string {
	return strings.ToLower(host) + "|" + strings.ToLower(kind)
}

func authCachePath() string {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" {
		return filepath.Join(xdg, "beacon", "auth.json")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cache", "beacon", "auth.json")
	}
	return "auth.json"
}

func loadAuthCache() (map[string]authCacheEntry, error) {
	path := authCachePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]authCacheEntry{}, nil
		}
		return nil, err
	}
	var entries map[string]authCacheEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		entries = map[string]authCacheEntry{}
	}
	return entries, nil
}

func saveAuthCache(entries map[string]authCacheEntry) error {
	path := authCachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
