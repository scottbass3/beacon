package registry

import (
	"fmt"
	"net/url"
	"strings"
)

func PullCommand(registryHost, project, image, tag string) string {
	return fmt.Sprintf("docker pull %s", PullReference(registryHost, project, image, tag))
}

func PullReference(registryHost, project, image, tag string) string {
	registryHost = normalizeRegistryHost(registryHost)
	image = normalizeImagePath(project, image)
	if tag == "" {
		tag = "latest"
	}
	if registryHost == "" {
		return fmt.Sprintf("%s:%s", image, tag)
	}
	return fmt.Sprintf("%s/%s:%s", registryHost, image, tag)
}

func normalizeRegistryHost(registryHost string) string {
	registryHost = strings.TrimSpace(registryHost)
	if registryHost == "" {
		return ""
	}
	if parsed, err := url.Parse(registryHost); err == nil && parsed.Host != "" {
		registryHost = parsed.Host
	}
	registryHost = strings.Trim(registryHost, "/")
	if slash := strings.Index(registryHost, "/"); slash >= 0 {
		registryHost = registryHost[:slash]
	}
	return registryHost
}

func normalizeImagePath(project, image string) string {
	project = strings.Trim(project, " /")
	image = strings.Trim(image, " /")
	if project == "" || image == "" || strings.HasPrefix(image, project+"/") {
		return image
	}
	return project + "/" + image
}
