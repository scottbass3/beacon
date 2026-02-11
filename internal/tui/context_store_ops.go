package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/contextstore"
	"github.com/scottbass3/beacon/internal/registry"
)

func normalizeContextKind(input string) (string, bool) {
	return contextstore.NormalizeKindInput(input)
}

func (m Model) resolveContextIndex(name string) (int, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return 0, false
	}
	if index, ok := contextstore.ResolveByName(contextOptionsToStoredContexts(m.contexts), trimmed); ok && index >= 0 && index < len(m.contexts) {
		return index, true
	}
	if index, ok := m.contextNameIndex[strings.ToLower(trimmed)]; ok && index >= 0 && index < len(m.contexts) {
		return index, true
	}
	for i, ctx := range m.contexts {
		if strings.EqualFold(contextDisplayName(ctx, i), trimmed) {
			return i, true
		}
		if strings.EqualFold(strings.TrimSpace(ctx.Host), trimmed) {
			return i, true
		}
	}
	return 0, false
}

func (m *Model) rebuildContextNameIndex() {
	index := make(map[string]int, len(m.contexts))
	for i, ctx := range m.contexts {
		name := strings.ToLower(strings.TrimSpace(ctx.Name))
		if name == "" {
			continue
		}
		index[name] = i
	}
	m.contextNameIndex = index
}

func (m Model) currentContextIndex() int {
	if len(m.contexts) == 0 {
		return -1
	}
	if m.contextSelectionIndex >= 0 && m.contextSelectionIndex < len(m.contexts) {
		ctx := m.contexts[m.contextSelectionIndex]
		if strings.EqualFold(strings.TrimSpace(ctx.Host), strings.TrimSpace(m.registryHost)) {
			return m.contextSelectionIndex
		}
	}
	for i, ctx := range m.contexts {
		if strings.EqualFold(strings.TrimSpace(ctx.Host), strings.TrimSpace(m.registryHost)) {
			return i
		}
		if strings.EqualFold(contextDisplayName(ctx, i), strings.TrimSpace(m.context)) {
			return i
		}
	}
	return -1
}

func (m Model) removeContextByName(name string) (tea.Model, tea.Cmd) {
	serviceManager := contextstore.NewService(m.configPath)
	updatedStored, removedContext, index, err := serviceManager.RemoveByName(contextOptionsToStoredContexts(m.contexts), name)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	currentIndex := m.currentContextIndex()
	if err := serviceManager.Save(updatedStored); err != nil {
		m.status = fmt.Sprintf("failed to save contexts: %v", err)
		return m, nil
	}
	updated := storedContextsToContextOptions(updatedStored)
	removed := strings.TrimSpace(removedContext.Name)
	if removed == "" {
		removed = strings.TrimSpace(removedContext.Host)
	}
	if removed == "" {
		removed = fmt.Sprintf("context-%d", index+1)
	}

	m.contexts = updated
	m.rebuildContextNameIndex()
	m.contextSelectionError = ""

	if len(m.contexts) == 0 {
		m.clearRegistryContext()
		m.status = fmt.Sprintf("Removed context %s. No contexts remain.", removed)
		m.syncTable()
		return m, nil
	}

	if currentIndex == index {
		nextIndex := index
		if nextIndex >= len(m.contexts) {
			nextIndex = len(m.contexts) - 1
		}
		m.contextSelectionIndex = nextIndex
		return m.switchContextAt(nextIndex)
	}

	if currentIndex > index {
		currentIndex--
	}
	if currentIndex >= 0 && currentIndex < len(m.contexts) {
		m.contextSelectionIndex = currentIndex
	} else if m.contextSelectionIndex >= len(m.contexts) {
		m.contextSelectionIndex = len(m.contexts) - 1
	}
	m.status = fmt.Sprintf("Removed context %s", removed)
	m.syncTable()
	return m, nil
}

func (m *Model) clearRegistryContext() {
	m.context = ""
	m.registryHost = ""
	m.registryClient = nil
	m.auth = registry.Auth{}
	m.auth.Normalize()
	m.provider = registry.ProviderForAuth(m.auth)
	m.authRequired = false
	m.authError = ""
	m.authFocus = 0
	m.usernameInput.SetValue("")
	m.passwordInput.SetValue("")
	m.usernameInput.Blur()
	m.passwordInput.Blur()
	m.remember = false

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

	m.contextSelectionActive = false
	m.contextSelectionRequired = false
	m.contextSelectionIndex = 0
	m.contextSelectionError = ""

	m.filterActive = false
	m.filterInput.SetValue("")
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
}

func (m Model) persistContextOptions(contexts []ContextOption) error {
	service := contextstore.NewService(m.configPath)
	if strings.TrimSpace(contextstore.New(m.configPath).Path()) == "" {
		return fmt.Errorf("cannot save contexts: config path is not set")
	}
	if err := service.Save(contextOptionsToStoredContexts(contexts)); err != nil {
		return fmt.Errorf("failed to save contexts: %w", err)
	}
	return nil
}

func contextOptionsToStoredContexts(contexts []ContextOption) []contextstore.Context {
	if len(contexts) == 0 {
		return nil
	}
	stored := make([]contextstore.Context, 0, len(contexts))
	for _, ctx := range contexts {
		stored = append(stored, contextOptionToStoredContext(ctx))
	}
	return stored
}

func storedContextToContextOption(ctx contextstore.Context) ContextOption {
	auth := ctx.Auth
	auth.Normalize()
	return ContextOption{
		Name: strings.TrimSpace(ctx.Name),
		Host: strings.TrimSpace(ctx.Host),
		Auth: auth,
	}
}

func storedContextsToContextOptions(contexts []contextstore.Context) []ContextOption {
	if len(contexts) == 0 {
		return nil
	}
	out := make([]ContextOption, 0, len(contexts))
	for _, ctx := range contexts {
		out = append(out, storedContextToContextOption(ctx))
	}
	return out
}

func contextOptionToStoredContext(ctx ContextOption) contextstore.Context {
	kind, ok := normalizeContextKind(ctx.Auth.Kind)
	if !ok {
		kind = "registry_v2"
	}
	auth := registry.Auth{Kind: kind}
	switch kind {
	case "harbor":
		auth.Harbor.Anonymous = ctx.Auth.Harbor.Anonymous
		auth.Harbor.Service = strings.TrimSpace(ctx.Auth.Harbor.Service)
	default:
		auth.RegistryV2.Anonymous = ctx.Auth.RegistryV2.Anonymous
		auth.RegistryV2.Service = strings.TrimSpace(ctx.Auth.RegistryV2.Service)
	}
	auth.Normalize()
	return contextstore.Context{
		Name: strings.TrimSpace(ctx.Name),
		Host: strings.TrimSpace(ctx.Host),
		Auth: auth,
	}
}
