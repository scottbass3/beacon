package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m *Model) refreshCurrent() tea.Cmd {
	if m.githubActive {
		if m.focus == FocusHistory && m.hasSelectedTag && strings.TrimSpace(m.githubImage) != "" {
			m.status = fmt.Sprintf("Refreshing history for %s:%s...", m.githubImage, m.selectedTag.Name)
			m.startLoading()
			return loadGitHubHistoryCmd(m.githubImage, m.selectedTag.Name, m.logger)
		}
		return m.refreshGitHub()
	}
	if m.dockerHubActive {
		if m.focus == FocusHistory && m.hasSelectedTag && strings.TrimSpace(m.dockerHubImage) != "" {
			m.status = fmt.Sprintf("Refreshing history for %s:%s...", m.dockerHubImage, m.selectedTag.Name)
			m.startLoading()
			return loadDockerHubHistoryCmd(m.dockerHubImage, m.selectedTag.Name, m.logger)
		}
		return m.refreshDockerHub()
	}
	switch m.focus {
	case FocusProjects:
		if m.registryClient == nil {
			m.status = "Registry not configured"
			return nil
		}
		if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
			m.status = fmt.Sprintf("Refreshing projects from %s...", m.registryHost)
			m.startLoading()
			return loadProjectsCmd(projectClient)
		}
		m.status = "Project listing is not available for this registry client"
		return nil
	case FocusImages:
		if m.registryClient == nil {
			m.status = "Registry not configured"
			return nil
		}
		if m.hasSelectedProject {
			if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
				m.status = fmt.Sprintf("Refreshing images for %s...", m.selectedProject)
				m.startLoading()
				return loadProjectImagesCmd(projectClient, m.selectedProject)
			}
			m.status = "Project images are not available for this registry client"
			return nil
		}
		m.status = fmt.Sprintf("Refreshing images from %s...", m.registryHost)
		m.startLoading()
		return loadImagesCmd(m.registryClient)
	case FocusTags:
		if !m.hasSelectedImage {
			if m.registryClient == nil {
				m.status = "Registry not configured"
				return nil
			}
			if m.hasSelectedProject {
				if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
					m.status = fmt.Sprintf("Refreshing images for %s...", m.selectedProject)
					m.startLoading()
					return loadProjectImagesCmd(projectClient, m.selectedProject)
				}
				m.status = "Project images are not available for this registry client"
				return nil
			}
			m.status = fmt.Sprintf("Refreshing images from %s...", m.registryHost)
			m.startLoading()
			return loadImagesCmd(m.registryClient)
		}
		m.status = fmt.Sprintf("Refreshing tags for %s...", m.selectedImage.Name)
		m.startLoading()
		return loadTagsCmd(m.registryClient, m.selectedImage.Name)
	case FocusHistory:
		if !m.hasSelectedTag {
			if m.registryClient == nil {
				m.status = "Registry not configured"
				return nil
			}
			m.status = fmt.Sprintf("Refreshing tags for %s...", m.selectedImage.Name)
			m.startLoading()
			return loadTagsCmd(m.registryClient, m.selectedImage.Name)
		}
		m.status = fmt.Sprintf("Refreshing history for %s:%s...", m.selectedImage.Name, m.selectedTag.Name)
		m.startLoading()
		return loadHistoryCmd(m.registryClient, m.selectedImage.Name, m.selectedTag.Name)
	default:
		return m.initialLoadCmd()
	}
}

func (m *Model) refreshDockerHub() tea.Cmd {
	return m.refreshExternal(externalModeDockerHub)
}

func (m *Model) searchDockerHub(query string) tea.Cmd {
	return m.searchExternal(externalModeDockerHub, query)
}

func (m *Model) openDockerHubTagHistory() tea.Cmd {
	return m.openExternalTagHistory(externalModeDockerHub)
}

func (m *Model) maybeLoadDockerHubOnBottom(msg tea.KeyMsg) tea.Cmd {
	return m.maybeLoadExternalOnBottomKey(externalModeDockerHub, msg)
}

