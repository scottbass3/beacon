package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scottbass3/beacon/internal/registry"
)

func (m *Model) handleEnter() tea.Cmd {
	list := m.listView()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(list.indices) {
		return nil
	}
	index := list.indices[cursor]

	switch m.focus {
	case FocusProjects:
		if index < 0 || index >= len(m.projects) {
			return nil
		}
		selected := m.projects[index]
		if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
			m.selectedProject = selected.Name
			m.hasSelectedProject = true
			m.images = nil
			m.selectedImage = registry.Image{}
			m.hasSelectedImage = false
			m.tags = nil
			m.focus = FocusImages
			m.status = fmt.Sprintf("Loading images for %s...", selected.Name)
			m.clearFilter()
			m.syncTable()
			m.startLoading()
			return loadProjectImagesCmd(projectClient, selected.Name)
		}
		m.status = "Project images are not available for this registry client"
		m.syncTable()
		return nil
	case FocusImages:
		visible := m.visibleImages()
		if index < 0 || index >= len(visible) {
			return nil
		}
		selected := visible[index]
		m.selectedImage = selected
		m.hasSelectedImage = true
		m.selectedTag = registry.Tag{}
		m.hasSelectedTag = false
		m.tags = nil
		m.focus = FocusTags
		m.status = fmt.Sprintf("Loading tags for %s...", selected.Name)
		m.clearFilter()
		m.syncTable()
		m.startLoading()
		return loadTagsCmd(m.registryClient, selected.Name)
	case FocusTags:
		selected := m.tags[index]
		m.selectedTag = selected
		m.hasSelectedTag = true
		m.history = nil
		m.focus = FocusHistory
		m.status = fmt.Sprintf("Loading history for %s:%s...", m.selectedImage.Name, selected.Name)
		m.clearFilter()
		m.syncTable()
		m.startLoading()
		return loadHistoryCmd(m.registryClient, m.selectedImage.Name, selected.Name)
	default:
		return nil
	}
}

func (m *Model) handleEscape() tea.Cmd {
	switch m.focus {
	case FocusHistory:
		m.history = nil
		m.selectedTag = registry.Tag{}
		m.hasSelectedTag = false
		if m.dockerHubActive {
			m.focus = FocusDockerHubTags
		} else if m.githubActive {
			m.focus = FocusGitHubTags
		} else {
			m.focus = FocusTags
		}
		m.clearFilter()
		m.syncTable()
		return nil
	case FocusTags:
		m.tags = nil
		m.hasSelectedImage = false
		m.selectedImage = registry.Image{}
		m.focus = FocusImages
		m.clearFilter()
		m.syncTable()
		return nil
	case FocusImages:
		if m.tableSpec().SupportsProjects {
			m.selectedProject = ""
			m.hasSelectedProject = false
			m.focus = FocusProjects
			m.clearFilter()
			m.syncTable()
			return nil
		}
		m.clearFilter()
		m.syncTable()
		return nil
	case FocusProjects:
		m.clearFilter()
		m.syncTable()
		return nil
	default:
		return nil
	}
}

func (m *Model) clearFilter() {
	m.filterInput.SetValue("")
	m.stopFilterEditing()
}

func (m *Model) stopFilterEditing() {
	m.filterInput.Blur()
	m.filterActive = false
}

func (m *Model) startLoading() {
	m.loadingCount++
}

func (m *Model) stopLoading() {
	if m.loadingCount <= 0 {
		return
	}
	m.loadingCount--
}

func (m Model) isLoading() bool {
	return m.loadingCount > 0
}

func (m Model) emptyBodyMessage() string {
	if m.isLoading() {
		return "Loading, waiting for server response..."
	}

	filter := strings.TrimSpace(m.filterInput.Value())
	if filter != "" {
		return fmt.Sprintf("No results for filter %q", filter)
	}

	switch m.focus {
	case FocusProjects:
		return "No projects to display."
	case FocusImages:
		if m.hasSelectedProject {
			return fmt.Sprintf("No images found in project %s.", m.selectedProject)
		}
		return "No images to display."
	case FocusTags:
		if m.hasSelectedImage {
			return fmt.Sprintf("No tags found for %s.", m.selectedImage.Name)
		}
		return "No tags to display."
	case FocusHistory:
		if m.hasSelectedImage && m.hasSelectedTag {
			return fmt.Sprintf("No history found for %s:%s.", m.selectedImage.Name, m.selectedTag.Name)
		}
		return "No history entries to display."
	case FocusDockerHubTags:
		query := strings.TrimSpace(m.dockerHubInput.Value())
		if m.dockerHubImage != "" {
			return fmt.Sprintf("No tags found for %s.", m.dockerHubImage)
		}
		if query == "" {
			return "Type an image name and press Enter to search Docker Hub."
		}
		return fmt.Sprintf("No tags found for query %q.", query)
	case FocusGitHubTags:
		query := strings.TrimSpace(m.githubInput.Value())
		if m.githubImage != "" {
			return fmt.Sprintf("No tags found for %s.", m.githubImage)
		}
		if query == "" {
			return "Type an image name and press Enter to search GHCR."
		}
		return fmt.Sprintf("No tags found for query %q.", query)
	default:
		return "No data to display."
	}
}
