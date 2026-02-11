package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m Model) externalActive(kind externalModeKind) bool {
	switch kind {
	case externalModeGitHub:
		return m.githubActive
	default:
		return m.dockerHubActive
	}
}

func (m *Model) setExternalActive(kind externalModeKind, value bool) {
	switch kind {
	case externalModeGitHub:
		m.githubActive = value
	default:
		m.dockerHubActive = value
	}
}

func (m Model) externalPrevFocus(kind externalModeKind) Focus {
	switch kind {
	case externalModeGitHub:
		return m.githubPrevFocus
	default:
		return m.dockerHubPrevFocus
	}
}

func (m *Model) setExternalPrevFocus(kind externalModeKind, value Focus) {
	switch kind {
	case externalModeGitHub:
		m.githubPrevFocus = value
	default:
		m.dockerHubPrevFocus = value
	}
}

func (m Model) externalPrevStatus(kind externalModeKind) string {
	switch kind {
	case externalModeGitHub:
		return m.githubPrevStatus
	default:
		return m.dockerHubPrevStatus
	}
}

func (m *Model) setExternalPrevStatus(kind externalModeKind, value string) {
	switch kind {
	case externalModeGitHub:
		m.githubPrevStatus = value
	default:
		m.dockerHubPrevStatus = value
	}
}

func (m Model) externalInputFocused(kind externalModeKind) bool {
	switch kind {
	case externalModeGitHub:
		return m.githubInputFocus
	default:
		return m.dockerHubInputFocus
	}
}

func (m Model) isExternalInputFocused(kind externalModeKind) bool {
	switch kind {
	case externalModeGitHub:
		return m.githubInput.Focused()
	default:
		return m.dockerHubInput.Focused()
	}
}

func (m *Model) setExternalInputFocus(kind externalModeKind, value bool) {
	switch kind {
	case externalModeGitHub:
		m.githubInputFocus = value
	default:
		m.dockerHubInputFocus = value
	}
}

func (m *Model) focusExternalInput(kind externalModeKind) tea.Cmd {
	switch kind {
	case externalModeGitHub:
		return m.githubInput.Focus()
	default:
		return m.dockerHubInput.Focus()
	}
}

func (m *Model) blurExternalInput(kind externalModeKind) {
	switch kind {
	case externalModeGitHub:
		m.githubInput.Blur()
	default:
		m.dockerHubInput.Blur()
	}
}

func (m *Model) externalInputCursorEnd(kind externalModeKind) {
	switch kind {
	case externalModeGitHub:
		m.githubInput.CursorEnd()
	default:
		m.dockerHubInput.CursorEnd()
	}
}

func (m Model) externalInputValue(kind externalModeKind) string {
	switch kind {
	case externalModeGitHub:
		return m.githubInput.Value()
	default:
		return m.dockerHubInput.Value()
	}
}

func (m *Model) setExternalInputValue(kind externalModeKind, value string) {
	switch kind {
	case externalModeGitHub:
		m.githubInput.SetValue(value)
	default:
		m.dockerHubInput.SetValue(value)
	}
}

func (m *Model) updateExternalInput(kind externalModeKind, msg tea.KeyMsg) tea.Cmd {
	var cmd tea.Cmd
	switch kind {
	case externalModeGitHub:
		m.githubInput, cmd = m.githubInput.Update(msg)
	default:
		m.dockerHubInput, cmd = m.dockerHubInput.Update(msg)
	}
	return cmd
}

func (m Model) externalImage(kind externalModeKind) string {
	switch kind {
	case externalModeGitHub:
		return m.githubImage
	default:
		return m.dockerHubImage
	}
}

func (m *Model) setExternalImage(kind externalModeKind, value string) {
	switch kind {
	case externalModeGitHub:
		m.githubImage = value
	default:
		m.dockerHubImage = value
	}
}

func (m Model) externalTags(kind externalModeKind) []registry.Tag {
	switch kind {
	case externalModeGitHub:
		return m.githubTags
	default:
		return m.dockerHubTags
	}
}

func (m *Model) setExternalTags(kind externalModeKind, tags []registry.Tag) {
	switch kind {
	case externalModeGitHub:
		m.githubTags = tags
	default:
		m.dockerHubTags = tags
	}
}

func (m Model) externalNext(kind externalModeKind) string {
	switch kind {
	case externalModeGitHub:
		return m.githubNext
	default:
		return m.dockerHubNext
	}
}

func (m *Model) setExternalNext(kind externalModeKind, next string) {
	switch kind {
	case externalModeGitHub:
		m.githubNext = next
	default:
		m.dockerHubNext = next
	}
}

func (m Model) externalLoading(kind externalModeKind) bool {
	switch kind {
	case externalModeGitHub:
		return m.githubLoading
	default:
		return m.dockerHubLoading
	}
}

func (m *Model) setExternalLoading(kind externalModeKind, value bool) {
	switch kind {
	case externalModeGitHub:
		m.githubLoading = value
	default:
		m.dockerHubLoading = value
	}
}
