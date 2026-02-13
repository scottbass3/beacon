package tui

import (
	"errors"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m Model) updateKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.helpActive {
		return m.handleHelpKey(msg)
	}
	if isHelpShortcut(msg) &&
		!m.commandActive &&
		!m.filterActive &&
		!(m.dockerHubActive && m.dockerHubInputFocus) &&
		!(m.githubActive && m.githubInputFocus) &&
		!m.isConfirmModalActive() &&
		!m.isContextFormActive() &&
		!m.isContextSelectionActive() &&
		!m.isAuthModalActive() {
		return m.openHelp()
	}
	if m.isConfirmModalActive() {
		return m.handleConfirmKey(msg)
	}
	if m.isContextFormActive() {
		return m.handleContextFormKey(msg)
	}
	if m.isContextSelectionActive() {
		return m.handleContextSelectionKey(msg)
	}
	if m.isAuthModalActive() {
		return m.handleAuthKey(msg)
	}
	if m.commandActive {
		return m.handleCommandKey(msg)
	}
	if m.dockerHubActive {
		return m.handleDockerHubKey(msg)
	}
	if m.githubActive {
		return m.handleGitHubKey(msg)
	}
	return m.handleKey(msg)
}

func (m Model) updateMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.helpActive ||
		m.commandActive ||
		m.isConfirmModalActive() ||
		m.isContextFormActive() ||
		m.isContextSelectionActive() ||
		m.isAuthModalActive() {
		return m, nil
	}
	if m.dockerHubActive {
		return m.handleExternalMouse(externalModeDockerHub, msg)
	}
	if m.githubActive {
		return m.handleExternalMouse(externalModeGitHub, msg)
	}
	return m.handleMouse(msg)
}

func (m Model) updateWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.syncTable()
	return m, nil
}

func (m Model) updateImagesMsg(msg imagesMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	if msg.err != nil {
		m.status = fmt.Sprintf("Error loading images: %v", msg.err)
		m.syncTable()
		return m, nil
	}
	m.images = msg.images
	m.projects = nil
	m.tags = nil
	m.history = nil
	m.selectedProject = ""
	m.hasSelectedProject = false
	m.hasSelectedImage = false
	m.hasSelectedTag = false
	m.selectedTag = registry.Tag{}
	m.focus = m.defaultFocus()
	if m.tableSpec().SupportsProjects {
		m.projects = deriveProjects(msg.images)
		m.status = fmt.Sprintf("Loaded %d images across %d projects", len(msg.images), len(m.projects))
	} else {
		m.status = fmt.Sprintf("Loaded %d images", len(msg.images))
	}
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m Model) updateProjectsMsg(msg projectsMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	if msg.err != nil {
		m.status = fmt.Sprintf("Error loading projects: %v", msg.err)
		m.syncTable()
		return m, nil
	}
	m.projects = toProjectInfos(msg.projects)
	m.images = nil
	m.tags = nil
	m.history = nil
	m.selectedProject = ""
	m.hasSelectedProject = false
	m.selectedImage = registry.Image{}
	m.hasSelectedImage = false
	m.selectedTag = registry.Tag{}
	m.hasSelectedTag = false
	m.focus = FocusProjects
	m.status = fmt.Sprintf("Loaded %d projects", len(msg.projects))
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m Model) updateProjectImagesMsg(msg projectImagesMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	if msg.err != nil {
		m.status = fmt.Sprintf("Error loading images for %s: %v", msg.project, msg.err)
		m.syncTable()
		return m, nil
	}
	if !m.hasSelectedProject || m.selectedProject != msg.project {
		return m, nil
	}
	m.images = msg.images
	m.tags = nil
	m.history = nil
	m.selectedImage = registry.Image{}
	m.hasSelectedImage = false
	m.selectedTag = registry.Tag{}
	m.hasSelectedTag = false
	m.focus = FocusImages
	m.status = fmt.Sprintf("Loaded %d images for %s", len(msg.images), msg.project)
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m Model) updateTagsMsg(msg tagsMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	if msg.err != nil {
		m.status = fmt.Sprintf("Error loading tags: %v", msg.err)
		m.syncTable()
		return m, nil
	}
	m.tags = msg.tags
	m.history = nil
	m.hasSelectedTag = false
	m.selectedTag = registry.Tag{}
	if m.hasSelectedImage {
		m.selectedImage.TagCount = len(msg.tags)
		for i := range m.images {
			if m.images[i].Name == m.selectedImage.Name {
				m.images[i].TagCount = len(msg.tags)
				break
			}
		}
	}
	m.focus = FocusTags
	m.status = fmt.Sprintf("Loaded %d tags", len(msg.tags))
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m Model) updateHistoryMsg(msg historyMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	if msg.err != nil {
		m.status = fmt.Sprintf("Error loading history: %v", msg.err)
		m.syncTable()
		return m, nil
	}
	m.history = msg.history
	m.focus = FocusHistory
	m.status = fmt.Sprintf("Loaded %d history entries", len(msg.history))
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m Model) updateDockerPullMsg(msg dockerPullMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	if msg.err != nil {
		m.status = fmt.Sprintf("Failed to pull %s: %v", msg.reference, msg.err)
		return m, nil
	}
	m.status = fmt.Sprintf("Pulled %s", msg.reference)
	return m, nil
}

