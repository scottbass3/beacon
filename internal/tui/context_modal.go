package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lipglossv2 "github.com/charmbracelet/lipgloss/v2"

	"github.com/scottbass3/beacon/internal/config"
	"github.com/scottbass3/beacon/internal/registry"
)

type contextFormMode int

const (
	contextFormModeAdd contextFormMode = iota
	contextFormModeEdit
)

const (
	contextFormFocusName = iota
	contextFormFocusRegistry
	contextFormFocusKind
	contextFormFocusService
	contextFormFocusAnonymous
	contextFormFocusSecondaryButton
	contextFormFocusPrimaryButton
	contextFormFocusCount
)

func newContextInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.CharLimit = 256
	return input
}

func (m Model) contextSelectionHelpText() string {
	if m.contextSelectionRequired {
		return "up/down move  enter select  a add context  q quit"
	}
	return "up/down move  enter select  a add context  esc close  q quit"
}

func (m Model) openContextSelection(required bool) (tea.Model, tea.Cmd) {
	m.contextSelectionActive = true
	m.contextSelectionRequired = required
	m.contextSelectionError = ""
	if len(m.contexts) == 0 {
		m.contextSelectionIndex = 0
		m.status = "No contexts configured"
		m.syncTable()
		return m, nil
	}
	if current := m.currentContextIndex(); current >= 0 {
		m.contextSelectionIndex = current
	}
	m.syncTable()
	return m, nil
}

func (m Model) closeContextSelection() (tea.Model, tea.Cmd) {
	m.contextSelectionActive = false
	m.contextSelectionRequired = false
	m.contextSelectionError = ""
	m.syncTable()
	return m, nil
}

func (m Model) runContextCommand(args []string) (tea.Model, tea.Cmd) {
	if len(args) == 0 {
		return m.openContextSelection(false)
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "add":
		if len(args) != 1 {
			m.status = "Usage: :context add"
			return m, nil
		}
		return m.openContextFormAdd(false, false)
	case "remove", "rm", "delete":
		if len(args) < 2 {
			m.status = "Usage: :context remove <name>"
			return m, nil
		}
		return m.removeContextByName(strings.Join(args[1:], " "))
	case "edit":
		if len(args) < 2 {
			m.status = "Usage: :context edit <name>"
			return m, nil
		}
		return m.openContextFormEditByName(strings.Join(args[1:], " "))
	default:
		return m.switchContext(strings.Join(args, " "))
	}
}

func (m Model) openContextFormAdd(returnSelection, allowSkip bool) (tea.Model, tea.Cmd) {
	m.contextFormActive = true
	m.contextFormMode = contextFormModeAdd
	m.contextFormIndex = -1
	m.contextFormReturnSelection = returnSelection
	m.contextFormAllowSkip = allowSkip
	m.contextFormError = ""
	m.contextFormFocus = contextFormFocusName
	m.contextFormAnonymous = true
	m.contextFormNameInput.SetValue("")
	m.contextFormRegistryInput.SetValue("")
	m.contextFormKindInput.SetValue("registry_v2")
	m.contextFormServiceInput.SetValue("")
	if returnSelection {
		m.contextSelectionActive = false
		m.contextSelectionRequired = false
	}
	cmd := m.syncContextFormFocus()
	m.syncTable()
	return m, cmd
}

func (m Model) openContextFormEditByName(name string) (tea.Model, tea.Cmd) {
	index, ok := m.resolveContextIndex(name)
	if !ok {
		m.status = fmt.Sprintf("Unknown context: %s", name)
		return m, nil
	}
	return m.openContextFormEdit(index, false)
}

