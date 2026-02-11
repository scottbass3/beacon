package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
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

	manifest, err := c.getRegistryManifest(ctx, image, tag)
	if err != nil {
		return nil, err
	}
	if manifest.Config.Digest == "" {
		resolvedDigest := preferredManifestDigest(manifest)
		if resolvedDigest != "" {
			manifest, err = c.getRegistryManifest(ctx, image, resolvedDigest)
			if err != nil {
				return nil, err
			}
		}
	}
	if manifest.Config.Digest == "" {
		return nil, fmt.Errorf("docker hub config digest missing for %s:%s", image, tag)
	}
	cfg, err := c.getRegistryConfig(ctx, image, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}
	return buildHistory(manifest, cfg), nil
}

func (c *DockerHubClient) getRegistryManifest(ctx context.Context, image, reference string) (manifestV2, error) {
	endpoint := fmt.Sprintf("%s/v2/%s/manifests/%s", dockerHubRegistryBaseURL, image, url.PathEscape(reference))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return manifestV2{}, err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.index.v1+json",
	}, ", "))

	resp, err := c.doRegistryRequest(ctx, req, image)
	if err != nil {
		return manifestV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return manifestV2{}, fmt.Errorf("docker hub manifest request failed: %s", resp.Status)
	}

	var manifest manifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return manifestV2{}, err
	}
	return manifest, nil
}

func (c *DockerHubClient) getRegistryConfig(ctx context.Context, image, digest string) (configV2, error) {
	endpoint := fmt.Sprintf("%s/v2/%s/blobs/%s", dockerHubRegistryBaseURL, image, url.PathEscape(digest))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return configV2{}, err
	}

	resp, err := c.doRegistryRequest(ctx, req, image)
	if err != nil {
		return configV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return configV2{}, fmt.Errorf("docker hub config request failed: %s", resp.Status)
	}

	var cfg configV2
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return configV2{}, err
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

	token, err := fetchBearerToken(ctx, c.httpClient, c.logger, realm, service, scope)
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

func fetchBearerToken(ctx context.Context, client *http.Client, logger RequestLogger, realm, service, scope string) (string, error) {
	tokenURL, err := url.Parse(realm)
	if err != nil {
		return "", fmt.Errorf("invalid token realm: %w", err)
	}
	query := tokenURL.Query()
	if service != "" {
		query.Set("service", service)
	}
	if scope != "" {
		query.Set("scope", scope)
	}
	tokenURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	logRequestWithLogger(logger, req, resp)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("token request failed: %s", resp.Status)
	}

	token, _, _, err := decodeTokenResponse(resp)
	if err != nil {
		return "", err
	}
	if token == "" {
		return "", fmt.Errorf("token response missing token")
	}
	return token, nil
}

func logRequestWithLogger(logger RequestLogger, req *http.Request, resp *http.Response) {
	if logger == nil {
		return
	}
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	logger(RequestLog{
		Method:  req.Method,
		URL:     req.URL.String(),
		Headers: cloneHeader(req.Header),
		Status:  status,
	})
}
