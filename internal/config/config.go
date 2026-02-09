package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Contexts []Context `json:"contexts"`
}

type Context struct {
	Name      string `json:"name"`
	Registry  string `json:"registry"`
	Kind      string `json:"kind"`
	Anonymous bool   `json:"anonymous"`
	Service   string `json:"service"`
}

func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "beacon", "config.json")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".config", "beacon", "config.json")
	}
	return "config.json"
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config JSON: %w", err)
	}

	for i := range cfg.Contexts {
		cfg.Contexts[i].Name = strings.TrimSpace(cfg.Contexts[i].Name)
		cfg.Contexts[i].Registry = strings.TrimSpace(cfg.Contexts[i].Registry)
		cfg.Contexts[i].Kind = strings.TrimSpace(cfg.Contexts[i].Kind)
		cfg.Contexts[i].Service = strings.TrimSpace(cfg.Contexts[i].Service)
		if cfg.Contexts[i].Registry == "" {
			return Config{}, fmt.Errorf("context %d missing registry", i+1)
		}
		if cfg.Contexts[i].Kind == "" {
			return Config{}, fmt.Errorf("context %d missing kind", i+1)
		}
	}

	return cfg, nil
}

func (c *Config) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}

	switch trimmed[0] {
	case '[':
		var contexts []Context
		if err := json.Unmarshal(trimmed, &contexts); err != nil {
			return err
		}
		c.Contexts = contexts
		return nil
	case '{':
		var wrapper struct {
			Contexts []Context `json:"contexts"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			return err
		}
		c.Contexts = wrapper.Contexts
		return nil
	default:
		return fmt.Errorf("invalid config JSON: expected array at root")
	}
}