func (m Model) openContextFormEdit(index int, returnSelection bool) (tea.Model, tea.Cmd) {
	if index < 0 || index >= len(m.contexts) {
		m.status = "Invalid context selection"
		return m, nil
	}
	ctx := m.contexts[index]
	kind, ok := normalizeContextKind(ctx.Auth.Kind)
	if !ok {
		kind = "registry_v2"
	}
	anonymous := true
	service := ""
	switch kind {
	case "harbor":
		anonymous = ctx.Auth.Harbor.Anonymous
		service = ctx.Auth.Harbor.Service
	default:
		anonymous = ctx.Auth.RegistryV2.Anonymous
		service = ctx.Auth.RegistryV2.Service
	}

	m.contextFormActive = true
	m.contextFormMode = contextFormModeEdit
	m.contextFormIndex = index
	m.contextFormReturnSelection = returnSelection
	m.contextFormAllowSkip = false
	m.contextFormError = ""
	m.contextFormFocus = contextFormFocusName
	m.contextFormAnonymous = anonymous
	m.contextFormNameInput.SetValue(contextDisplayName(ctx, index))
	m.contextFormRegistryInput.SetValue(strings.TrimSpace(ctx.Host))
	m.contextFormKindInput.SetValue(kind)
	m.contextFormServiceInput.SetValue(strings.TrimSpace(service))
	if returnSelection {
		m.contextSelectionActive = false
		m.contextSelectionRequired = false
	}
	cmd := m.syncContextFormFocus()
	m.syncTable()
	return m, cmd
}

func (m *Model) syncContextFormFocus() tea.Cmd {
	m.contextFormNameInput.Blur()
	m.contextFormRegistryInput.Blur()
	m.contextFormKindInput.Blur()
	m.contextFormServiceInput.Blur()

	switch m.contextFormFocus {
	case contextFormFocusName:
		return m.contextFormNameInput.Focus()
	case contextFormFocusRegistry:
		return m.contextFormRegistryInput.Focus()
	case contextFormFocusKind:
		return m.contextFormKindInput.Focus()
	case contextFormFocusService:
		return m.contextFormServiceInput.Focus()
	default:
		return nil
	}
}

func (m Model) handleContextFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m.openQuitConfirm()
	case "q":
		return m.openQuitConfirm()
	case "esc":
		return m.cancelContextForm()
	case "tab", "down":
		m.contextFormFocus = m.nextContextFormFocus(m.contextFormFocus)
		return m, m.syncContextFormFocus()
	case "shift+tab", "up":
		m.contextFormFocus = m.prevContextFormFocus(m.contextFormFocus)
		return m, m.syncContextFormFocus()
	case "left", "h":
		if m.shouldSwapContextFormActions() {
			if m.contextFormFocus == contextFormFocusSecondaryButton {
				m.contextFormFocus = contextFormFocusPrimaryButton
				return m, m.syncContextFormFocus()
			}
		} else {
			if m.contextFormFocus == contextFormFocusPrimaryButton {
				m.contextFormFocus = contextFormFocusSecondaryButton
				return m, m.syncContextFormFocus()
			}
		}
	case "right", "l":
		if m.shouldSwapContextFormActions() {
			if m.contextFormFocus == contextFormFocusPrimaryButton {
				m.contextFormFocus = contextFormFocusSecondaryButton
				return m, m.syncContextFormFocus()
			}
		} else {
			if m.contextFormFocus == contextFormFocusSecondaryButton {
				m.contextFormFocus = contextFormFocusPrimaryButton
				return m, m.syncContextFormFocus()
			}
		}
	case " ":
		if m.contextFormFocus == contextFormFocusAnonymous {
			m.contextFormAnonymous = !m.contextFormAnonymous
			return m, nil
		}
	case "enter":
		switch m.contextFormFocus {
		case contextFormFocusSecondaryButton:
			return m.cancelContextForm()
		case contextFormFocusPrimaryButton:
			return m.submitContextForm()
		case contextFormFocusAnonymous:
			m.contextFormAnonymous = !m.contextFormAnonymous
			return m, nil
		default:
			m.contextFormFocus = m.nextContextFormFocus(m.contextFormFocus)
			return m, m.syncContextFormFocus()
		}
	}

	var cmd tea.Cmd
	switch m.contextFormFocus {
	case contextFormFocusName:
		m.contextFormNameInput, cmd = m.contextFormNameInput.Update(msg)
	case contextFormFocusRegistry:
		m.contextFormRegistryInput, cmd = m.contextFormRegistryInput.Update(msg)
	case contextFormFocusKind:
		m.contextFormKindInput, cmd = m.contextFormKindInput.Update(msg)
	case contextFormFocusService:
		m.contextFormServiceInput, cmd = m.contextFormServiceInput.Update(msg)
	}
	return m, cmd
}

