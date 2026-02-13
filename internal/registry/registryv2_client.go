package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultCatalogPageSize = 1000

// HTTPClient implements the Docker Registry HTTP API v2.
type HTTPClient struct {
	baseURL        *url.URL
	httpClient     *http.Client
	auth           Auth
	logger         RequestLogger
	tokenMu        sync.Mutex
	registryToken  string
	registryExpiry time.Time
}

func newRegistryV2Client(baseURL *url.URL, auth Auth, logger RequestLogger) *HTTPClient {
	return &HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		auth:   auth,
		logger: logger,
	}
}

func (c *HTTPClient) ListImages(ctx context.Context) ([]Image, error) {
	repos, err := c.listRepositories(ctx)
	if err != nil {
		return nil, err
	}

	images := make([]Image, 0, len(repos))
	for _, repo := range repos {
		images = append(images, Image{
			Name:       repo,
			Repository: repo,
			TagCount:   -1,
			PullCount:  -1,
		})
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Name < images[j].Name
	})

	return images, nil
}

func (c *HTTPClient) ListTags(ctx context.Context, image string) ([]Tag, error) {
	return c.listTags(ctx, image)
}

func (c *HTTPClient) ListTagHistory(ctx context.Context, image, tag string) ([]HistoryEntry, error) {
	image = strings.TrimSpace(image)
	tag = strings.TrimSpace(tag)
	if image == "" || tag == "" {
		return nil, nil
	}
	return listTagHistoryFromManifest(ctx, "registry", image, tag, c.getManifest, c.getConfig)
}

func (c *HTTPClient) DeleteTag(ctx context.Context, image, tag string) error {
	return ErrNotSupported
}

func (c *HTTPClient) RenameTag(ctx context.Context, image, from, to string) error {
	return ErrNotSupported
}

func (c *HTTPClient) listRepositories(ctx context.Context) ([]string, error) {
	endpoint := c.resolve("/v2/_catalog", url.Values{
		"n": []string{fmt.Sprintf("%d", defaultCatalogPageSize)},
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if err := c.applyAuth(ctx, req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("catalog request failed: %s", resp.Status)
	}

	var payload struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	sort.Strings(payload.Repositories)
	return payload.Repositories, nil
}

func (c *HTTPClient) listTags(ctx context.Context, repository string) ([]Tag, error) {
	endpoint := c.resolve("/v2/"+repository+"/tags/list", nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	if err := c.applyAuth(ctx, req); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("tags request failed: %s", resp.Status)
	}

	var payload struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if len(payload.Tags) == 0 {
		return nil, nil
	}

	tags := make([]Tag, 0, len(payload.Tags))
	for _, name := range payload.Tags {
		tags = append(tags, Tag{Name: name, SizeBytes: -1})
	}
	return tags, nil
}

func (c *HTTPClient) getManifest(ctx context.Context, image, reference string) (ManifestV2, error) {
	endpoint := c.resolve("/v2/"+image+"/manifests/"+reference, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ManifestV2{}, err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))
	if err := c.applyAuth(ctx, req); err != nil {
		return ManifestV2{}, err
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return ManifestV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return ManifestV2{}, fmt.Errorf("manifest request failed: %s", resp.Status)
	}

	var manifest ManifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return ManifestV2{}, err
	}
	return manifest, nil
}

func (c *HTTPClient) getConfig(ctx context.Context, image, digest string) (ConfigV2, error) {
	endpoint := c.resolve("/v2/"+image+"/blobs/"+digest, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ConfigV2{}, err
	}
	if err := c.applyAuth(ctx, req); err != nil {
		return ConfigV2{}, err
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return ConfigV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return ConfigV2{}, fmt.Errorf("config request failed: %s", resp.Status)
	}

	var cfg ConfigV2
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return ConfigV2{}, err
	}
	return cfg, nil
}

func (c *HTTPClient) resolve(path string, query url.Values) string {
	return resolveURL(c.baseURL, path, query)
}

func (c *HTTPClient) applyAuth(ctx context.Context, req *http.Request) error {
	switch c.auth.Kind {
	case "registry_v2":
		if c.auth.RegistryV2.Anonymous {
			return nil
		}
		token, err := c.getRegistryV2Token(ctx)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return nil
}

func (c *HTTPClient) logRequest(req *http.Request, resp *http.Response) {
	if c.logger == nil {
		return
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	c.logger(RequestLog{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: cloneHeader(req.Header),
		Status:  status,
	})
}

func (c *HTTPClient) getRegistryV2Token(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	if c.registryToken != "" && time.Until(c.registryExpiry) > 30*time.Second {
		token := c.registryToken
		c.tokenMu.Unlock()
		return token, nil
	}
	c.tokenMu.Unlock()

	token, expiry, refresh, err := c.fetchRegistryV2Token(ctx)
	if err != nil {
		return "", err
	}

	c.tokenMu.Lock()
	c.registryToken = token
	c.registryExpiry = expiry
	if refresh != "" {
		c.auth.RegistryV2.RefreshToken = refresh
	}
	c.tokenMu.Unlock()
	PersistAuthCache(c.baseURL.Host, c.auth)

	return token, nil
}

func (c *HTTPClient) fetchRegistryV2Token(ctx context.Context) (string, time.Time, string, error) {
	auth := c.auth.RegistryV2
	form := url.Values{}
	scope := registryScope()
	form.Set("scope", scope)
	if auth.Service != "" {
		form.Set("service", auth.Service)
	} else if c.baseURL != nil && c.baseURL.Host != "" {
		form.Set("service", c.baseURL.Host)
	}
	if auth.Username != "" {
		form.Set("client_id", auth.Username)
	}

	grantType := "password"
	if auth.Remember && auth.RefreshToken != "" && auth.Password == "" {
		grantType = "refresh_token"
	}
	form.Set("grant_type", grantType)

	if grantType == "password" {
		form.Set("username", auth.Username)
		form.Set("password", auth.Password)
	} else {
		form.Set("refresh_token", auth.RefreshToken)
	}

	tokenURL := auth.TokenURL
	if tokenURL == "" {
		tokenURL = c.resolve("/token", nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", time.Time{}, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return "", time.Time{}, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", time.Time{}, "", fmt.Errorf("registry_v2 token request failed: %s", resp.Status)
	}

	token, refresh, expiry, err := decodeTokenResponse(resp)
	if err != nil {
		return "", time.Time{}, "", err
	}
	if token == "" {
		return "", time.Time{}, "", errors.New("registry_v2 token response missing token")
	}
	return token, expiry, refresh, nil
}

func registryScope() string {
	return "registry:catalog:*"
}
