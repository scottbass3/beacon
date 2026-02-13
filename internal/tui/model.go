package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

func NewModel(registryHost string, auth registry.Auth, logger registry.RequestLogger, debug bool, logCh <-chan string, contexts []ContextOption, currentContext, configPath string) Model {
	status := "Registry not configured"
	if registryHost != "" {
		status = fmt.Sprintf("Registry: %s", registryHost)
	}
	if strings.TrimSpace(currentContext) == "" && len(contexts) > 0 && registryHost == "" {
		currentContext = contexts[0].Name
	}

	filter := textinput.New()
	filter.Prompt = "/ "
	filter.Placeholder = "filter"
	filter.CharLimit = 64
	filter.Blur()

	tbl := table.New()
	tbl.SetStyles(tableStyles())
	tbl.SetHeight(defaultTableHeight)
	tbl.Focus()

	dockerHubInput := textinput.New()
	dockerHubInput.Prompt = "Search: "
	dockerHubInput.Placeholder = "library/nginx"
	dockerHubInput.CharLimit = 128
	dockerHubInput.Blur()

	githubInput := textinput.New()
	githubInput.Prompt = "Search: "
	githubInput.Placeholder = "owner/image"
	githubInput.CharLimit = 128
	githubInput.Blur()

	commandInput := textinput.New()
	commandInput.Prompt = ":"
	commandInput.Placeholder = "help | context add | dockerhub | github"
	commandInput.CharLimit = 64
	commandInput.Blur()

	contextNameInput := newContextInput("name")
	contextRegistryInput := newContextInput("https://registry.example.com")
	contextKindInput := newContextInput("registry_v2 | harbor")
	contextServiceInput := newContextInput("optional service")
	contextKindInput.SetValue("registry_v2")
	contextNameInput.Blur()
	contextRegistryInput.Blur()
	contextKindInput.Blur()
	contextServiceInput.Blur()

	auth.Normalize()
	if registryHost != "" {
		registry.ApplyAuthCache(&auth, registryHost)
		if auth.Kind == "registry_v2" && auth.RegistryV2.RefreshToken != "" {
			auth.RegistryV2.Remember = true
		}
	}
	provider := registry.ProviderForAuth(auth)

	username := textinput.New()
	username.Prompt = ""
	username.Placeholder = "username"
	username.CharLimit = 128
	username.Blur()

	password := textinput.New()
	password.Prompt = ""
	password.Placeholder = "password"
	password.CharLimit = 128
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'
	password.Blur()

	remember := false
	switch auth.Kind {
	case "registry_v2":
		username.SetValue(auth.RegistryV2.Username)
		remember = auth.RegistryV2.Remember
	case "harbor":
		username.SetValue(auth.Harbor.Username)
	}
	authRequired := provider.NeedsAuthPrompt(auth)

	contextIndex := make(map[string]int, len(contexts))
	for i, ctx := range contexts {
		contextIndex[strings.ToLower(ctx.Name)] = i
	}
	contextSelectionActive := registryHost == "" && len(contexts) > 1
	contextSelectionRequired := contextSelectionActive
	contextFormStartup := registryHost == "" && len(contexts) == 0
	contextSelectionIndex := 0
	if i, ok := contextIndex[strings.ToLower(strings.TrimSpace(currentContext))]; ok {
		contextSelectionIndex = i
	}
	if contextSelectionActive {
		status = "Select context to continue"
	} else if contextFormStartup {
		status = "No contexts configured. Add one or continue without context."
		contextNameInput.Focus()
	} else if authRequired {
		username.Focus()
	}
	displayContext := currentContext
	if contextSelectionActive {
		displayContext = ""
	}

	return Model{
		status: status,
		focus: func() Focus {
			if provider.TableSpec().SupportsProjects {
				return FocusProjects
			}
			return FocusImages
		}(),
		context: displayContext,
		contextSelectionState: contextSelectionState{
			contextSelectionActive:   contextSelectionActive,
			contextSelectionRequired: contextSelectionRequired,
			contextSelectionIndex:    contextSelectionIndex,
		},
		contextFormState: contextFormState{
			contextFormActive:        contextFormStartup,
			contextFormMode:          contextFormModeAdd,
			contextFormIndex:         -1,
			contextFormAllowSkip:     contextFormStartup,
			contextFormFocus:         contextFormFocusName,
			contextFormNameInput:     contextNameInput,
			contextFormRegistryInput: contextRegistryInput,
			contextFormKindInput:     contextKindInput,
			contextFormServiceInput:  contextServiceInput,
			contextFormAnonymous:     true,
		},
		configPath:     configPath,
		registryHost:   registryHost,
		auth:           auth,
		provider:       provider,
		authRequired:   authRequired,
		authFocus:      0,
		usernameInput:  username,
		passwordInput:  password,
		remember:       remember,
		filterInput:    filter,
		table:          tbl,
		dockerHubInput: dockerHubInput,
		githubInput:    githubInput,
		commandState: commandState{
			commandInput: commandInput,
		},
		contexts:         contexts,
		contextNameIndex: contextIndex,
		debug:            debug,
		logCh:            logCh,
		logMax:           maxLogLines,
		logger:           logger,
	}
}

func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.registryHost != "" && !m.authRequired && !m.isContextSelectionActive() {
		cmds = append(cmds, initClientCmd(m.registryHost, m.auth, m.logger))
	}
	if m.logCh != nil {
		cmds = append(cmds, listenLogs(m.logCh))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKeyMsg(msg)
	case tea.MouseMsg:
		return m.updateMouseMsg(msg)
	case tea.WindowSizeMsg:
		return m.updateWindowSizeMsg(msg)
	case imagesMsg:
		return m.updateImagesMsg(msg)
	case projectsMsg:
		return m.updateProjectsMsg(msg)
	case projectImagesMsg:
		return m.updateProjectImagesMsg(msg)
	case tagsMsg:
		return m.updateTagsMsg(msg)
	case historyMsg:
		return m.updateHistoryMsg(msg)
	case dockerPullMsg:
		return m.updateDockerPullMsg(msg)
	case dockerHubTagsMsg:
		return m.updateDockerHubTagsMsg(msg)
	case githubTagsMsg:
		return m.updateGitHubTagsMsg(msg)
	case logMsg:
		return m.updateLogMsg(msg)
	case initClientMsg:
		return m.updateInitClientMsg(msg)
	}

	return m, nil
}

func (m Model) View() string {
	view := m.renderApp()
	if m.isContextSelectionActive() {
		view = m.renderModal(view, m.renderContextSelectionModal())
	}
	if m.isContextFormActive() {
		view = m.renderModal(view, m.renderContextFormModal())
	}
	if m.isAuthModalActive() {
		view = m.renderModal(view, m.renderAuthModal())
	}
	if m.isConfirmModalActive() {
		view = m.renderModal(view, m.renderConfirmModal())
	}
	return view
}
