package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const githubContainerBaseURL = "https://ghcr.io"

type GitHubContainerClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     RequestLogger

	tokenMu     sync.Mutex
	token       string
	tokenExpiry time.Time
}

type GitHubContainerTagsPage struct {
	Image string
	Tags  []Tag
	Next  string
}

func NewGitHubContainerClient(logger RequestLogger) *GitHubContainerClient {
	parsed, _ := url.Parse(githubContainerBaseURL)
	return &GitHubContainerClient{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

func (c *GitHubContainerClient) SearchTagsPage(ctx context.Context, input string) (GitHubContainerTagsPage, error) {
	image, err := normalizeGitHubContainerInput(input)
	if err != nil {
		return GitHubContainerTagsPage{}, err
	}
	return c.listTagsPage(ctx, image, "")
}

func (c *GitHubContainerClient) NextTagsPage(ctx context.Context, image, next string) (GitHubContainerTagsPage, error) {
	image = strings.TrimSpace(image)
	next = strings.TrimSpace(next)
	if image == "" {
		return GitHubContainerTagsPage{}, errors.New("github container image is required")
	}
	if next == "" {
		return GitHubContainerTagsPage{}, errors.New("github container next page URL is required")
	}
	return c.listTagsPage(ctx, image, next)
}

func (c *GitHubContainerClient) listTagsPage(ctx context.Context, image, next string) (GitHubContainerTagsPage, error) {
	image = strings.Trim(strings.TrimSpace(image), "/")
	if image == "" {
		return GitHubContainerTagsPage{}, errors.New("github container image is required")
	}

	endpoint := strings.TrimSpace(next)
	if endpoint == "" {
		query := url.Values{}
		query.Set("n", "100")
		endpoint = c.resolve(fmt.Sprintf("/v2/%s/tags/list", image), query)
	} else {
		endpoint = c.resolveNext(endpoint)
	}

	var payload githubContainerTagsResponse
	headers, err := c.doJSON(ctx, endpoint, image, &payload)
	if err != nil {
		return GitHubContainerTagsPage{}, err
	}

	tags := make([]Tag, 0, len(payload.Tags))
	for _, tagName := range payload.Tags {
		tags = append(tags, Tag{Name: tagName})
	}

	resolvedImage := strings.TrimSpace(payload.Name)
	if resolvedImage == "" {
		resolvedImage = image
	}

	return GitHubContainerTagsPage{
		Image: resolvedImage,
		Tags:  tags,
		Next:  parseGitHubContainerNext(headers.Get("Link"), c.baseURL),
	}, nil
}

func (c *GitHubContainerClient) ListTagHistory(ctx context.Context, image, tag string) ([]HistoryEntry, error) {
	image = strings.Trim(strings.TrimSpace(image), "/")
	tag = strings.TrimSpace(tag)
	if image == "" {
		return nil, errors.New("github container image is required")
	}
	if tag == "" {
		return nil, errors.New("github container tag is required")
	}
	return listTagHistoryFromManifest(ctx, "github", image, tag, c.getManifest, c.getConfig)
}

func (c *GitHubContainerClient) doJSON(ctx context.Context, endpoint, image string, out interface{}) (http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.doWithAuth(ctx, req, image)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return resp.Header.Clone(), fmt.Errorf("no GitHub Container Registry repository found for %q", image)
	}
	if resp.StatusCode >= 300 {
		return resp.Header.Clone(), fmt.Errorf("github container registry request failed: %s", resp.Status)
	}

	if out == nil {
		return resp.Header.Clone(), nil
	}
	return resp.Header.Clone(), json.NewDecoder(resp.Body).Decode(out)
}

func (c *GitHubContainerClient) doWithAuth(ctx context.Context, req *http.Request, image string) (*http.Response, error) {
	if token := c.cachedToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
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
		return nil, errors.New("github container registry requires bearer auth")
	}
	if service == "" && c.baseURL != nil {
		service = c.baseURL.Host
	}
	if scope == "" {
		scope = fmt.Sprintf("repository:%s:pull", strings.Trim(image, "/"))
	}

	token, expiry, err := c.fetchToken(ctx, realm, service, scope)
	if err != nil {
		return nil, err
	}
	c.cacheToken(token, expiry)

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

func (c *GitHubContainerClient) getManifest(ctx context.Context, image, reference string) (ManifestV2, error) {
	endpoint := c.resolve("/v2/"+image+"/manifests/"+url.PathEscape(reference), nil)
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

	resp, err := c.doWithAuth(ctx, req, image)
	if err != nil {
		return ManifestV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return ManifestV2{}, fmt.Errorf("github manifest request failed: %s", resp.Status)
	}

	var manifest ManifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return ManifestV2{}, err
	}
	return manifest, nil
}