func (m *Model) maybeLoadDockerHubForFilter() tea.Cmd {
	return m.maybeLoadExternalForFilter(externalModeDockerHub)
}

func (m *Model) requestNextDockerHubPage(forFilter bool) tea.Cmd {
	return m.requestNextExternalPage(externalModeDockerHub, forFilter)
}

func (m *Model) applyDockerHubRateLimit(retryAfter time.Duration) {
	if retryAfter > 0 {
		m.dockerHubRetryUntil = time.Now().Add(retryAfter)
		return
	}
	if m.dockerHubRateLimit.Remaining == 0 && !m.dockerHubRateLimit.ResetAt.IsZero() {
		m.dockerHubRetryUntil = m.dockerHubRateLimit.ResetAt
		return
	}
	if !m.dockerHubRetryUntil.IsZero() && time.Now().After(m.dockerHubRetryUntil) {
		m.dockerHubRetryUntil = time.Time{}
	}
}

func (m Model) dockerHubRateLimitStatus(prefix string) string {
	now := time.Now()
	if !m.dockerHubRetryUntil.IsZero() && now.Before(m.dockerHubRetryUntil) {
		wait := m.dockerHubRetryUntil.Sub(now).Round(time.Second)
		return fmt.Sprintf("%s. Retry in %s", prefix, wait)
	}
	if !m.dockerHubRateLimit.ResetAt.IsZero() && now.Before(m.dockerHubRateLimit.ResetAt) {
		return fmt.Sprintf("%s. Resets at %s", prefix, m.dockerHubRateLimit.ResetAt.Local().Format("15:04:05"))
	}
	return prefix
}

func (m Model) dockerHubRateLimitSuffix() string {
	limit := m.dockerHubRateLimit
	if limit.Limit <= 0 || limit.Remaining < 0 {
		return ""
	}
	suffix := fmt.Sprintf(" | rate %d/%d", limit.Remaining, limit.Limit)
	if !limit.ResetAt.IsZero() {
		suffix += fmt.Sprintf(" reset %s", limit.ResetAt.Local().Format("15:04:05"))
	}
	return suffix
}

func (m Model) dockerHubLoadedStatus() string {
	return m.externalLoadedStatus(externalModeDockerHub)
}

func (m *Model) refreshGitHub() tea.Cmd {
	return m.refreshExternal(externalModeGitHub)
}

func (m *Model) searchGitHub(query string) tea.Cmd {
	return m.searchExternal(externalModeGitHub, query)
}

func (m *Model) openGitHubTagHistory() tea.Cmd {
	return m.openExternalTagHistory(externalModeGitHub)
}

func (m *Model) maybeLoadGitHubOnBottom(msg tea.KeyMsg) tea.Cmd {
	return m.maybeLoadExternalOnBottomKey(externalModeGitHub, msg)
}

func (m *Model) maybeLoadGitHubForFilter() tea.Cmd {
	return m.maybeLoadExternalForFilter(externalModeGitHub)
}

func (m *Model) requestNextGitHubPage(forFilter bool) tea.Cmd {
	return m.requestNextExternalPage(externalModeGitHub, forFilter)
}

func (m Model) githubLoadedStatus() string {
	return m.externalLoadedStatus(externalModeGitHub)
}

func (m *Model) initialLoadCmd() tea.Cmd {
	if m.registryClient == nil {
		m.status = "Registry not configured"
		return nil
	}
	if m.tableSpec().SupportsProjects {
		if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
			m.status = fmt.Sprintf("Loading projects from %s...", m.registryHost)
			m.startLoading()
			return loadProjectsCmd(projectClient)
		}
		m.status = "Project listing is not available for this registry client"
		return nil
	}
	m.status = fmt.Sprintf("Connecting to %s...", m.registryHost)
	m.startLoading()
	return loadImagesCmd(m.registryClient)
}
