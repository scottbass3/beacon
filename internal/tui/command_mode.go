package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "esc":
		return m.exitCommandMode()
	case "tab":
		if len(m.commandMatches) > 0 {
			m.commandInput.SetValue(m.commandMatches[m.commandIndex])
			m.commandInput.CursorEnd()
			return m, nil
		}
	case "up":
		if len(m.commandMatches) > 0 {
			m.commandIndex--
			if m.commandIndex < 0 {
				m.commandIndex = len(m.commandMatches) - 1
			}
		}
	case "down":
		if len(m.commandMatches) > 0 {
			m.commandIndex = (m.commandIndex + 1) % len(m.commandMatches)
		}
	case "enter":
		return m.runCommand()
	}

	before := m.commandInput.Value()
	var cmd tea.Cmd
	m.commandInput, cmd = m.commandInput.Update(msg)
	if m.commandInput.Value() != before {
		m.commandIndex = 0
		m.commandMatches = matchCommands(commandToken(m.commandInput.Value()))
	}
	return m, cmd
}

func (m Model) enterCommandMode() (tea.Model, tea.Cmd) {
	m.commandPrevFilterActive = m.filterActive
	m.commandPrevDockerHubSearch = m.dockerHubActive && m.dockerHubInputFocus
	m.commandPrevGitHubSearch = m.githubActive && m.githubInputFocus
	if m.filterActive {
		m.stopFilterEditing()
	}
	if m.dockerHubInputFocus {
		m.dockerHubInputFocus = false
		m.dockerHubInput.Blur()
	}
	if m.githubInputFocus {
		m.githubInputFocus = false
		m.githubInput.Blur()
	}
	m.commandActive = true
	m.commandError = ""
	m.commandInput.SetValue("")
	cmd := m.commandInput.Focus()
	m.commandInput.CursorEnd()
	m.commandMatches = matchCommands("")
	m.commandIndex = 0
	m.syncTable()
	return m, cmd
}

func (m Model) exitCommandMode() (tea.Model, tea.Cmd) {
	m.commandActive = false
	m.commandInput.Blur()
	m.commandInput.SetValue("")
	m.commandIndex = 0
	m.commandError = ""
	m.commandMatches = nil
	var cmd tea.Cmd
	if m.commandPrevFilterActive {
		m.filterActive = true
		cmd = m.filterInput.Focus()
		m.filterInput.CursorEnd()
	} else if m.commandPrevDockerHubSearch {
		m.dockerHubInputFocus = true
		cmd = m.dockerHubInput.Focus()
		m.dockerHubInput.CursorEnd()
	} else if m.commandPrevGitHubSearch {
		m.githubInputFocus = true
		cmd = m.githubInput.Focus()
		m.githubInput.CursorEnd()
	}
	m.commandPrevFilterActive = false
	m.commandPrevDockerHubSearch = false
	m.commandPrevGitHubSearch = false
	m.syncTable()
	return m, cmd
}

func (m Model) runCommand() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.commandInput.Value())
	if input == "" {
		return m.exitCommandMode()
	}

	// Hide command input after execution.
	m.commandActive = false
	m.commandInput.Blur()
	m.commandInput.SetValue("")
	m.commandMatches = nil
	m.commandIndex = 0
	m.commandError = ""
	m.commandPrevFilterActive = false
	m.commandPrevDockerHubSearch = false
	m.commandPrevGitHubSearch = false
	m.syncTable()

	cmdName, args := parseCommand(input)
	command, ok := resolveCommand(cmdName)
	if !ok {
		m.status = fmt.Sprintf("Unknown command: %s", cmdName)
		return m, nil
	}
	return command.Run(m, args)
}

func (m Model) switchContext(name string) (tea.Model, tea.Cmd) {
	index, ok := m.resolveContextIndex(name)
	if !ok {
		m.commandError = ""
		m.status = fmt.Sprintf("Unknown context: %s", name)
		return m, nil
	}
	return m.switchContextAt(index)
}

