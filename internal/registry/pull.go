package registry

import "fmt"

func PullCommand(registryHost, project, image, tag string) string {
	if tag == "" {
		tag = "latest"
	}
	if project == "" {
		return fmt.Sprintf("docker pull %s/%s:%s", registryHost, image, tag)
	}
	return fmt.Sprintf("docker pull %s/%s/%s:%s", registryHost, project, image, tag)
}
