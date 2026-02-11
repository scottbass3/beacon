package tui

import "fmt"

type externalModeKind int

const (
	externalModeDockerHub externalModeKind = iota
	externalModeGitHub
)

func (k externalModeKind) focus() Focus {
	switch k {
	case externalModeGitHub:
		return FocusGitHubTags
	default:
		return FocusDockerHubTags
	}
}

func (k externalModeKind) searchPlaceholder() string {
	switch k {
	case externalModeGitHub:
		return "Enter an image name to search GHCR (owner/image)"
	default:
		return "Enter an image name to search Docker Hub"
	}
}

func (k externalModeKind) searchingStatus(query string) string {
	switch k {
	case externalModeGitHub:
		return fmt.Sprintf("Searching GHCR for %s...", query)
	default:
		return fmt.Sprintf("Searching Docker Hub for %s...", query)
	}
}

func (k externalModeKind) loadedStatus(image string, count int, hasMore bool) string {
	switch k {
	case externalModeGitHub:
		status := fmt.Sprintf("GHCR: %s (%d tags)", image, count)
		if hasMore {
			status += " [more]"
		}
		return status
	default:
		status := fmt.Sprintf("Docker Hub: %s (%d tags)", image, count)
		if hasMore {
			status += " [more]"
		}
		return status
	}
}

func (k externalModeKind) modeStatus() string {
	switch k {
	case externalModeGitHub:
		return "GHCR search"
	default:
		return "Docker Hub search"
	}
}

func (k externalModeKind) loadingMoreStatus(image string, forFilter bool) string {
	if forFilter {
		return fmt.Sprintf("Loading more tags for %s to match filter...", image)
	}
	return fmt.Sprintf("Loading more tags for %s...", image)
}

func (k externalModeKind) loadingHistoryStatus(image, tag string) string {
	return fmt.Sprintf("Loading history for %s:%s...", image, tag)
}
