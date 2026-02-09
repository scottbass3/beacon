package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const harborPageSize = 100

// HarborClient implements Harbor API v2.0.
type HarborClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	auth       Auth
	logger     RequestLogger
}

func newHarborClient(baseURL *url.URL, auth Auth, logger RequestLogger) *HarborClient {
	return &HarborClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		auth:   auth,
		logger: logger,
	}
}

func (c *HarborClient) ListImages(ctx context.Context) ([]Image, error) {
	var all []harborProject
	page := 1
	for {
		var batch []harborProject
		endpoint := c.resolve("/api/v2.0/projects", url.Values{
			"page":      []string{fmt.Sprintf("%d", page)},
			"page_size": []string{fmt.Sprintf("%d", harborPageSize)},
		})
		if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < harborPageSize {
			break
		}
		page++
	}

	images := make([]Image, 0)
	for _, project := range all {
		repos, err := c.listProjectRepos(ctx, project.Name)
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			images = append(images, Image{
				Name:       repo.Name,
				Repository: repo.Name,
				TagCount:   repo.ArtifactCount,
				PullCount:  repo.PullCount,
				UpdatedAt:  parseHarborTime(repo.UpdateTime),
			})
		}
	}

	sort.Slice(images, func(i, j int) bool {
		return images[i].Name < images[j].Name
	})

	return images, nil
}

func (c *HarborClient) ListTags(ctx context.Context, image string) ([]Tag, error) {
	project, repo := splitHarborImage(image)
	if project == "" || repo == "" {
		return nil, nil
	}

	var all []harborArtifact
	page := 1
	for {
		var batch []harborArtifact
		endpoint := c.resolve(fmt.Sprintf("/api/v2.0/projects/%s/repositories/%s/artifacts", url.PathEscape(project), url.PathEscape(repo)), url.Values{
			"page":      []string{fmt.Sprintf("%d", page)},
			"page_size": []string{fmt.Sprintf("%d", harborPageSize)},
		})
		if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < harborPageSize {
			break
		}
		page++
	}

	var tags []Tag
	for _, artifact := range all {
		if len(artifact.Tags) == 0 {
			continue
		}
		for _, t := range artifact.Tags {
			tags = append(tags, Tag{
				Name:         t.Name,
				Digest:       artifact.Digest,
				SizeBytes:    artifact.Size,
				UpdatedAt:    parseHarborTime(artifact.UpdateTime),
				PushedAt:     parseHarborTime(t.PushTime),
				LastPulledAt: parseHarborTime(t.PullTime),
			})
		}
	}

	sort.Slice(tags, func(i, j int) bool {
		return tags[i].Name < tags[j].Name
	})

	return tags, nil
}

func (c *HarborClient) ListTagHistory(ctx context.Context, image, tag string) ([]HistoryEntry, error) {
	image = strings.TrimSpace(image)
	tag = strings.TrimSpace(tag)
	if image == "" || tag == "" {
		return nil, nil
	}

	manifest, err := c.getManifest(ctx, image, tag)
	if err != nil {
		return nil, err
	}
	if manifest.Config.Digest == "" {
		return nil, nil
	}
	cfg, err := c.getConfig(ctx, image, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}
	return buildHistory(manifest, cfg), nil
}

func (c *HarborClient) DeleteTag(ctx context.Context, image, tag string) error {
	return ErrNotSupported
}

func (c *HarborClient) RenameTag(ctx context.Context, image, from, to string) error {
	return ErrNotSupported
}

func (c *HarborClient) resolve(path string, query url.Values) string {
	resolved := *c.baseURL
	resolved.Path = strings.TrimSuffix(resolved.Path, "/") + path
	if query != nil {
		resolved.RawQuery = query.Encode()
	} else {
		resolved.RawQuery = ""
	}
	return resolved.String()
}

func (c *HarborClient) doJSON(ctx context.Context, method, endpoint string, body io.Reader, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	if !c.auth.Harbor.Anonymous {
		req.SetBasicAuth(c.auth.Harbor.Username, c.auth.Harbor.Password)
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("harbor request failed: %s", resp.Status)
	}

	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *HarborClient) getManifest(ctx context.Context, image, reference string) (manifestV2, error) {
	endpoint := c.resolve("/v2/"+image+"/manifests/"+reference, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return manifestV2{}, err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
	}, ", "))
	if !c.auth.Harbor.Anonymous {
		req.SetBasicAuth(c.auth.Harbor.Username, c.auth.Harbor.Password)
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return manifestV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return manifestV2{}, fmt.Errorf("harbor manifest request failed: %s", resp.Status)
	}

	var manifest manifestV2
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return manifestV2{}, err
	}
	return manifest, nil
}

func (c *HarborClient) getConfig(ctx context.Context, image, digest string) (configV2, error) {
	endpoint := c.resolve("/v2/"+image+"/blobs/"+digest, nil)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return configV2{}, err
	}
	if !c.auth.Harbor.Anonymous {
		req.SetBasicAuth(c.auth.Harbor.Username, c.auth.Harbor.Password)
	}

	resp, err := c.httpClient.Do(req)
	c.logRequest(req, resp)
	if err != nil {
		return configV2{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return configV2{}, fmt.Errorf("harbor config request failed: %s", resp.Status)
	}

	var cfg configV2
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return configV2{}, err
	}
	return cfg, nil
}

func (c *HarborClient) logRequest(req *http.Request, resp *http.Response) {
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

type harborProject struct {
	Name         string `json:"name"`
	RepoCount    int    `json:"repo_count"`
	CreationTime string `json:"creation_time"`
	UpdateTime   string `json:"update_time"`
}

type harborRepository struct {
	Name          string `json:"name"`
	ArtifactCount int    `json:"artifact_count"`
	PullCount     int    `json:"pull_count"`
	UpdateTime    string `json:"update_time"`
}

type harborArtifact struct {
	Digest     string        `json:"digest"`
	Size       int64         `json:"size"`
	Tags       []harborTag   `json:"tags"`
	UpdateTime string        `json:"update_time"`
	PushTime   string        `json:"push_time"`
	PullTime   string        `json:"pull_time"`
	ExtraAttrs harborAttrs   `json:"extra_attrs"`
	Type       string        `json:"type"`
	References []interface{} `json:"references"`
}

type harborTag struct {
	Name     string `json:"name"`
	PushTime string `json:"push_time"`
	PullTime string `json:"pull_time"`
}

type harborAttrs map[string]interface{}

func parseHarborTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed
	}
	return time.Time{}
}

func (c *HarborClient) listProjectRepos(ctx context.Context, project string) ([]harborRepository, error) {
	if project == "" {
		return nil, nil
	}
	var all []harborRepository
	page := 1
	for {
		var batch []harborRepository
		endpoint := c.resolve(fmt.Sprintf("/api/v2.0/projects/%s/repositories", url.PathEscape(project)), url.Values{
			"page":      []string{fmt.Sprintf("%d", page)},
			"page_size": []string{fmt.Sprintf("%d", harborPageSize)},
		})
		if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &batch); err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < harborPageSize {
			break
		}
		page++
	}
	return all, nil
}

func splitHarborImage(image string) (string, string) {
	trimmed := strings.Trim(image, "/")
	if trimmed == "" {
		return "", ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
