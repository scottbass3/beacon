package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/contextstore"
	"github.com/scottbass3/beacon/internal/registry"
)

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
	kind, ok := contextstore.NormalizeKindInput(kindInput)
	if !ok {
		m.contextFormError = "Kind must be registry_v2 or harbor"
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

	candidate := contextstore.Context{
		Name: name,
		Host: registryHost,
		Auth: auth,
	}

	serviceManager := contextstore.NewService(m.configPath)
	existing := contextOptionsToStoredContexts(m.contexts)
	var (
		updatedStored []contextstore.Context
		targetIndex   int
		err           error
	)
	if m.contextFormMode == contextFormModeEdit {
		targetIndex = m.contextFormIndex
		updatedStored, err = serviceManager.Edit(existing, targetIndex, candidate)
	} else {
		updatedStored, targetIndex, err = serviceManager.Add(existing, candidate)
	}
	if err != nil {
		m.contextFormError = err.Error()
		return m, nil
	}
	if err := serviceManager.Save(updatedStored); err != nil {
		m.contextFormError = fmt.Sprintf("failed to save contexts: %v", err)
		return m, nil
	}
	updated := storedContextsToContextOptions(updatedStored)

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