func (m Model) cancelContextForm() (tea.Model, tea.Cmd) {
	allowSkip := m.contextFormAllowSkip && len(m.contexts) == 0 && strings.TrimSpace(m.registryHost) == ""
	returnSelection := m.contextFormReturnSelection
	m.deactivateContextForm()
	if returnSelection {
		m.contextSelectionActive = true
		m.contextSelectionRequired = false
	}
	if allowSkip {
		m.status = "No context selected. Use :context add to configure one."
	}
	m.syncTable()
	return m, nil
}

func (m *Model) deactivateContextForm() {
	m.contextFormActive = false
	m.contextFormMode = contextFormModeAdd
	m.contextFormIndex = -1
	m.contextFormReturnSelection = false
	m.contextFormAllowSkip = false
	m.contextFormError = ""
	m.contextFormFocus = contextFormFocusName
	m.contextFormNameInput.Blur()
	m.contextFormRegistryInput.Blur()
	m.contextFormKindInput.Blur()
	m.contextFormServiceInput.Blur()
}

func (m Model) renderContextFormModal() string {
	title := "Add Context"
	subtitle := "Enter context details."
	if m.contextFormMode == contextFormModeEdit {
		title = "Edit Context"
		subtitle = "Update context details."
	} else if m.contextFormAllowSkip && len(m.contexts) == 0 {
		subtitle = "Add a context now or continue without one."
	}

	name := m.contextFormNameInput.View()
	registryHost := m.contextFormRegistryInput.View()
	kind := m.contextFormKindInput.View()
	service := m.contextFormServiceInput.View()

	if m.contextFormFocus == contextFormFocusName {
		name = modalInputFocusStyle.Render(name)
	} else {
		name = modalInputStyle.Render(name)
	}
	if m.contextFormFocus == contextFormFocusRegistry {
		registryHost = modalInputFocusStyle.Render(registryHost)
	} else {
		registryHost = modalInputStyle.Render(registryHost)
	}
	if m.contextFormFocus == contextFormFocusKind {
		kind = modalInputFocusStyle.Render(kind)
	} else {
		kind = modalInputStyle.Render(kind)
	}
	if m.contextFormFocus == contextFormFocusService {
		service = modalInputFocusStyle.Render(service)
	} else {
		service = modalInputStyle.Render(service)
	}

	anonymous := "[ ] Anonymous"
	if m.contextFormAnonymous {
		anonymous = "[x] Anonymous"
	}
	if m.contextFormFocus == contextFormFocusAnonymous {
		anonymous = modalFocusStyle.Render(anonymous)
	} else {
		anonymous = modalLabelStyle.Render(anonymous)
	}

	secondaryLabel := "Cancel"
	if m.contextFormAllowSkip && len(m.contexts) == 0 {
		secondaryLabel = "Continue without context"
	}
	secondary := modalButtonStyle.Render(secondaryLabel)
	if m.contextFormFocus == contextFormFocusSecondaryButton {
		secondary = modalButtonFocusStyle.Render(secondaryLabel)
	}

	primaryLabel := "Add Context"
	if m.contextFormMode == contextFormModeEdit {
		primaryLabel = "Save Context"
	}
	primary := modalButtonStyle.Render(primaryLabel)
	if m.contextFormFocus == contextFormFocusPrimaryButton {
		primary = modalButtonFocusStyle.Render(primaryLabel)
	}
	leftButton := lipglossv2.NewStyle().MarginRight(2).Render(secondary)
	rightButton := primary
	if m.shouldSwapContextFormActions() {
		leftButton = lipglossv2.NewStyle().MarginRight(2).Render(primary)
		rightButton = secondary
	}
	buttonRow := lipglossv2.JoinHorizontal(
		lipglossv2.Top,
		leftButton,
		rightButton,
	)

	lines := []string{
		modalTitleStyle.Render(title),
		modalLabelStyle.Render(subtitle),
		modalDividerStyle.Render(strings.Repeat("─", 24)),
	}
	if m.contextFormError != "" {
		lines = append(lines, modalErrorStyle.Render(m.contextFormError))
	}
	lines = append(lines,
		"",
		modalLabelStyle.Render("Name"),
		name,
		modalLabelStyle.Render("Registry"),
		registryHost,
		modalLabelStyle.Render("Kind"),
		kind,
		modalLabelStyle.Render("Service"),
		service,
		anonymous,
		"",
		buttonRow,
		"",
		modalHelpStyle.Render("tab/shift+tab move  space toggle anonymous  enter select  esc cancel"),
	)
	return m.renderModalCard(strings.Join(lines, "\n"), 88)
}