func (c *GitHubContainerClient) getConfig(ctx context.Context, image, digest string) (ConfigV2, error) {
	endpoint := c.resolve("/v2/"+image+"/blobs/"+url.PathEscape(digest), nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ConfigV2{}, err
	}

	resp, err := c.doWithAuth(ctx, req, image)
	if err != nil {
		return ConfigV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return ConfigV2{}, fmt.Errorf("github config request failed: %s", resp.Status)
	}

	var cfg ConfigV2
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return ConfigV2{}, err
	}
	return cfg, nil
}

func (c *GitHubContainerClient) fetchToken(ctx context.Context, realm, service, scope string) (string, time.Time, error) {
	token, expiry, err := fetchBearerToken(ctx, c.httpClient, c.logger, realm, service, scope)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiry, nil
}

func (c *GitHubContainerClient) cachedToken() string {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.token == "" {
		return ""
	}
	if time.Until(c.tokenExpiry) <= 30*time.Second {
		c.token = ""
		c.tokenExpiry = time.Time{}
		return ""
	}
	return c.token
}

func (c *GitHubContainerClient) cacheToken(token string, expiry time.Time) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	c.token = token
	c.tokenExpiry = expiry
}

func (c *GitHubContainerClient) resolve(p string, query url.Values) string {
	return resolveURL(c.baseURL, p, query)
}

func (c *GitHubContainerClient) resolveNext(next string) string {
	return resolveNextURL(c.baseURL, next)
}

func (c *GitHubContainerClient) logRequest(req *http.Request, resp *http.Response) {
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

type githubContainerTagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func normalizeGitHubContainerInput(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", errors.New("github search requires an image name (owner/image)")
	}
	trimmed = strings.TrimPrefix(trimmed, "ghcr.io/")
	trimmed = strings.TrimPrefix(trimmed, "https://ghcr.io/")
	trimmed = strings.TrimPrefix(trimmed, "http://ghcr.io/")

	if strings.HasPrefix(trimmed, "https://") || strings.HasPrefix(trimmed, "http://") {
		parsed, err := url.Parse(trimmed)
		if err == nil {
			trimmed = strings.TrimPrefix(parsed.Path, "/")
		}
	}
	if at := strings.Index(trimmed, "@"); at != -1 {
		trimmed = trimmed[:at]
	}
	if colon := strings.LastIndex(trimmed, ":"); colon != -1 {
		if slash := strings.LastIndex(trimmed, "/"); slash == -1 || colon > slash {
			trimmed = trimmed[:colon]
		}
	}

	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return "", errors.New("github search requires an image name (owner/image)")
	}

	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub container image %q (expected owner/image)", trimmed)
	}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("invalid GitHub container image %q", trimmed)
		}
	}
	return trimmed, nil
}

func parseGitHubContainerNext(headerValue string, baseURL *url.URL) string {
	for _, segment := range strings.Split(headerValue, ",") {
		segment = strings.TrimSpace(segment)
		if segment == "" || !strings.Contains(strings.ToLower(segment), `rel="next"`) {
			continue
		}
		start := strings.Index(segment, "<")
		end := strings.Index(segment, ">")
		if start == -1 || end <= start+1 {
			continue
		}
		target := segment[start+1 : end]
		nextURL, err := url.Parse(target)
		if err != nil {
			continue
		}
		if nextURL.IsAbs() || baseURL == nil {
			return nextURL.String()
		}
		return baseURL.ResolveReference(nextURL).String()
	}
	return ""
}
