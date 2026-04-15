package registry

import (
	"net/http"
	"net/url"
	"strings"
)

// manifestAcceptHeader is the Accept value for fetching OCI/Docker manifests.
const manifestAcceptHeader = "application/vnd.docker.distribution.manifest.v2+json, " +
	"application/vnd.oci.image.manifest.v1+json, " +
	"application/vnd.docker.distribution.manifest.list.v2+json, " +
	"application/vnd.oci.image.index.v1+json"

func cloneHeader(header http.Header) map[string][]string {
	if len(header) == 0 {
		return nil
	}
	out := make(map[string][]string, len(header))
	for key, values := range header {
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}
	return out
}

func resolveURL(base *url.URL, p string, query url.Values) string {
	if base == nil {
		parsed, err := url.Parse(p)
		if err != nil {
			return p
		}
		if query != nil {
			parsed.RawQuery = query.Encode()
		} else {
			parsed.RawQuery = ""
		}
		return parsed.String()
	}
	resolved := *base
	resolved.Path = strings.TrimSuffix(resolved.Path, "/") + p
	if query != nil {
		resolved.RawQuery = query.Encode()
	} else {
		resolved.RawQuery = ""
	}
	return resolved.String()
}

func resolveNextURL(base *url.URL, next string) string {
	next = strings.TrimSpace(next)
	if next == "" {
		return ""
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return next
	}
	if base == nil {
		return next
	}
	return base.ResolveReference(parsed).String()
}
