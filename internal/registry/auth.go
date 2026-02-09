package registry

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Auth struct {
	Kind       string
	RegistryV2 RegistryV2Auth
	Harbor     HarborAuth
}

type RegistryV2Auth struct {
	Anonymous    bool   `json:"anonymous"`
	TokenURL     string `json:"token_url"`
	Service      string `json:"service"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Remember     bool   `json:"remember"`
	RefreshToken string `json:"refresh_token"`
}

type HarborAuth struct {
	Anonymous bool   `json:"anonymous"`
	TokenURL  string `json:"token_url"`
	Service   string `json:"service"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Remember  bool   `json:"remember"`
}

func (a *Auth) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		a.Kind = "none"
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw) == 0 {
		a.Kind = "none"
		return nil
	}
	if len(raw) > 1 {
		return fmt.Errorf("auth must define a single registry auth block")
	}

	for key, payload := range raw {
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "registry_v2", "v2", "registry":
			a.Kind = "registry_v2"
			if err := json.Unmarshal(payload, &a.RegistryV2); err != nil {
				return fmt.Errorf("invalid registry_v2 auth: %w", err)
			}
		case "harbor":
			a.Kind = "harbor"
			if err := json.Unmarshal(payload, &a.Harbor); err != nil {
				return fmt.Errorf("invalid harbor auth: %w", err)
			}
		case "none", "anonymous":
			a.Kind = "none"
		default:
			return fmt.Errorf("unsupported auth method: %s", key)
		}
	}

	return nil
}

func (a *Auth) Normalize() {
	kind := strings.ToLower(strings.TrimSpace(a.Kind))
	if kind == "" {
		kind = "none"
	}
	switch kind {
	case "registry", "v2":
		kind = "registry_v2"
	case "anonymous":
		kind = "none"
	}
	a.Kind = kind
	a.RegistryV2.TokenURL = strings.TrimSpace(a.RegistryV2.TokenURL)
	a.RegistryV2.Service = strings.TrimSpace(a.RegistryV2.Service)
	a.RegistryV2.Username = strings.TrimSpace(a.RegistryV2.Username)
	a.RegistryV2.Password = strings.TrimSpace(a.RegistryV2.Password)
	a.RegistryV2.RefreshToken = strings.TrimSpace(a.RegistryV2.RefreshToken)
	a.Harbor.TokenURL = strings.TrimSpace(a.Harbor.TokenURL)
	a.Harbor.Service = strings.TrimSpace(a.Harbor.Service)
	a.Harbor.Username = strings.TrimSpace(a.Harbor.Username)
	a.Harbor.Password = strings.TrimSpace(a.Harbor.Password)
}

func (a Auth) Validate() error {
	switch a.Kind {
	case "none":
		return nil
	case "registry_v2":
		if a.RegistryV2.Anonymous {
			return nil
		}
		if a.RegistryV2.Username == "" {
			return fmt.Errorf("registry_v2 auth requires username")
		}
		if a.RegistryV2.Password == "" && !(a.RegistryV2.Remember && a.RegistryV2.RefreshToken != "") {
			return fmt.Errorf("registry_v2 auth requires password unless remember is set with a refresh_token")
		}
		return nil
	case "harbor":
		if a.Harbor.Anonymous {
			return nil
		}
		if a.Harbor.Username == "" || a.Harbor.Password == "" {
			return fmt.Errorf("harbor auth requires username and password")
		}
		return nil
	default:
		return fmt.Errorf("unsupported auth method: %s", a.Kind)
	}
}