func (m Model) submitContextForm() (tea.Model, tea.Cmd) {
	name := strings.TrimSpace(m.contextFormNameInput.Value())
	registryHost := strings.TrimSpace(m.contextFormRegistryInput.Value())
	kindInput := strings.TrimSpace(m.contextFormKindInput.Value())
	service := strings.TrimSpace(m.contextFormServiceInput.Value())

	if name == "" {
		m.contextFormError = "Context name is required"
		return m, nil
	}
	if registryHost == "" {
		m.contextFormError = "Registry is required"
		return m, nil
	}
	kind, ok := normalizeContextKind(kindInput)
	if !ok {
		m.contextFormError = "Kind must be registry_v2 or harbor"
		return m, nil
	}

	nameKey := strings.ToLower(name)
	if existing, found := m.contextNameIndex[nameKey]; found && (m.contextFormMode != contextFormModeEdit || existing != m.contextFormIndex) {
		m.contextFormError = fmt.Sprintf("Context %q already exists", name)
		return m, nil
	}

	auth := registry.Auth{Kind: kind}
	switch kind {
	case "harbor":
		auth.Harbor.Anonymous = m.contextFormAnonymous
		auth.Harbor.Service = service
	default:
		auth.RegistryV2.Anonymous = m.contextFormAnonymous
		auth.RegistryV2.Service = service
	}
	auth.Normalize()

	context := ContextOption{
		Name: name,
		Host: registryHost,
		Auth: auth,
	}

	updated := make([]ContextOption, 0, len(m.contexts)+1)
	updated = append(updated, m.contexts...)
	targetIndex := len(updated)
	if m.contextFormMode == contextFormModeEdit {
		if m.contextFormIndex < 0 || m.contextFormIndex >= len(updated) {
			m.contextFormError = "Invalid context selection"
			return m, nil
		}
		targetIndex = m.contextFormIndex
		updated[targetIndex] = context
	} else {
		updated = append(updated, context)
	}

	if err := m.persistContextOptions(updated); err != nil {
		m.contextFormError = err.Error()
		return m, nil
	}

	oldCount := len(m.contexts)
	activeIndex := m.currentContextIndex()
	mode := m.contextFormMode
	returnSelection := m.contextFormReturnSelection

	m.contexts = updated
	m.rebuildContextNameIndex()
	m.contextSelectionIndex = clampInt(targetIndex, 0, maxInt(0, len(m.contexts)-1))
	m.contextSelectionError = ""

	m.deactivateContextForm()
	if returnSelection {
		m.contextSelectionActive = true
		m.contextSelectionRequired = false
	}

	switch mode {
	case contextFormModeAdd:
		m.status = fmt.Sprintf("Added context %s", name)
		if oldCount == 0 || strings.TrimSpace(m.registryHost) == "" {
			m.contextSelectionActive = false
			m.contextSelectionRequired = false
			return m.switchContextAt(targetIndex)
		}
		m.syncTable()
		return m, nil
	case contextFormModeEdit:
		m.status = fmt.Sprintf("Updated context %s", name)
		if activeIndex == targetIndex {
			m.contextSelectionActive = false
			m.contextSelectionRequired = false
			return m.switchContextAt(targetIndex)
		}
		m.syncTable()
		return m, nil
	default:
		m.syncTable()
		return m, nil
	}
}

