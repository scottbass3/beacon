package registry

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func ProviderForKind(kind string) Provider {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "harbor":
		return HarborProvider{}
	default:
		return RegistryV2Provider{}
	}
}

func ProviderForAuth(auth Auth) Provider {
	kind := strings.ToLower(strings.TrimSpace(auth.Kind))
	if kind == "" || kind == "none" || kind == "anonymous" {
		kind = "registry_v2"
	}
	return ProviderForKind(kind)
}

func NewClient(registryHost string, auth Auth) (Client, error) {
	return NewClientWithLogger(registryHost, auth, nil)
}

func NewClientWithLogger(registryHost string, auth Auth, logger RequestLogger) (Client, error) {
	trimmed := strings.TrimSpace(registryHost)
	if trimmed == "" {
		return nil, errors.New("registry host is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "https://" + trimmed
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid registry host: %w", err)
	}
	if parsed.Host == "" {
		return nil, errors.New("registry host must include a host name")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/")

	auth.Normalize()
	provider := ProviderForAuth(auth)
	if auth.Kind == "none" {
		auth.Kind = provider.Kind()
		if auth.Kind == "registry_v2" {
			auth.RegistryV2.Anonymous = true
		}
	}

	ApplyAuthCache(&auth, parsed.Host)
	if err := provider.PrepareAuth(parsed, &auth); err != nil {
		return nil, err
	}
	if err := auth.Validate(); err != nil {
		return nil, err
	}

	return provider.NewClient(parsed, auth, logger)
}