func (m Model) switchContextAt(index int) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(m.contexts) {
		m.commandError = ""
		m.status = "Invalid context selection"
		return m, nil
	}
	ctx := m.contexts[index]
	if ctx.Host == "" {
		m.contextSelectionError = fmt.Sprintf("Context %s has no registry configured", contextDisplayName(ctx, index))
		m.commandError = ""
		m.status = m.contextSelectionError
		return m, nil
	}

	m.commandActive = false
	m.commandInput.Blur()
	m.commandError = ""
	m.commandMatches = nil
	m.commandPrevFilterActive = false
	m.commandPrevDockerHubSearch = false
	m.commandPrevGitHubSearch = false
	m.contextSelectionActive = false
	m.contextSelectionRequired = false
	m.contextSelectionIndex = index
	m.contextSelectionError = ""

	m.context = contextDisplayName(ctx, index)
	m.registryHost = ctx.Host
	m.auth = ctx.Auth
	m.auth.Normalize()
	registry.ApplyAuthCache(&m.auth, m.registryHost)
	if m.auth.Kind == "registry_v2" && m.auth.RegistryV2.RefreshToken != "" {
		m.auth.RegistryV2.Remember = true
	}
	m.provider = registry.ProviderForAuth(m.auth)

	m.registryClient = nil
	m.authRequired = m.provider.NeedsAuthPrompt(m.auth)
	m.authError = ""
	m.authFocus = 0
	m.usernameInput.SetValue("")
	m.passwordInput.SetValue("")
	m.remember = false
	switch m.auth.Kind {
	case "registry_v2":
		m.usernameInput.SetValue(m.auth.RegistryV2.Username)
		m.remember = m.auth.RegistryV2.Remember
	case "harbor":
		m.usernameInput.SetValue(m.auth.Harbor.Username)
	}

	m.images = nil
	m.projects = nil
	m.tags = nil
	m.history = nil
	m.selectedProject = ""
	m.hasSelectedProject = false
	m.selectedImage = registry.Image{}
	m.hasSelectedImage = false
	m.selectedTag = registry.Tag{}
	m.hasSelectedTag = false
	m.focus = m.defaultFocus()
	m.status = fmt.Sprintf("Registry: %s", m.registryHost)
	m.dockerHubActive = false
	m.dockerHubInputFocus = false
	m.dockerHubInput.Blur()
	m.dockerHubLoading = false
	m.dockerHubImage = ""
	m.dockerHubTags = nil
	m.dockerHubNext = ""
	m.dockerHubRateLimit = registry.DockerHubRateLimit{}
	m.dockerHubRetryUntil = time.Time{}
	m.githubActive = false
	m.githubInputFocus = false
	m.githubInput.Blur()
	m.githubLoading = false
	m.githubImage = ""
	m.githubTags = nil
	m.githubNext = ""
	m.filterActive = false
	m.filterInput.SetValue("")

	if m.authRequired {
		cmd := m.usernameInput.Focus()
		m.syncTable()
		return m, cmd
	}

	m.syncTable()
	return m, initClientCmd(m.registryHost, m.auth, m.logger)
}

func parseCommand(input string) (string, []string) {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", nil
	}
	return strings.ToLower(fields[0]), fields[1:]
}

func commandToken(input string) string {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func contextNames(contexts []ContextOption) []string {
	if len(contexts) == 0 {
		return nil
	}
	names := make([]string, 0, len(contexts))
	for _, ctx := range contexts {
		if ctx.Name != "" {
			names = append(names, ctx.Name)
		}
	}
	return names
}

func contextDisplayName(ctx ContextOption, index int) string {
	if name := strings.TrimSpace(ctx.Name); name != "" {
		return name
	}
	if host := strings.TrimSpace(ctx.Host); host != "" {
		return host
	}
	return fmt.Sprintf("context-%d", index+1)
}
