package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

const dockerHubBaseURL = "https://hub.docker.com"

type DockerHubClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     RequestLogger
}

func NewDockerHubClient(logger RequestLogger) *DockerHubClient {
	parsed, _ := url.Parse(dockerHubBaseURL)
	return &DockerHubClient{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
}

func (c *DockerHubClient) SearchTags(ctx context.Context, input string) ([]Tag, string, error) {
	namespace, repo, err := c.resolveRepository(ctx, input)
	if err != nil {
		return nil, "", err
	}

	tags, err := c.listTags(ctx, namespace, repo)
	if err != nil {
		return nil, "", err
	}
	return tags, fmt.Sprintf("%s/%s", namespace, repo), nil
}

func (c *DockerHubClient) resolveRepository(ctx context.Context, input string) (string, string, error) {
	trimmed := normalizeDockerHubInput(input)
	if trimmed == "" {
		return "", "", errors.New("docker hub search requires an image name")
	}
	if strings.Contains(trimmed, "/") {
		ns, repo := splitRepo(trimmed)
		if ns == "" || repo == "" {
			return "", "", errors.New("invalid repository name")
		}
		return ns, repo, nil
	}

	// Use Docker Hub search API to resolve a namespace for a short name.
	results, err := c.searchRepositories(ctx, trimmed)
	if err != nil {
		return "", "", err
	}
	if len(results) == 0 {
		return "", "", fmt.Errorf("no Docker Hub repository found for %q", trimmed)
	}

	lower := strings.ToLower(trimmed)
	preferred := "library/" + lower
	for _, result := range results {
		if strings.ToLower(result.RepoFullName()) == preferred {
			ns, repo := splitRepo(result.RepoFullName())
			return ns, repo, nil
		}
	}
	for _, result := range results {
		if strings.EqualFold(result.Name, trimmed) {
			ns, repo := splitRepo(result.RepoFullName())
			return ns, repo, nil
		}
	}

	ns, repo := splitRepo(results[0].RepoFullName())
	if ns == "" || repo == "" {
		return "", "", fmt.Errorf("unable to resolve Docker Hub repository for %q", trimmed)
	}
	return ns, repo, nil
}

func (c *DockerHubClient) searchRepositories(ctx context.Context, query string) ([]dockerHubSearchResult, error) {
	queryValues := url.Values{}
	queryValues.Set("query", query)
	queryValues.Set("page_size", "25")
	endpoint := c.resolve("/v2/search/repositories/", queryValues)

	var payload dockerHubSearchResponse
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Results, nil
}

func (c *DockerHubClient) listTags(ctx context.Context, namespace, repo string) ([]Tag, error) {
	query := url.Values{}
	query.Set("page_size", "100")
	endpoint := c.resolve(fmt.Sprintf("/v2/namespaces/%s/repositories/%s/tags", namespace, repo), query)

	var tags []Tag
	for endpoint != "" {
		var payload dockerHubTagsResponse
		if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &payload); err != nil {
			return nil, err
		}
		for _, entry := range payload.Results {
			tags = append(tags, Tag{
				Name:         entry.Name,
				Digest:       entry.Digest,
				SizeBytes:    entry.FullSize,
				UpdatedAt:    parseDockerHubTime(entry.LastUpdated),
				PushedAt:     parseDockerHubTime(firstNonEmptyString(entry.TagLastPushed, entry.LastUpdated)),
				LastPulledAt: parseDockerHubTime(entry.TagLastPulled),
			})
		}

		endpoint = ""
		if payload.Next != "" {
			endpoint = c.resolveNext(payload.Next)
		}
	}

	return tags, nil
}

func (c *DockerHubClient) resolveNext(next string) string {
	if next == "" {
		return ""
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Host != "" {
		return next
	}
	resolved := *c.baseURL
	resolved.Path = path.Join(strings.TrimSuffix(c.baseURL.Path, "/"), strings.TrimPrefix(parsed.Path, "/"))
	resolved.RawQuery = parsed.RawQuery
	return resolved.String()
}

func (c *DockerHubClient) resolve(p string, query url.Values) string {
	resolved := *c.baseURL
	resolved.Path = strings.TrimSuffix(resolved.Path, "/") + p
	if query != nil {
		resolved.RawQuery = query.Encode()
	}
	return resolved.String()
}

func (c *DockerHubClient) doJSON(ctx context.Context, method, endpoint string, body io.Reader, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("docker hub request failed: %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *DockerHubClient) logRequest(req *http.Request, resp *http.Response) {
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

type dockerHubSearchResponse struct {
	Results []dockerHubSearchResult `json:"results"`
}

type dockerHubSearchResult struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	RepoName  string `json:"repo_name"`
}

func (r dockerHubSearchResult) RepoFullName() string {
	if r.RepoName != "" {
		return r.RepoName
	}
	if r.Namespace != "" && r.Name != "" {
		return r.Namespace + "/" + r.Name
	}
	return ""
}

type dockerHubTagsResponse struct {
	Next    string               `json:"next"`
	Results []dockerHubTagResult `json:"results"`
}

type dockerHubTagResult struct {
	Name          string `json:"name"`
	Digest        string `json:"digest"`
	FullSize      int64  `json:"full_size"`
	LastUpdated   string `json:"last_updated"`
	TagLastPushed string `json:"tag_last_pushed"`
	TagLastPulled string `json:"tag_last_pulled"`
}

func normalizeDockerHubInput(input string) string {
	trimmed := strings.TrimSpace(input)
	trimmed = strings.TrimPrefix(trimmed, "docker.io/")
	trimmed = strings.TrimPrefix(trimmed, "index.docker.io/")
	trimmed = strings.TrimPrefix(trimmed, "registry-1.docker.io/")
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
	return strings.TrimSpace(trimmed)
}

func splitRepo(input string) (string, string) {
	parts := strings.Split(strings.Trim(input, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	namespace := parts[0]
	repo := strings.Join(parts[1:], "/")
	return namespace, repo
}

func parseDockerHubTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
