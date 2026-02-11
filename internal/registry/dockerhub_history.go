package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/scottbass3/beacon/internal/registry/history"
)

const dockerHubRegistryBaseURL = "https://registry-1.docker.io"

func (c *DockerHubClient) ListTagHistory(ctx context.Context, image, tag string) ([]HistoryEntry, error) {
	image = strings.Trim(strings.TrimSpace(image), "/")
	tag = strings.TrimSpace(tag)
	if image == "" {
		return nil, fmt.Errorf("docker hub image is required")
	}
	if tag == "" {
		return nil, fmt.Errorf("docker hub tag is required")
	}
	return listTagHistoryFromManifest(ctx, "docker hub", image, tag, c.getRegistryManifest, c.getRegistryConfig)
}

func (c *DockerHubClient) getRegistryManifest(ctx context.Context, image, reference string) (history.ManifestV2, error) {
	endpoint := fmt.Sprintf("%s/v2/%s/manifests/%s", dockerHubRegistryBaseURL, image, url.PathEscape(reference))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return history.ManifestV2{}, err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))

	resp, err := c.doRegistryRequest(ctx, req, image)
	if err != nil {
		return history.ManifestV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return history.ManifestV2{}, fmt.Errorf("docker hub manifest request failed: %s", resp.Status)
	}

	var manifest history.ManifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return history.ManifestV2{}, err
	}
	return manifest, nil
}

func (c *DockerHubClient) getRegistryConfig(ctx context.Context, image, digest string) (history.ConfigV2, error) {
	endpoint := fmt.Sprintf("%s/v2/%s/blobs/%s", dockerHubRegistryBaseURL, image, url.PathEscape(digest))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return history.ConfigV2{}, err
	}

	resp, err := c.doRegistryRequest(ctx, req, image)
	if err != nil {
		return history.ConfigV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return history.ConfigV2{}, fmt.Errorf("docker hub config request failed: %s", resp.Status)
	}

	var cfg history.ConfigV2
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return history.ConfigV2{}, err
	}
	return cfg, nil
}

func (c *DockerHubClient) doRegistryRequest(ctx context.Context, req *http.Request, image string) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	challenge := resp.Header.Get("Www-Authenticate")
	resp.Body.Close()

	realm, service, scope, ok := parseBearerChallenge(challenge)
	if !ok {
		return nil, fmt.Errorf("docker hub registry requires bearer auth")
	}
	if service == "" {
		service = "registry.docker.io"
	}
	if scope == "" {
		scope = fmt.Sprintf("repository:%s:pull", image)
	}

	token, _, err := fetchBearerToken(ctx, c.httpClient, c.logger, realm, service, scope)
	if err != nil {
		return nil, err
	}

	retryReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	retryReq.Header = req.Header.Clone()
	retryReq.Header.Set("Authorization", "Bearer "+token)

	retryResp, retryErr := c.httpClient.Do(retryReq)
	c.logRequest(retryReq, retryResp)
	if retryErr != nil {
		return nil, retryErr
	}
	return retryResp, nil
}
