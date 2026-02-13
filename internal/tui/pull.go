package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

var runDockerPull = dockerPull

func (m *Model) pullSelectedTagWithDocker() tea.Cmd {
	reference, ok := m.selectedTagReferenceForPull()
	if !ok {
		m.status = "No tag selected to pull"
		return nil
	}

	m.status = fmt.Sprintf("Pulling %s...", reference)
	m.startLoading()
	return pullSelectedTagCmd(reference)
}

func (m Model) selectedTagReferenceForPull() (string, bool) {
	image, tag, ok := m.selectedTagImageAndTag()
	if !ok {
		return "", false
	}

	switch m.focus {
	case FocusTags:
		if _, ok := formatTagReference(image, tag); !ok {
			return "", false
		}
		return registry.PullReference(m.registryHost, m.selectedProject, image, tag), true
	default:
		return formatTagReference(image, tag)
	}
}

func pullSelectedTagCmd(reference string) tea.Cmd {
	return func() tea.Msg {
		return dockerPullMsg{reference: reference, err: runDockerPull(reference)}
	}
}

func dockerPull(reference string) error {
	cmd := exec.Command("docker", "pull", reference)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	details := strings.TrimSpace(string(output))
	if details == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, details)
}