func (m Model) updateDockerHubTagsMsg(msg dockerHubTagsMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	m.dockerHubLoading = false
	if !m.dockerHubActive {
		return m, nil
	}
	m.dockerHubRateLimit = msg.rateLimit
	m.applyDockerHubRateLimit(msg.retryAfter)
	if msg.err != nil {
		var rateErr *registry.DockerHubRateLimitError
		if errors.As(msg.err, &rateErr) {
			m.status = m.dockerHubRateLimitStatus("Docker Hub rate limit reached")
		} else {
			m.status = fmt.Sprintf("Error searching Docker Hub: %v", msg.err)
		}
		m.syncTable()
		return m, nil
	}
	if msg.appendPage {
		m.dockerHubTags = append(m.dockerHubTags, msg.tags...)
	} else {
		m.dockerHubTags = msg.tags
		m.clearFilter()
	}
	m.dockerHubImage = msg.image
	m.dockerHubNext = msg.next
	m.focus = FocusDockerHubTags
	m.status = m.dockerHubLoadedStatus()
	m.syncTable()
	if cmd := m.maybeLoadDockerHubForFilter(); cmd != nil {
		return m, cmd
	}
	return m, nil
}

func (m Model) updateGitHubTagsMsg(msg githubTagsMsg) (tea.Model, tea.Cmd) {
	m.stopLoading()
	m.githubLoading = false
	if !m.githubActive {
		return m, nil
	}
	if msg.err != nil {
		m.status = fmt.Sprintf("Error searching GHCR: %v", msg.err)
		m.syncTable()
		return m, nil
	}
	if msg.appendPage {
		m.githubTags = append(m.githubTags, msg.tags...)
	} else {
		m.githubTags = msg.tags
		m.clearFilter()
	}
	m.githubImage = msg.image
	m.githubNext = msg.next
	m.focus = FocusGitHubTags
	m.status = m.githubLoadedStatus()
	m.syncTable()
	if cmd := m.maybeLoadGitHubForFilter(); cmd != nil {
		return m, cmd
	}
	return m, nil
}

func (m Model) updateLogMsg(msg logMsg) (tea.Model, tea.Cmd) {
	m.appendLog(string(msg))
	m.syncTable()
	if m.logCh != nil {
		return m, listenLogs(m.logCh)
	}
	return m, nil
}

func (m Model) updateInitClientMsg(msg initClientMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.status = fmt.Sprintf("Error initializing registry: %v", msg.err)
		m.authError = msg.err.Error()
		return m, nil
	}
	m.registryClient = msg.client
	return m, m.initialLoadCmd()
}