func normalizeContextKind(input string) (string, bool) {
	kind := strings.ToLower(strings.TrimSpace(input))
	switch kind {
	case "registry", "v2", "registry_v2":
		return "registry_v2", true
	case "harbor":
		return "harbor", true
	default:
		return "", false
	}
}

func (m Model) resolveContextIndex(name string) (int, bool) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return 0, false
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
	index, ok := m.resolveContextIndex(name)
	if !ok {
		m.status = fmt.Sprintf("Unknown context: %s", name)
		return m, nil
	}

	currentIndex := m.currentContextIndex()
	removed := contextDisplayName(m.contexts[index], index)

	updated := make([]ContextOption, 0, len(m.contexts)-1)
	updated = append(updated, m.contexts[:index]...)
	updated = append(updated, m.contexts[index+1:]...)

	if err := m.persistContextOptions(updated); err != nil {
		m.status = err.Error()
		return m, nil
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
	path := strings.TrimSpace(m.configPath)
	if path == "" {
		return fmt.Errorf("cannot save contexts: config path is not set")
	}

	cfg := config.Config{Contexts: make([]config.Context, 0, len(contexts))}
	for _, ctx := range contexts {
		cfg.Contexts = append(cfg.Contexts, contextOptionToConfigContext(ctx))
	}
	if err := config.Save(path, cfg); err != nil {
		return fmt.Errorf("failed to save contexts: %w", err)
	}
	return nil
}

func contextOptionToConfigContext(ctx ContextOption) config.Context {
	kind, ok := normalizeContextKind(ctx.Auth.Kind)
	if !ok {
		kind = "registry_v2"
	}
	out := config.Context{
		Name:     strings.TrimSpace(ctx.Name),
		Registry: strings.TrimSpace(ctx.Host),
		Kind:     kind,
	}
	switch kind {
	case "harbor":
		out.Anonymous = ctx.Auth.Harbor.Anonymous
		out.Service = strings.TrimSpace(ctx.Auth.Harbor.Service)
	default:
		out.Anonymous = ctx.Auth.RegistryV2.Anonymous
		out.Service = strings.TrimSpace(ctx.Auth.RegistryV2.Service)
	}
	return out
}

func (m Model) shouldSwapContextFormActions() bool {
	return m.contextFormAllowSkip && len(m.contexts) == 0 && m.contextFormMode == contextFormModeAdd
}

func (m Model) nextContextFormFocus(current int) int {
	if !m.shouldSwapContextFormActions() {
		return (current + 1) % contextFormFocusCount
	}
	switch current {
	case contextFormFocusName:
		return contextFormFocusRegistry
	case contextFormFocusRegistry:
		return contextFormFocusKind
	case contextFormFocusKind:
		return contextFormFocusService
	case contextFormFocusService:
		return contextFormFocusAnonymous
	case contextFormFocusAnonymous:
		return contextFormFocusPrimaryButton
	case contextFormFocusPrimaryButton:
		return contextFormFocusSecondaryButton
	case contextFormFocusSecondaryButton:
		return contextFormFocusName
	default:
		return contextFormFocusName
	}
}

func (m Model) prevContextFormFocus(current int) int {
	if !m.shouldSwapContextFormActions() {
		current--
		if current < 0 {
			return contextFormFocusCount - 1
		}
		return current
	}
	switch current {
	case contextFormFocusName:
		return contextFormFocusSecondaryButton
	case contextFormFocusRegistry:
		return contextFormFocusName
	case contextFormFocusKind:
		return contextFormFocusRegistry
	case contextFormFocusService:
		return contextFormFocusKind
	case contextFormFocusAnonymous:
		return contextFormFocusService
	case contextFormFocusPrimaryButton:
		return contextFormFocusAnonymous
	case contextFormFocusSecondaryButton:
		return contextFormFocusPrimaryButton
	default:
		return contextFormFocusName
	}
}
