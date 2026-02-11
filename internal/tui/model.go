package tui

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	lipglossv2 "github.com/charmbracelet/lipgloss/v2"

	"github.com/scottbass3/beacon/internal/registry"
)

type Focus int

const (
	FocusProjects Focus = iota
	FocusImages
	FocusTags
	FocusHistory
	FocusDockerHubTags
	FocusGitHubTags
)

type confirmAction int

const (
	confirmActionNone confirmAction = iota
	confirmActionQuit
)

const (
	defaultTableHeight      = 10
	minTableHeight          = 1
	maxLogLines             = 25
	maxVisibleLogs          = 5
	maxFilterWidth          = 40
	tableChromeLines        = 2
	mainSectionTitleLines   = 1
	mainSectionBorderLines  = 2
	mainSectionHChromeChars = 4
	defaultRenderWidth      = 80
)

type Model struct {
	width  int
	height int

	status  string
	focus   Focus
	context string

	contextSelectionActive   bool
	contextSelectionRequired bool
	contextSelectionIndex    int
	contextSelectionError    string

	contextFormActive          bool
	contextFormMode            contextFormMode
	contextFormIndex           int
	contextFormReturnSelection bool
	contextFormAllowSkip       bool
	contextFormError           string
	contextFormFocus           int
	contextFormNameInput       textinput.Model
	contextFormRegistryInput   textinput.Model
	contextFormKindInput       textinput.Model
	contextFormServiceInput    textinput.Model
	contextFormAnonymous       bool

	confirmAction  confirmAction
	confirmTitle   string
	confirmMessage string
	confirmFocus   int

	configPath string

	registryHost   string
	registryClient registry.Client
	auth           registry.Auth
	provider       registry.Provider
	authRequired   bool
	authError      string
	authFocus      int
	usernameInput  textinput.Model
	passwordInput  textinput.Model
	remember       bool
	logger         registry.RequestLogger

	images   []registry.Image
	projects []projectInfo
	tags     []registry.Tag
	history  []registry.HistoryEntry

	selectedProject    string
	hasSelectedProject bool
	selectedImage      registry.Image
	hasSelectedImage   bool
	selectedTag        registry.Tag
	hasSelectedTag     bool

	filterActive bool
	filterInput  textinput.Model

	table table.Model

	dockerHubActive     bool
	dockerHubPrevFocus  Focus
	dockerHubPrevStatus string
	dockerHubInput      textinput.Model
	dockerHubInputFocus bool
	dockerHubImage      string
	dockerHubTags       []registry.Tag
	dockerHubNext       string
	dockerHubRateLimit  registry.DockerHubRateLimit
	dockerHubRetryUntil time.Time
	dockerHubLoading    bool

	githubActive     bool
	githubPrevFocus  Focus
	githubPrevStatus string
	githubInput      textinput.Model
	githubInputFocus bool
	githubImage      string
	githubTags       []registry.Tag
	githubNext       string
	githubLoading    bool

	commandActive              bool
	commandInput               textinput.Model
	commandMatches             []string
	commandIndex               int
	commandError               string
	commandPrevFilterActive    bool
	commandPrevDockerHubSearch bool
	commandPrevGitHubSearch    bool
	helpActive                 bool
	contexts                   []ContextOption
	contextNameIndex           map[string]int
	tableColumns               []table.Column

	debug  bool
	logCh  <-chan string
	logs   []string
	logMax int

	loadingCount int
}

type imagesMsg struct {
	images []registry.Image
	err    error
}

type projectsMsg struct {
	projects []registry.Project
	err      error
}

type projectImagesMsg struct {
	project string
	images  []registry.Image
	err     error
}

type tagsMsg struct {
	tags []registry.Tag
	err  error
}

type historyMsg struct {
	history []registry.HistoryEntry
	err     error
}

type dockerHubTagsMsg struct {
	tags       []registry.Tag
	image      string
	next       string
	rateLimit  registry.DockerHubRateLimit
	appendPage bool
	retryAfter time.Duration
	err        error
}

type githubTagsMsg struct {
	tags       []registry.Tag
	image      string
	next       string
	appendPage bool
	err        error
}

type projectInfo struct {
	Name       string
	ImageCount int
}

type helpEntry struct {
	Keys   string
	Action string
}

type commandHelp struct {
	Command string
	Usage   string
}

type initClientMsg struct {
	client registry.Client
	err    error
}

type logMsg string

var (
	colorPrimary   = lipgloss.Color("39")
	colorAccent    = lipgloss.Color("214")
	colorMuted     = lipgloss.Color("244")
	colorSelected  = lipgloss.Color("16")
	colorBorder    = lipgloss.Color("74")
	colorSurface   = lipgloss.Color("236")
	colorSurface2  = lipgloss.Color("234")
	colorTitleText = lipgloss.Color("230")
	colorSuccess   = lipgloss.Color("78")
)

var (
	modalColorPrimary  = lipglossv2.Color("39")
	modalColorAccent   = lipglossv2.Color("214")
	modalColorMuted    = lipglossv2.Color("244")
	modalColorBorder   = lipglossv2.Color("74")
	modalColorSurface  = lipglossv2.Color("236")
	modalColorSurface2 = lipglossv2.Color("234")
	modalColorTitle    = lipglossv2.Color("230")
	modalColorDanger   = lipglossv2.Color("196")
)

var (
	titleStyle             = lipgloss.NewStyle().Foreground(colorTitleText).Background(colorPrimary).Bold(true).Padding(0, 1).MarginRight(1)
	statusStyle            = lipgloss.NewStyle().Foreground(colorTitleText).Background(colorSurface2).Padding(0, 1)
	statusLoadingStyle     = lipgloss.NewStyle().Foreground(colorSurface2).Background(colorSuccess).Bold(true).Padding(0, 1)
	metaLabelStyle         = lipgloss.NewStyle().Foreground(colorMuted).Bold(true).MarginRight(1)
	metaValueStyle         = lipgloss.NewStyle().Foreground(colorTitleText).MarginRight(2)
	modeInputStyle         = lipgloss.NewStyle().Foreground(colorAccent).Background(colorSurface2).Padding(0, 1)
	shortcutHintStyle      = lipgloss.NewStyle().Foreground(colorMuted)
	helpHeadingStyle       = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	helpItemStyle          = lipgloss.NewStyle().Foreground(colorTitleText)
	helpFooterStyle        = lipgloss.NewStyle().Foreground(colorMuted)
	emptyStyle             = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	mainSectionStyle       = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)
	mainSectionTitleStyle  = lipgloss.NewStyle().Foreground(colorSurface2).Background(colorAccent).Bold(true).Padding(0, 2)
	mainSectionTitleLine   = lipgloss.NewStyle()
	topSectionStyle        = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)
	logTitleStyle          = lipgloss.NewStyle().Foreground(colorTitleText).Background(colorPrimary).Bold(true).Padding(0, 1)
	logBoxStyle            = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Background(colorSurface).Padding(0, 1)
	modalBackdropStyle     = lipglossv2.NewStyle().Foreground(modalColorMuted).Background(modalColorSurface2).Faint(true)
	modalPanelStyle        = lipglossv2.NewStyle().BorderStyle(lipglossv2.DoubleBorder()).BorderForeground(modalColorBorder).Background(modalColorSurface).Padding(1, 2)
	modalTitleStyle        = lipglossv2.NewStyle().Foreground(modalColorPrimary).Bold(true)
	modalLabelStyle        = lipglossv2.NewStyle().Foreground(modalColorMuted)
	modalErrorStyle        = lipglossv2.NewStyle().Foreground(modalColorDanger).Bold(true)
	modalInputStyle        = lipglossv2.NewStyle().Foreground(modalColorTitle).Background(modalColorSurface2).BorderStyle(lipglossv2.NormalBorder()).BorderForeground(modalColorMuted).Padding(0, 1)
	modalInputFocusStyle   = lipglossv2.NewStyle().Foreground(modalColorTitle).Background(modalColorSurface2).BorderStyle(lipglossv2.NormalBorder()).BorderForeground(modalColorAccent).Bold(true).Padding(0, 1)
	modalFocusStyle        = lipglossv2.NewStyle().Foreground(modalColorAccent).Bold(true)
	modalButtonStyle       = lipglossv2.NewStyle().Foreground(modalColorMuted).Background(modalColorSurface2).BorderStyle(lipglossv2.RoundedBorder()).BorderForeground(modalColorMuted).BorderBackground(modalColorSurface).Padding(0, 1)
	modalButtonFocusStyle  = lipglossv2.NewStyle().Foreground(modalColorSurface2).Background(modalColorAccent).BorderStyle(lipglossv2.RoundedBorder()).BorderForeground(modalColorAccent).BorderBackground(modalColorSurface).Bold(true).Padding(0, 1)
	modalDangerButtonStyle = lipglossv2.NewStyle().Foreground(modalColorDanger).Background(modalColorSurface2).BorderStyle(lipglossv2.RoundedBorder()).BorderForeground(modalColorDanger).BorderBackground(modalColorSurface).Padding(0, 1)
	modalDangerFocusStyle  = lipglossv2.NewStyle().Foreground(modalColorSurface2).Background(modalColorDanger).BorderStyle(lipglossv2.RoundedBorder()).BorderForeground(modalColorDanger).BorderBackground(modalColorSurface).Bold(true).Padding(0, 1)
	modalOptionStyle       = lipglossv2.NewStyle().Foreground(modalColorTitle).Background(modalColorSurface2).BorderStyle(lipglossv2.NormalBorder()).BorderForeground(modalColorMuted).BorderBackground(modalColorSurface).Padding(0, 1)
	modalOptionFocusStyle  = lipglossv2.NewStyle().Foreground(modalColorSurface2).Background(modalColorAccent).BorderStyle(lipglossv2.NormalBorder()).BorderForeground(modalColorAccent).BorderBackground(modalColorSurface).Bold(true).Padding(0, 1)
	modalOptionMutedStyle  = lipglossv2.NewStyle().Foreground(modalColorMuted)
	modalOptionErrorStyle  = lipglossv2.NewStyle().Foreground(modalColorDanger).Faint(true)
	modalHelpStyle         = lipglossv2.NewStyle().Foreground(modalColorMuted)
	modalDividerStyle      = lipglossv2.NewStyle().Foreground(modalColorBorder)
)

type ContextOption struct {
	Name string
	Host string
	Auth registry.Auth
}

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
		context:                  displayContext,
		contextSelectionActive:   contextSelectionActive,
		contextSelectionRequired: contextSelectionRequired,
		contextSelectionIndex:    contextSelectionIndex,
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
		configPath:               configPath,
		registryHost:             registryHost,
		auth:                     auth,
		provider:                 provider,
		authRequired:             authRequired,
		authFocus:                0,
		usernameInput:            username,
		passwordInput:            password,
		remember:                 remember,
		filterInput:              filter,
		table:                    tbl,
		dockerHubInput:           dockerHubInput,
		githubInput:              githubInput,
		commandInput:             commandInput,
		contexts:                 contexts,
		contextNameIndex:         contextIndex,
		debug:                    debug,
		logCh:                    logCh,
		logMax:                   maxLogLines,
		logger:                   logger,
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
		if !m.commandActive && (msg.String() == ":" || (len(msg.Runes) == 1 && msg.Runes[0] == ':')) {
			return m.enterCommandMode()
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncTable()
	case imagesMsg:
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
	case projectsMsg:
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
	case projectImagesMsg:
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
	case tagsMsg:
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
	case historyMsg:
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
	case dockerHubTagsMsg:
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
	case githubTagsMsg:
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
	case logMsg:
		m.appendLog(string(msg))
		m.syncTable()
		if m.logCh != nil {
			return m, listenLogs(m.logCh)
		}
	case initClientMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("Error initializing registry: %v", msg.err)
			m.authError = msg.err.Error()
			return m, nil
		}
		m.registryClient = msg.client
		return m, m.initialLoadCmd()
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

func (m Model) renderApp() string {
	sections := []string{
		m.renderTopSection(),
		m.renderMainSection(),
	}
	if m.debug {
		sections = append(sections, m.renderLogs())
	}
	return strings.Join(sections, "\n")
}

func (m Model) renderTopSection() string {
	contextName := strings.TrimSpace(m.context)
	if contextName == "" {
		contextName = "-"
	}
	statusValue := strings.TrimSpace(m.status)
	if statusValue == "" {
		statusValue = "-"
	}
	statusLine := statusStyle.Render(statusValue)
	if m.isLoading() {
		statusLine = statusLoadingStyle.Render("Loading")
		if statusValue != "-" {
			statusLine = statusLoadingStyle.Render("Loading " + statusValue)
		}
	}
	pathValue := strings.TrimSpace(m.currentPath())
	if pathValue == "" {
		pathValue = "/"
	}
	headerLine := lipgloss.JoinHorizontal(lipgloss.Top, titleStyle.Render("Beacon"), statusLine)
	metaLine := lipgloss.JoinHorizontal(
		lipgloss.Top,
		metaLabelStyle.Render("Context"),
		metaValueStyle.Render(contextName),
		metaLabelStyle.Render("Path"),
		metaValueStyle.Render(pathValue),
	)
	lines := []string{
		headerLine,
		metaLine,
	}
	if inputLine := m.renderModeInputLine(); inputLine != "" {
		lines = append(lines, modeInputStyle.Render(inputLine))
	}
	lines = append(lines, shortcutHintStyle.Render(m.renderShortcutHintLine()))
	return topSectionStyle.Width(sectionPanelWidth(m.width)).Render(strings.Join(lines, "\n"))
}

func (m Model) renderMainSection() string {
	panelWidth := sectionPanelWidth(m.width)
	contentWidth := m.mainSectionContentWidth()
	titleLabel := focusLabel(m.focus)
	body := m.renderBody()
	if m.helpActive {
		titleLabel = "Help"
		body = m.renderHelpSectionBody()
	}
	title := mainSectionTitleStyle.Render(strings.ToUpper(titleLabel))
	titleLine := mainSectionTitleLine.
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(title)
	content := strings.Join([]string{
		titleLine,
		body,
	}, "\n")
	return mainSectionStyle.Width(panelWidth).Render(content)
}

func sectionPanelWidth(width int) int {
	if width <= 0 {
		width = defaultRenderWidth
	}
	panelWidth := width - 2
	if panelWidth < 24 {
		panelWidth = width
	}
	if panelWidth < 1 {
		panelWidth = 1
	}
	return panelWidth
}

func (m Model) mainSectionContentWidth() int {
	contentWidth := sectionPanelWidth(m.width) - mainSectionHChromeChars
	if contentWidth < 1 {
		return 1
	}
	return contentWidth
}

func (m Model) renderModeInputLine() string {
	if m.commandActive {
		return m.commandInput.View()
	}
	if m.filterActive {
		return m.filterInput.View()
	}
	if value := strings.TrimSpace(m.filterInput.Value()); value != "" {
		return m.filterInput.Prompt + value
	}
	if !m.dockerHubActive {
		if !m.githubActive {
			return ""
		}
		if m.githubInputFocus {
			return m.githubInput.View()
		}
		if value := strings.TrimSpace(m.githubInput.Value()); value != "" {
			return "Search: " + value
		}
		return ""
	}
	if m.dockerHubInputFocus {
		return m.dockerHubInput.View()
	}
	if value := strings.TrimSpace(m.dockerHubInput.Value()); value != "" {
		return "Search: " + value
	}
	return ""
}

func (m Model) renderShortcutHintLine() string {
	switch {
	case m.helpActive:
		return "Help: esc/?/f1 close  q quit"
	case m.commandActive:
		return "Command: tab complete  up/down cycle  enter run  esc cancel  ? help"
	case m.filterActive:
		return "Filter: type text  enter apply  esc clear  : command  ? help"
	case m.dockerHubActive && m.dockerHubInputFocus:
		return "Docker Hub search: type image  enter search  esc exit Docker Hub  ? help"
	case m.githubActive && m.githubInputFocus:
		return "GHCR search: type image  enter search  esc exit GHCR  ? help"
	case m.dockerHubActive:
		return "Common: ? help  : command  / filter  s search  enter open  esc exit  r refresh  q quit"
	case m.githubActive:
		return "Common: ? help  : command  / filter  s search  enter open  esc exit  r refresh  q quit"
	default:
		return "Common: ? help  : command  / filter  enter open  esc back  r refresh  q quit"
	}
}

func (m Model) renderHelpSectionBody() string {
	pageTitle := m.helpPageTitle()
	shortcuts := m.currentPageHelpEntries()
	lines := []string{
		helpFooterStyle.Render(fmt.Sprintf("Current page: %s", pageTitle)),
		"",
		helpHeadingStyle.Render("Shortcuts"),
	}
	lines = append(lines, m.renderHelpEntries(shortcuts)...)
	lines = append(lines,
		"",
		helpHeadingStyle.Render("Commands"),
	)
	lines = append(lines, m.renderCommandHelpEntries(availableCommands())...)
	lines = append(lines,
		"",
		helpFooterStyle.Render("Press esc, ?, or f1 to close help."),
	)
	return strings.Join(lines, "\n")
}

func (m Model) renderHelpEntries(entries []helpEntry) []string {
	if len(entries) == 0 {
		return []string{helpFooterStyle.Render("No shortcuts available.")}
	}
	maxKey := 0
	for _, entry := range entries {
		if len(entry.Keys) > maxKey {
			maxKey = len(entry.Keys)
		}
	}
	if maxKey < 8 {
		maxKey = 8
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := fmt.Sprintf("%-*s  %s", maxKey, entry.Keys, entry.Action)
		lines = append(lines, helpItemStyle.Render(line))
	}
	return lines
}

func (m Model) renderCommandHelpEntries(entries []commandHelp) []string {
	if len(entries) == 0 {
		return []string{helpFooterStyle.Render("No commands available.")}
	}
	maxCommand := 0
	for _, entry := range entries {
		if len(entry.Command) > maxCommand {
			maxCommand = len(entry.Command)
		}
	}
	if maxCommand < 12 {
		maxCommand = 12
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		line := fmt.Sprintf(":%-*s  %s", maxCommand, entry.Command, entry.Usage)
		lines = append(lines, helpItemStyle.Render(line))
	}
	return lines
}

func (m Model) helpPageTitle() string {
	if m.dockerHubActive {
		if m.dockerHubInputFocus {
			return "Docker Hub Search"
		}
		return "Docker Hub Tags"
	}
	if m.githubActive {
		if m.githubInputFocus {
			return "GHCR Search"
		}
		return "GHCR Tags"
	}
	if m.commandActive {
		return "Command Input"
	}
	if m.filterActive {
		return "Filter Input"
	}
	return focusLabel(m.focus)
}

func (m Model) currentPageHelpEntries() []helpEntry {
	entries := []helpEntry{
		{Keys: "?", Action: "Open/close help"},
		{Keys: ":", Action: "Open command input"},
		{Keys: "q / Ctrl+C", Action: "Quit"},
	}

	if m.commandActive {
		entries = append(entries,
			helpEntry{Keys: "Tab", Action: "Autocomplete command"},
			helpEntry{Keys: "Up/Down", Action: "Cycle command suggestions"},
			helpEntry{Keys: "Enter", Action: "Run command"},
			helpEntry{Keys: "Esc", Action: "Close command input"},
		)
		return entries
	}
	if m.filterActive {
		entries = append(entries,
			helpEntry{Keys: "Type", Action: "Set filter text"},
			helpEntry{Keys: "Enter", Action: "Apply and close filter input"},
			helpEntry{Keys: "Esc", Action: "Clear filter"},
		)
		return entries
	}
	if m.dockerHubActive && m.dockerHubInputFocus {
		entries = append(entries,
			helpEntry{Keys: "Type", Action: "Set Docker Hub image query"},
			helpEntry{Keys: "Enter", Action: "Search image tags"},
			helpEntry{Keys: "Esc", Action: "Exit Docker Hub mode"},
		)
		return entries
	}
	if m.githubActive && m.githubInputFocus {
		entries = append(entries,
			helpEntry{Keys: "Type", Action: "Set GHCR image query"},
			helpEntry{Keys: "Enter", Action: "Search image tags"},
			helpEntry{Keys: "Esc", Action: "Exit GHCR mode"},
		)
		return entries
	}

	entries = append(entries,
		helpEntry{Keys: "/", Action: "Filter current list"},
		helpEntry{Keys: "Up/Down, j/k", Action: "Move selection"},
		helpEntry{Keys: "PgUp/PgDn, b/f", Action: "Move one page"},
		helpEntry{Keys: "Ctrl+U/Ctrl+D", Action: "Move half page"},
		helpEntry{Keys: "Home/End, g/G", Action: "Jump to top/bottom"},
		helpEntry{Keys: "r", Action: "Refresh current data"},
	)

	if m.dockerHubActive {
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected tag"},
			helpEntry{Keys: "s", Action: "Focus Docker Hub search input"},
			helpEntry{Keys: "Esc", Action: "Exit Docker Hub mode"},
		)
		return entries
	}
	if m.githubActive {
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected tag"},
			helpEntry{Keys: "s", Action: "Focus GHCR search input"},
			helpEntry{Keys: "Esc", Action: "Exit GHCR mode"},
		)
		return entries
	}

	switch m.focus {
	case FocusProjects:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected project images"},
			helpEntry{Keys: "Esc", Action: "Clear filter / stay on projects"},
		)
	case FocusImages:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected image tags"},
			helpEntry{Keys: "Esc", Action: "Back to projects (when available)"},
		)
	case FocusTags:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected tag history"},
			helpEntry{Keys: "Esc", Action: "Back to images"},
		)
	case FocusHistory:
		entries = append(entries,
			helpEntry{Keys: "Esc", Action: "Back to tags"},
		)
	default:
		entries = append(entries,
			helpEntry{Keys: "Enter", Action: "Open selected item"},
			helpEntry{Keys: "Esc", Action: "Go back one level"},
		)
	}

	return entries
}

func availableCommands() []commandHelp {
	return []commandHelp{
		{Command: "help", Usage: "Open the help page"},
		{Command: "dockerhub", Usage: "Open Docker Hub mode"},
		{Command: "dockerhub <image>", Usage: "Search Docker Hub image tags"},
		{Command: "github", Usage: "Open GitHub Container Registry mode"},
		{Command: "github <image>", Usage: "Search GHCR image tags"},
		{Command: "ghcr", Usage: "Alias for github"},
		{Command: "ghcr <image>", Usage: "Alias search for GHCR tags"},
		{Command: "context", Usage: "Open context selection"},
		{Command: "context add", Usage: "Create a new context"},
		{Command: "context edit <name>", Usage: "Edit an existing context"},
		{Command: "context remove <name>", Usage: "Remove a context"},
		{Command: "context <name>", Usage: "Switch to context by name"},
	}
}

func (m Model) renderContextSelectionModal() string {
	lines := []string{
		modalTitleStyle.Render("Select Context"),
		modalLabelStyle.Render("Choose a registry context to continue."),
		modalDividerStyle.Render(strings.Repeat("─", 24)),
	}
	if m.contextSelectionError != "" {
		lines = append(lines, modalErrorStyle.Render(m.contextSelectionError))
	}
	if len(m.contexts) == 0 {
		lines = append(lines,
			modalErrorStyle.Render("No contexts configured."),
			"",
			modalHelpStyle.Render("a add context  esc close  q quit"),
		)
		return m.renderModalCard(strings.Join(lines, "\n"), 84)
	}

	selected := clampInt(m.contextSelectionIndex, 0, len(m.contexts)-1)
	for i, ctx := range m.contexts {
		prefix := "  "
		if i == selected {
			prefix = "> "
		}

		name := contextDisplayName(ctx, i)
		host := strings.TrimSpace(ctx.Host)
		hostLabel := modalOptionMutedStyle.Render(host)
		if host == "" {
			hostLabel = modalOptionErrorStyle.Render("(no registry configured)")
		}

		row := prefix + lipglossv2.JoinHorizontal(
			lipglossv2.Top,
			name,
			"  ",
			hostLabel,
		)

		style := modalOptionStyle
		if i == selected {
			style = modalOptionFocusStyle
		}
		lines = append(lines, style.Render(row))
	}
	lines = append(lines,
		"",
		modalHelpStyle.Render(m.contextSelectionHelpText()),
	)
	return m.renderModalCard(strings.Join(lines, "\n"), 84)
}

func (m Model) renderAuthModal() string {
	registryHost := strings.TrimSpace(m.registryHost)
	if registryHost == "" {
		registryHost = "-"
	}
	lines := []string{
		modalTitleStyle.Render("Authentication Required"),
		modalLabelStyle.Render(fmt.Sprintf("Registry  %s", registryHost)),
		modalDividerStyle.Render(strings.Repeat("─", 24)),
	}
	if m.authError != "" {
		lines = append(lines, modalErrorStyle.Render(m.authError))
	}

	username := m.usernameInput.View()
	password := m.passwordInput.View()
	if m.authFocus == 0 {
		username = modalInputFocusStyle.Render(username)
	} else {
		username = modalInputStyle.Render(username)
	}
	if m.authFocus == 1 {
		password = modalInputFocusStyle.Render(password)
	} else {
		password = modalInputStyle.Render(password)
	}

	remember := ""
	if m.authUI().ShowRemember {
		remember = "[ ] Remember session"
		if m.remember {
			remember = "[x] Remember session"
		}
	}

	if m.authFocus == 2 && m.authUI().ShowRemember {
		remember = modalFocusStyle.Render(remember)
	} else if m.authUI().ShowRemember {
		remember = modalLabelStyle.Render(remember)
	}

	help := "tab/shift+tab move  enter submit  q quit"
	if m.authUI().ShowRemember {
		help = "tab/shift+tab move  space toggle  enter submit  q quit"
	}

	lines = append(lines,
		"",
		modalLabelStyle.Render("Username"),
		username,
		modalLabelStyle.Render("Password"),
		password,
	)
	if m.authUI().ShowRemember {
		lines = append(lines, remember)
	}
	lines = append(lines,
		"",
		modalHelpStyle.Render(strings.ToUpper(help)),
	)

	return m.renderModalCard(strings.Join(lines, "\n"), 72)
}

func (m Model) renderConfirmModal() string {
	title := strings.TrimSpace(m.confirmTitle)
	if title == "" {
		title = "Confirm action"
	}
	confirmLabel := "Confirm"
	confirmButtonStyle := modalButtonStyle
	confirmButtonFocusStyle := modalButtonFocusStyle
	switch m.confirmAction {
	case confirmActionQuit:
		confirmLabel = "Quit"
		confirmButtonStyle = modalDangerButtonStyle
		confirmButtonFocusStyle = modalDangerFocusStyle
	}

	cancel := "Cancel"
	if m.confirmFocus == 0 {
		cancel = modalButtonFocusStyle.Render(cancel)
	} else {
		cancel = modalButtonStyle.Render(cancel)
	}
	confirm := confirmButtonStyle.Render(confirmLabel)
	if m.confirmFocus == 1 {
		confirm = confirmButtonFocusStyle.Render(confirmLabel)
	}
	buttonRow := lipglossv2.JoinHorizontal(
		lipglossv2.Top,
		lipglossv2.NewStyle().MarginRight(2).Render(cancel),
		confirm,
	)

	lines := []string{
		modalTitleStyle.Render(title),
	}
	if message := strings.TrimSpace(m.confirmMessage); message != "" {
		lines = append(lines, modalLabelStyle.Render(message))
	}
	lines = append(lines,
		"",
		buttonRow,
		"",
		modalHelpStyle.Render("tab/left/right move  enter choose  y/n quick select"),
	)
	return m.renderModalCard(strings.Join(lines, "\n"), 64)
}

func (m Model) renderModal(base, modal string) string {
	width, height := m.modalViewport(base)
	background := lipglossv2.Place(width, height, lipglossv2.Left, lipglossv2.Top, modalBackdropStyle.Render(base))
	canvas := lipglossv2.NewCanvas(lipglossv2.NewLayer(background))
	canvas.AddLayers(
		lipglossv2.NewLayer(modal).
			X(maxInt(0, (width-lipglossv2.Width(modal))/2)).
			Y(maxInt(0, (height-lipglossv2.Height(modal))/2)).
			Z(1),
	)
	return canvas.Render()
}

func (m Model) renderModalCard(content string, maxWidth int) string {
	return modalPanelStyle.Width(m.modalWidth(maxWidth)).Render(content)
}

func (m Model) modalWidth(maxWidth int) int {
	width, _ := m.modalViewport("")
	if width <= 2 {
		return width
	}
	modalWidth := width - 8
	if modalWidth < 24 {
		modalWidth = width - 2
	}
	if maxWidth > 0 && modalWidth > maxWidth {
		modalWidth = maxWidth
	}
	if modalWidth < 12 {
		modalWidth = 12
	}
	return modalWidth
}

func (m Model) modalViewport(base string) (int, int) {
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = maxInt(24, lineCount(base))
	}
	return width, height
}

func (m Model) isContextSelectionActive() bool {
	return m.contextSelectionActive
}

func (m Model) isContextFormActive() bool {
	return m.contextFormActive
}

func (m Model) isAuthModalActive() bool {
	return !m.isContextSelectionActive() && !m.isContextFormActive() && m.authRequired && m.registryClient == nil
}

func (m Model) isConfirmModalActive() bool {
	return m.confirmAction != confirmActionNone
}

func (m Model) renderBody() string {
	view := m.table.View()
	if len(m.table.Rows()) == 0 {
		return view + "\n" + emptyStyle.Render(m.emptyBodyMessage())
	}
	return view
}

func (m Model) renderLogs() string {
	panelWidth := sectionPanelWidth(m.width)
	contentWidth := maxInt(10, panelWidth-6)

	lines := []string{logTitleStyle.Render("Requests")}
	visible := m.visibleLogs()
	if len(visible) == 0 {
		lines = append(lines, emptyStyle.Render("(no requests yet)"))
		for i := 1; i < maxVisibleLogs; i++ {
			lines = append(lines, "")
		}
	} else {
		start := 0
		if len(visible) > maxVisibleLogs {
			start = len(visible) - maxVisibleLogs
		}
		for _, entry := range visible[start:] {
			lines = append(lines, truncateLogLine(entry, contentWidth))
		}
		for len(lines) < maxVisibleLogs+1 {
			lines = append(lines, "")
		}
	}
	return logBoxStyle.Width(panelWidth).Render(strings.Join(lines, "\n"))
}

func (m Model) visibleLogs() []string {
	if len(m.logs) == 0 {
		return nil
	}
	count := minInt(len(m.logs), maxVisibleLogs)
	return m.logs[len(m.logs)-count:]
}

func (m Model) currentPath() string {
	if m.dockerHubActive {
		if m.dockerHubImage != "" {
			return "dockerhub/" + m.dockerHubImage
		}
		return "dockerhub"
	}
	if m.githubActive {
		if m.githubImage != "" {
			return "ghcr/" + m.githubImage
		}
		return "ghcr"
	}
	if path := m.breadcrumb(); path != "" {
		return path
	}
	return "/"
}

func (m Model) handleContextSelectionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.contexts) == 0 {
		switch msg.String() {
		case "ctrl+c":
			return m.openQuitConfirm()
		case "q":
			return m.openQuitConfirm()
		case "esc":
			if m.contextSelectionRequired {
				return m.openQuitConfirm()
			}
			return m.closeContextSelection()
		case "a":
			return m.openContextFormAdd(true, false)
		}
		return m, nil
	}

	switch msg.String() {
	case "ctrl+c":
		return m.openQuitConfirm()
	case "q":
		return m.openQuitConfirm()
	case "esc":
		if m.contextSelectionRequired {
			return m.openQuitConfirm()
		}
		return m.closeContextSelection()
	case "up", "k", "shift+tab":
		m.contextSelectionIndex--
		if m.contextSelectionIndex < 0 {
			m.contextSelectionIndex = len(m.contexts) - 1
		}
		m.contextSelectionError = ""
		return m, nil
	case "down", "j", "tab":
		m.contextSelectionIndex = (m.contextSelectionIndex + 1) % len(m.contexts)
		m.contextSelectionError = ""
		return m, nil
	case "home", "g":
		m.contextSelectionIndex = 0
		m.contextSelectionError = ""
		return m, nil
	case "end", "G":
		m.contextSelectionIndex = len(m.contexts) - 1
		m.contextSelectionError = ""
		return m, nil
	case "a":
		return m.openContextFormAdd(true, false)
	case "enter":
		selected := clampInt(m.contextSelectionIndex, 0, len(m.contexts)-1)
		return m.switchContextAt(selected)
	}

	return m, nil
}

func (m Model) openHelp() (tea.Model, tea.Cmd) {
	m.helpActive = true
	return m, nil
}

func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "?", "f1":
		m.helpActive = false
		return m, nil
	case "enter":
		m.helpActive = false
		return m, nil
	case "q", "ctrl+c":
		m.helpActive = false
		return m.openQuitConfirm()
	default:
		return m, nil
	}
}

func isHelpShortcut(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "?", "f1":
		return true
	default:
		return false
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.clearFilter()
			m.syncTable()
			return m, nil
		case ":":
			return m.enterCommandMode()
		case "enter":
			m.stopFilterEditing()
			m.syncTable()
			return m, nil
		}
		before := m.filterInput.Value()
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		if m.filterInput.Value() != before {
			m.table.SetCursor(0)
			m.syncTable()
		}
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "esc":
		return m, m.handleEscape()
	case "/":
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.syncTable()
		return m, nil
	case ":":
		return m.enterCommandMode()
	case "r":
		return m, m.refreshCurrent()
	case "enter":
		return m, m.handleEnter()
	}

	if len(msg.Runes) == 1 && msg.Runes[0] == ':' {
		return m.enterCommandMode()
	}
	if m.handleTableNavKey(msg) {
		return m, nil
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) handleDockerHubKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.clearFilter()
			m.syncTable()
			return m, nil
		case ":":
			return m.enterCommandMode()
		case "enter":
			m.stopFilterEditing()
			m.syncTable()
			return m, nil
		}
		before := m.filterInput.Value()
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		if m.filterInput.Value() != before {
			m.table.SetCursor(0)
			m.syncTable()
			return m, tea.Batch(cmd, m.maybeLoadDockerHubForFilter())
		}
		return m, cmd
	}

	if m.dockerHubInputFocus {
		switch msg.String() {
		case "ctrl+c":
			return m.openQuitConfirm()
		case "esc":
			return m.exitDockerHubMode()
		case "enter":
			query := strings.TrimSpace(m.dockerHubInput.Value())
			if query == "" {
				m.status = "Enter an image name to search Docker Hub"
				return m, nil
			}
			return m, m.searchDockerHub(query)
		}
		var cmd tea.Cmd
		m.dockerHubInput, cmd = m.dockerHubInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "esc":
		if m.focus == FocusHistory {
			return m, m.handleEscape()
		}
		return m.exitDockerHubMode()
	case ":":
		return m.enterCommandMode()
	case "enter":
		return m, m.openDockerHubTagHistory()
	case "s":
		m.dockerHubInput.SetValue("")
		m.dockerHubInputFocus = true
		cmd := m.dockerHubInput.Focus()
		m.dockerHubInput.CursorEnd()
		return m, cmd
	case "/":
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.syncTable()
		return m, nil
	case "r":
		return m, m.refreshDockerHub()
	}

	if len(msg.Runes) == 1 && msg.Runes[0] == ':' {
		return m.enterCommandMode()
	}
	if !m.dockerHubInputFocus && m.handleTableNavKey(msg) {
		return m, m.maybeLoadDockerHubOnBottom(msg)
	}

	if len(msg.Runes) > 0 || msg.String() == "backspace" || msg.String() == "delete" {
		m.dockerHubInputFocus = true
		if !m.dockerHubInput.Focused() {
			return m, m.dockerHubInput.Focus()
		}
		var cmd tea.Cmd
		m.dockerHubInput, cmd = m.dockerHubInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) handleGitHubKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.clearFilter()
			m.syncTable()
			return m, nil
		case ":":
			return m.enterCommandMode()
		case "enter":
			m.stopFilterEditing()
			m.syncTable()
			return m, nil
		}
		before := m.filterInput.Value()
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		if m.filterInput.Value() != before {
			m.table.SetCursor(0)
			m.syncTable()
			return m, tea.Batch(cmd, m.maybeLoadGitHubForFilter())
		}
		return m, cmd
	}

	if m.githubInputFocus {
		switch msg.String() {
		case "ctrl+c":
			return m.openQuitConfirm()
		case "esc":
			return m.exitGitHubMode()
		case "enter":
			query := strings.TrimSpace(m.githubInput.Value())
			if query == "" {
				m.status = "Enter an image name to search GHCR (owner/image)"
				return m, nil
			}
			return m, m.searchGitHub(query)
		}
		var cmd tea.Cmd
		m.githubInput, cmd = m.githubInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "esc":
		if m.focus == FocusHistory {
			return m, m.handleEscape()
		}
		return m.exitGitHubMode()
	case ":":
		return m.enterCommandMode()
	case "enter":
		return m, m.openGitHubTagHistory()
	case "s":
		m.githubInput.SetValue("")
		m.githubInputFocus = true
		cmd := m.githubInput.Focus()
		m.githubInput.CursorEnd()
		return m, cmd
	case "/":
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
		m.syncTable()
		return m, nil
	case "r":
		return m, m.refreshGitHub()
	}

	if len(msg.Runes) == 1 && msg.Runes[0] == ':' {
		return m.enterCommandMode()
	}
	if !m.githubInputFocus && m.handleTableNavKey(msg) {
		return m, m.maybeLoadGitHubOnBottom(msg)
	}

	if len(msg.Runes) > 0 || msg.String() == "backspace" || msg.String() == "delete" {
		m.githubInputFocus = true
		if !m.githubInput.Focused() {
			return m, m.githubInput.Focus()
		}
		var cmd tea.Cmd
		m.githubInput, cmd = m.githubInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *Model) handleTableNavKey(msg tea.KeyMsg) bool {
	rowCount := len(m.table.Rows())
	if rowCount == 0 {
		return false
	}
	step := maxInt(1, m.table.Height())

	switch msg.String() {
	case "up", "k":
		m.table.MoveUp(1)
		return true
	case "down", "j":
		m.table.MoveDown(1)
		return true
	case "pgup", "b":
		m.table.MoveUp(step)
		return true
	case "pgdown", "f", " ":
		m.table.MoveDown(step)
		return true
	case "ctrl+u", "u":
		m.table.MoveUp(maxInt(1, step/2))
		return true
	case "ctrl+d", "d":
		m.table.MoveDown(maxInt(1, step/2))
		return true
	case "home", "g":
		m.table.GotoTop()
		return true
	case "end", "G":
		m.table.GotoBottom()
		return true
	default:
		return false
	}
}

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
	switch cmdName {
	case "context", "ctx":
		return m.runContextCommand(args)
	case "dockerhub", "dh", "hub":
		if len(args) > 0 {
			query := strings.Join(args, " ")
			model, _ := m.enterDockerHubMode()
			next := model.(Model)
			next.dockerHubInput.SetValue(query)
			next.dockerHubInput.CursorEnd()
			return next, next.searchDockerHub(query)
		}
		return m.enterDockerHubMode()
	case "github", "ghcr":
		if len(args) > 0 {
			query := strings.Join(args, " ")
			model, _ := m.enterGitHubMode()
			next := model.(Model)
			next.githubInput.SetValue(query)
			next.githubInput.CursorEnd()
			return next, next.searchGitHub(query)
		}
		return m.enterGitHubMode()
	case "help":
		return m.openHelp()
	default:
		m.status = fmt.Sprintf("Unknown command: %s", cmdName)
		return m, nil
	}
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

func matchCommands(prefix string) []string {
	candidates := []string{"context", "ctx", "dockerhub", "hub", "github", "ghcr", "help"}
	if prefix == "" {
		return candidates
	}
	prefix = strings.ToLower(prefix)
	var out []string
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			out = append(out, candidate)
		}
	}
	return out
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

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m.openQuitConfirm()
	case "tab", "down":
		m.authFocus = (m.authFocus + 1) % m.authFieldCount()
		m.syncAuthFocus()
	case "shift+tab", "up":
		m.authFocus--
		if m.authFocus < 0 {
			m.authFocus = m.authFieldCount() - 1
		}
		m.syncAuthFocus()
	case " ":
		if m.authFocus == 2 && m.authUI().ShowRemember {
			m.remember = !m.remember
		}
	case "enter":
		return m.submitAuth()
	}

	var cmd tea.Cmd
	switch m.authFocus {
	case 0:
		m.usernameInput, cmd = m.usernameInput.Update(msg)
	case 1:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	}
	return m, cmd
}

func (m Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "shift+tab":
		m.confirmFocus = 0
	case "right", "l", "tab":
		m.confirmFocus = 1
	case "esc", "n":
		m.clearConfirm()
		return m, nil
	case "y":
		return m.resolveConfirm(true)
	case "enter":
		return m.resolveConfirm(m.confirmFocus == 1)
	case "ctrl+c", "q":
		return m.resolveConfirm(true)
	}
	return m, nil
}

func (m Model) openQuitConfirm() (tea.Model, tea.Cmd) {
	m.confirmAction = confirmActionQuit
	m.confirmTitle = "Quit Beacon?"
	if m.isLoading() {
		m.confirmMessage = "A request is still in progress."
	} else {
		m.confirmMessage = "Close the current session?"
	}
	m.confirmFocus = 0
	return m, nil
}

func (m Model) resolveConfirm(accept bool) (tea.Model, tea.Cmd) {
	action := m.confirmAction
	m.clearConfirm()
	if !accept {
		return m, nil
	}
	switch action {
	case confirmActionQuit:
		return m, tea.Quit
	default:
		return m, nil
	}
}

func (m *Model) clearConfirm() {
	m.confirmAction = confirmActionNone
	m.confirmTitle = ""
	m.confirmMessage = ""
	m.confirmFocus = 0
}

func (m Model) submitAuth() (tea.Model, tea.Cmd) {
	auth := m.auth
	switch auth.Kind {
	case "registry_v2":
		auth.RegistryV2.Username = strings.TrimSpace(m.usernameInput.Value())
		auth.RegistryV2.Password = m.passwordInput.Value()
		auth.RegistryV2.Remember = m.remember
		if !auth.RegistryV2.Remember {
			auth.RegistryV2.RefreshToken = ""
		}
	case "harbor":
		auth.Harbor.Username = strings.TrimSpace(m.usernameInput.Value())
		auth.Harbor.Password = m.passwordInput.Value()
	}

	client, err := registry.NewClientWithLogger(m.registryHost, auth, m.logger)
	if err != nil {
		m.authError = err.Error()
		return m, nil
	}

	registry.PersistAuthCache(m.registryHost, auth)
	m.auth = auth
	m.registryClient = client
	m.authRequired = false
	m.authError = ""
	return m, m.initialLoadCmd()
}

func (m Model) enterDockerHubMode() (tea.Model, tea.Cmd) {
	if m.githubActive {
		m.focus = m.githubPrevFocus
		if m.githubPrevStatus != "" {
			m.status = m.githubPrevStatus
		}
	}
	m.githubActive = false
	m.githubInputFocus = false
	m.githubInput.Blur()
	m.githubLoading = false
	m.dockerHubActive = true
	m.dockerHubPrevFocus = m.focus
	m.dockerHubPrevStatus = m.status
	m.focus = FocusDockerHubTags
	m.status = "Docker Hub search"
	m.dockerHubInputFocus = true
	cmd := m.dockerHubInput.Focus()
	m.dockerHubInput.CursorEnd()
	m.clearFilter()
	m.syncTable()
	return m, cmd
}

func (m Model) exitDockerHubMode() (tea.Model, tea.Cmd) {
	m.dockerHubActive = false
	m.dockerHubInputFocus = false
	m.dockerHubInput.Blur()
	m.dockerHubLoading = false
	m.focus = m.dockerHubPrevFocus
	if m.dockerHubPrevStatus != "" {
		m.status = m.dockerHubPrevStatus
	}
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m Model) enterGitHubMode() (tea.Model, tea.Cmd) {
	if m.dockerHubActive {
		m.focus = m.dockerHubPrevFocus
		if m.dockerHubPrevStatus != "" {
			m.status = m.dockerHubPrevStatus
		}
	}
	m.dockerHubActive = false
	m.dockerHubInputFocus = false
	m.dockerHubInput.Blur()
	m.dockerHubLoading = false
	m.githubActive = true
	m.githubPrevFocus = m.focus
	m.githubPrevStatus = m.status
	m.focus = FocusGitHubTags
	m.status = "GHCR search"
	m.githubInputFocus = true
	cmd := m.githubInput.Focus()
	m.githubInput.CursorEnd()
	m.clearFilter()
	m.syncTable()
	return m, cmd
}

func (m Model) exitGitHubMode() (tea.Model, tea.Cmd) {
	m.githubActive = false
	m.githubInputFocus = false
	m.githubInput.Blur()
	m.githubLoading = false
	m.focus = m.githubPrevFocus
	if m.githubPrevStatus != "" {
		m.status = m.githubPrevStatus
	}
	m.clearFilter()
	m.syncTable()
	return m, nil
}

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
	query := strings.TrimSpace(m.dockerHubInput.Value())
	if query == "" {
		m.status = "Enter an image name to search Docker Hub"
		return nil
	}
	return m.searchDockerHub(query)
}

func (m *Model) searchDockerHub(query string) tea.Cmd {
	if m.dockerHubLoading {
		m.status = "Docker Hub request already in progress"
		return nil
	}
	// Once a search is submitted, return interaction to the list.
	m.dockerHubInputFocus = false
	m.dockerHubInput.Blur()
	m.table.Focus()
	m.status = fmt.Sprintf("Searching Docker Hub for %s...", query)
	m.dockerHubTags = nil
	m.dockerHubImage = ""
	m.dockerHubNext = ""
	m.dockerHubRateLimit = registry.DockerHubRateLimit{}
	m.dockerHubRetryUntil = time.Time{}
	m.dockerHubLoading = true
	m.startLoading()
	m.syncTable()
	return loadDockerHubTagsFirstPageCmd(query, m.logger)
}

func (m *Model) openDockerHubTagHistory() tea.Cmd {
	if m.focus != FocusDockerHubTags {
		return nil
	}
	list := m.listView()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(list.indices) {
		return nil
	}
	index := list.indices[cursor]
	if index < 0 || index >= len(m.dockerHubTags) {
		return nil
	}
	if strings.TrimSpace(m.dockerHubImage) == "" {
		m.status = "Select an image first"
		return nil
	}

	selected := m.dockerHubTags[index]
	m.selectedImage = registry.Image{Name: m.dockerHubImage}
	m.hasSelectedImage = true
	m.selectedTag = selected
	m.hasSelectedTag = true
	m.history = nil
	m.focus = FocusHistory
	m.status = fmt.Sprintf("Loading history for %s:%s...", m.dockerHubImage, selected.Name)
	m.clearFilter()
	m.syncTable()
	m.startLoading()
	return loadDockerHubHistoryCmd(m.dockerHubImage, selected.Name, m.logger)
}

func (m *Model) maybeLoadDockerHubOnBottom(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "down", "j", "pgdown", "f", " ", "ctrl+d", "d", "end", "G":
	default:
		return nil
	}
	if m.focus != FocusDockerHubTags {
		return nil
	}
	rows := m.table.Rows()
	if len(rows) == 0 {
		return nil
	}
	if m.table.Cursor() < len(rows)-1 {
		return nil
	}
	return m.requestNextDockerHubPage(false)
}

func (m *Model) maybeLoadDockerHubForFilter() tea.Cmd {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return nil
	}
	if m.focus != FocusDockerHubTags {
		return nil
	}
	if len(m.table.Rows()) >= maxInt(1, m.table.Height()) {
		return nil
	}
	return m.requestNextDockerHubPage(true)
}

func (m *Model) requestNextDockerHubPage(forFilter bool) tea.Cmd {
	if m.dockerHubLoading || m.dockerHubNext == "" || m.dockerHubImage == "" {
		return nil
	}
	now := time.Now()
	if !m.dockerHubRetryUntil.IsZero() && now.Before(m.dockerHubRetryUntil) {
		m.status = m.dockerHubRateLimitStatus("Docker Hub rate limit reached")
		return nil
	}
	if m.dockerHubRateLimit.Remaining == 0 && !m.dockerHubRateLimit.ResetAt.IsZero() && now.Before(m.dockerHubRateLimit.ResetAt) {
		m.dockerHubRetryUntil = m.dockerHubRateLimit.ResetAt
		m.status = m.dockerHubRateLimitStatus("Docker Hub rate limit reached")
		return nil
	}
	if forFilter {
		m.status = fmt.Sprintf("Loading more tags for %s to match filter...", m.dockerHubImage)
	} else {
		m.status = fmt.Sprintf("Loading more tags for %s...", m.dockerHubImage)
	}
	m.dockerHubLoading = true
	m.startLoading()
	return loadDockerHubTagsNextPageCmd(m.dockerHubImage, m.dockerHubNext, m.logger)
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
	status := fmt.Sprintf("Docker Hub: %s (%d tags)", m.dockerHubImage, len(m.dockerHubTags))
	if m.dockerHubNext != "" {
		status += " [more]"
	}
	return status + m.dockerHubRateLimitSuffix()
}

func (m *Model) refreshGitHub() tea.Cmd {
	query := strings.TrimSpace(m.githubInput.Value())
	if query == "" {
		m.status = "Enter an image name to search GHCR (owner/image)"
		return nil
	}
	return m.searchGitHub(query)
}

func (m *Model) searchGitHub(query string) tea.Cmd {
	if m.githubLoading {
		m.status = "GHCR request already in progress"
		return nil
	}
	m.githubInputFocus = false
	m.githubInput.Blur()
	m.table.Focus()
	m.status = fmt.Sprintf("Searching GHCR for %s...", query)
	m.githubTags = nil
	m.githubImage = ""
	m.githubNext = ""
	m.githubLoading = true
	m.startLoading()
	m.syncTable()
	return loadGitHubTagsFirstPageCmd(query, m.logger)
}

func (m *Model) openGitHubTagHistory() tea.Cmd {
	if m.focus != FocusGitHubTags {
		return nil
	}
	list := m.listView()
	cursor := m.table.Cursor()
	if cursor < 0 || cursor >= len(list.indices) {
		return nil
	}
	index := list.indices[cursor]
	if index < 0 || index >= len(m.githubTags) {
		return nil
	}
	if strings.TrimSpace(m.githubImage) == "" {
		m.status = "Select an image first"
		return nil
	}

	selected := m.githubTags[index]
	m.selectedImage = registry.Image{Name: m.githubImage}
	m.hasSelectedImage = true
	m.selectedTag = selected
	m.hasSelectedTag = true
	m.history = nil
	m.focus = FocusHistory
	m.status = fmt.Sprintf("Loading history for %s:%s...", m.githubImage, selected.Name)
	m.clearFilter()
	m.syncTable()
	m.startLoading()
	return loadGitHubHistoryCmd(m.githubImage, selected.Name, m.logger)
}

func (m *Model) maybeLoadGitHubOnBottom(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "down", "j", "pgdown", "f", " ", "ctrl+d", "d", "end", "G":
	default:
		return nil
	}
	if m.focus != FocusGitHubTags {
		return nil
	}
	rows := m.table.Rows()
	if len(rows) == 0 {
		return nil
	}
	if m.table.Cursor() < len(rows)-1 {
		return nil
	}
	return m.requestNextGitHubPage(false)
}

func (m *Model) maybeLoadGitHubForFilter() tea.Cmd {
	filter := strings.TrimSpace(m.filterInput.Value())
	if filter == "" {
		return nil
	}
	if m.focus != FocusGitHubTags {
		return nil
	}
	if len(m.table.Rows()) >= maxInt(1, m.table.Height()) {
		return nil
	}
	return m.requestNextGitHubPage(true)
}

func (m *Model) requestNextGitHubPage(forFilter bool) tea.Cmd {
	if m.githubLoading || m.githubNext == "" || m.githubImage == "" {
		return nil
	}
	if forFilter {
		m.status = fmt.Sprintf("Loading more tags for %s to match filter...", m.githubImage)
	} else {
		m.status = fmt.Sprintf("Loading more tags for %s...", m.githubImage)
	}
	m.githubLoading = true
	m.startLoading()
	return loadGitHubTagsNextPageCmd(m.githubImage, m.githubNext, m.logger)
}

func (m Model) githubLoadedStatus() string {
	status := fmt.Sprintf("GHCR: %s (%d tags)", m.githubImage, len(m.githubTags))
	if m.githubNext != "" {
		status += " [more]"
	}
	return status
}

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

func (m *Model) syncTable() {
	list := m.listView()
	width := m.width
	if width <= 0 {
		width = defaultRenderWidth
	}
	filterWidth := clampInt(width-10, 10, maxFilterWidth)
	m.filterInput.Width = filterWidth
	m.dockerHubInput.Width = filterWidth
	m.githubInput.Width = filterWidth
	m.commandInput.Width = filterWidth

	tableWidth := maxInt(10, m.mainSectionContentWidth())
	columns := makeColumns(m.focus, tableWidth, m.effectiveTableSpec())
	rows := normalizeTableRows(toTableRows(list.rows), len(columns))
	columnsChanged := !equalTableColumns(m.tableColumns, columns)
	if columnsChanged {
		// Clear rows only when column shape changes to avoid transient empty-frame flicker.
		// This still protects bubbles/table from row/column length mismatches.
		if len(m.table.Rows()) > 0 {
			m.table.SetRows(nil)
		}
		m.table.SetColumns(columns)
		m.tableColumns = append(m.tableColumns[:0], columns...)
	}

	if columnsChanged || !equalTableRows(m.table.Rows(), rows) {
		m.table.SetRows(rows)
	}

	tableHeight := m.tableHeight()
	if m.table.Height() != tableHeight {
		m.table.SetHeight(tableHeight)
	}
	if m.table.Width() != tableWidth {
		m.table.SetWidth(tableWidth)
	}
	m.table.SetStyles(tableStyles())
	cursor := m.table.Cursor()
	if len(list.rows) == 0 {
		m.table.SetCursor(0)
	} else if cursor >= len(list.rows) {
		m.table.SetCursor(len(list.rows) - 1)
	}
}

func (m Model) tableHeight() int {
	if m.height <= 0 {
		return defaultTableHeight
	}
	topLines := lineCount(m.renderTopSection())
	sectionSeparators := 1 // top section + main section
	debugLines := 0
	if m.debug {
		// Requests section: top/bottom border + title + fixed visible rows.
		debugLines = maxVisibleLogs + 3
		sectionSeparators++ // main section + debug section
	}
	// bubbles/table height controls only row viewport height; header + header border
	// plus the bordered main section and title consume extra terminal lines.
	available := m.height - topLines - mainSectionTitleLines - mainSectionBorderLines - debugLines - tableChromeLines - sectionSeparators
	if available < minTableHeight {
		return minTableHeight
	}
	return available
}

func focusLabel(focus Focus) string {
	switch focus {
	case FocusProjects:
		return "Projects"
	case FocusImages:
		return "Images"
	case FocusHistory:
		return "History"
	case FocusDockerHubTags:
		return "Docker Hub Tags"
	case FocusGitHubTags:
		return "GHCR Tags"
	default:
		return "Tags"
	}
}

func (m Model) breadcrumb() string {
	if m.hasSelectedTag {
		return fmt.Sprintf("%s:%s", m.selectedImage.Name, m.selectedTag.Name)
	}
	if m.hasSelectedImage {
		return m.selectedImage.Name
	}
	if m.hasSelectedProject {
		return m.selectedProject
	}
	return ""
}

func (m Model) defaultFocus() Focus {
	if m.tableSpec().SupportsProjects {
		return FocusProjects
	}
	return FocusImages
}

func (m Model) tableSpec() registry.TableSpec {
	if m.provider == nil {
		return registry.TableSpec{}
	}
	return m.provider.TableSpec()
}

func (m Model) effectiveTableSpec() registry.TableSpec {
	spec := m.tableSpec()
	if m.dockerHubActive || m.focus == FocusDockerHubTags {
		spec.Tag = registry.TagTableSpec{
			ShowSize:       true,
			ShowPushed:     true,
			ShowLastPulled: true,
		}
	} else if m.githubActive || m.focus == FocusGitHubTags {
		spec.Tag = registry.TagTableSpec{
			ShowSize:       false,
			ShowPushed:     false,
			ShowLastPulled: false,
		}
	}
	return spec
}

func (m Model) visibleImages() []registry.Image {
	if !m.tableSpec().SupportsProjects || !m.hasSelectedProject {
		return m.images
	}
	prefix := m.selectedProject + "/"
	filtered := make([]registry.Image, 0, len(m.images))
	for _, image := range m.images {
		if strings.HasPrefix(image.Name, prefix) {
			filtered = append(filtered, image)
		}
	}
	// Harbor responses can be project-qualified ("project/repo") or plain ("repo"),
	// depending on endpoint/version. If no project-qualified names are present,
	// show the loaded list as-is.
	if len(filtered) == 0 {
		return m.images
	}
	return filtered
}

func deriveProjects(images []registry.Image) []projectInfo {
	if len(images) == 0 {
		return nil
	}
	counts := make(map[string]int)
	for _, image := range images {
		trimmed := strings.Trim(image.Name, "/")
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, "/", 2)
		if len(parts) == 0 || parts[0] == "" {
			continue
		}
		counts[parts[0]]++
	}

	projects := make([]projectInfo, 0, len(counts))
	for name, count := range counts {
		projects = append(projects, projectInfo{Name: name, ImageCount: count})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})
	return projects
}

func toProjectInfos(projects []registry.Project) []projectInfo {
	if len(projects) == 0 {
		return nil
	}
	items := make([]projectInfo, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectInfo{
			Name:       project.Name,
			ImageCount: project.ImageCount,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items
}

type listView struct {
	headers []string
	rows    [][]string
	indices []int
}

func (m Model) listView() listView {
	filter := m.filterInput.Value()
	spec := m.effectiveTableSpec()
	switch m.focus {
	case FocusProjects:
		return filterRows(projectHeaders(), projectRows(m.projects), filter)
	case FocusImages:
		return filterRows(imageHeaders(spec.Image), imageRows(m.visibleImages(), m.selectedProject, spec.SupportsProjects, spec.Image), filter)
	case FocusHistory:
		return filterRows(historyHeaders(spec.History), historyRows(m.history, spec.History), filter)
	case FocusDockerHubTags:
		return filterRows(tagHeaders(spec.Tag), tagRows(m.dockerHubTags, spec.Tag), filter)
	case FocusGitHubTags:
		return filterRows(tagHeaders(spec.Tag), tagRows(m.githubTags, spec.Tag), filter)
	default:
		return filterRows(tagHeaders(spec.Tag), tagRows(m.tags, spec.Tag), filter)
	}
}

func imageHeaders(spec registry.ImageTableSpec) []string {
	headers := []string{"Name"}
	if spec.ShowTagCount {
		headers = append(headers, "Tags")
	}
	if spec.ShowPulls {
		headers = append(headers, "Pulls")
	}
	if spec.ShowUpdated {
		headers = append(headers, "Updated")
	}
	return headers
}

func projectHeaders() []string {
	return []string{"Name", "Images"}
}

func tagHeaders(spec registry.TagTableSpec) []string {
	headers := []string{"Name"}
	if spec.ShowSize {
		headers = append(headers, "Size")
	}
	if spec.ShowPushed {
		headers = append(headers, "Pushed")
	}
	if spec.ShowLastPulled {
		headers = append(headers, "Last Pull")
	}
	return headers
}

func historyHeaders(spec registry.HistoryTableSpec) []string {
	headers := []string{"Command", "Created"}
	if spec.ShowSize {
		headers = append(headers, "Size")
	}
	if spec.ShowComment {
		headers = append(headers, "Comment")
	}
	return headers
}

func imageRows(images []registry.Image, selectedProject string, supportsProjects bool, spec registry.ImageTableSpec) [][]string {
	if len(images) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(images))
	for _, image := range images {
		name := image.Name
		if supportsProjects && selectedProject != "" {
			prefix := selectedProject + "/"
			if strings.HasPrefix(name, prefix) {
				name = strings.TrimPrefix(name, prefix)
			}
		}
		row := []string{name}
		if spec.ShowTagCount {
			row = append(row, formatCount(image.TagCount))
		}
		if spec.ShowPulls {
			row = append(row, formatCount(image.PullCount))
		}
		if spec.ShowUpdated {
			row = append(row, formatTime(image.UpdatedAt))
		}
		rows = append(rows, row)
	}
	return rows
}

func projectRows(projects []projectInfo) [][]string {
	if len(projects) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(projects))
	for _, project := range projects {
		rows = append(rows, []string{
			project.Name,
			formatCount(project.ImageCount),
		})
	}
	return rows
}

func tagRows(tags []registry.Tag, spec registry.TagTableSpec) [][]string {
	if len(tags) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(tags))
	for _, tag := range tags {
		row := []string{tag.Name}
		if spec.ShowSize {
			row = append(row, formatSize(tag.SizeBytes))
		}
		if spec.ShowPushed {
			row = append(row, formatTime(tag.PushedAt))
		}
		if spec.ShowLastPulled {
			row = append(row, formatTime(tag.LastPulledAt))
		}
		rows = append(rows, row)
	}
	return rows
}

func historyRows(entries []registry.HistoryEntry, spec registry.HistoryTableSpec) [][]string {
	if len(entries) == 0 {
		return nil
	}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		comment := entry.Comment
		if comment == "" && entry.EmptyLayer {
			comment = "empty layer"
		}
		row := []string{
			formatHistoryCommand(entry.CreatedBy),
			formatTime(entry.CreatedAt),
		}
		if spec.ShowSize {
			row = append(row, formatSize(entry.SizeBytes))
		}
		if spec.ShowComment {
			row = append(row, firstNonEmpty(comment, "-"))
		}
		rows = append(rows, row)
	}
	return rows
}

func filterRows(headers []string, rows [][]string, filter string) listView {
	if len(rows) == 0 {
		return listView{headers: headers}
	}
	if filter == "" {
		indices := make([]int, len(rows))
		for i := range rows {
			indices[i] = i
		}
		return listView{headers: headers, rows: rows, indices: indices}
	}
	needle := strings.ToLower(filter)
	var filtered [][]string
	var indices []int
	for i, row := range rows {
		if len(row) == 0 {
			continue
		}
		if strings.Contains(strings.ToLower(row[0]), needle) {
			filtered = append(filtered, row)
			indices = append(indices, i)
		}
	}
	return listView{headers: headers, rows: filtered, indices: indices}
}

func toTableRows(rows [][]string) []table.Row {
	if len(rows) == 0 {
		return nil
	}
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		out = append(out, table.Row(row))
	}
	return out
}

func normalizeTableRows(rows []table.Row, columnCount int) []table.Row {
	if len(rows) == 0 || columnCount <= 0 {
		return rows
	}
	for i, row := range rows {
		if len(row) == columnCount {
			continue
		}
		if len(row) > columnCount {
			rows[i] = row[:columnCount]
			continue
		}
		padded := make(table.Row, columnCount)
		copy(padded, row)
		for j := len(row); j < columnCount; j++ {
			padded[j] = ""
		}
		rows[i] = padded
	}
	return rows
}

func makeColumns(focus Focus, width int, spec registry.TableSpec) []table.Column {
	contentWidth := func(columnCount int) int {
		if columnCount <= 0 {
			return maxInt(1, width)
		}
		// bubbles/table default cell style uses horizontal padding of 1 on each side.
		// Reserve that padding so the rendered table width matches the viewport width.
		available := width - (2 * columnCount)
		if available < columnCount {
			return columnCount
		}
		return available
	}

	timeWidth := 16
	countWidth := 6
	pullWidth := 6
	sizeWidth := 10
	commentWidth := 20

	switch focus {
	case FocusProjects:
		columnCount := 2
		content := contentWidth(columnCount)
		nameWidth := maxInt(1, content-countWidth)
		return []table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Images", Width: countWidth},
		}
	case FocusImages:
		fixed := 0
		columns := []table.Column{}
		if spec.Image.ShowTagCount {
			columns = append(columns, table.Column{Title: "Tags", Width: countWidth})
			fixed += countWidth
		}
		if spec.Image.ShowPulls {
			columns = append(columns, table.Column{Title: "Pulls", Width: pullWidth})
			fixed += pullWidth
		}
		if spec.Image.ShowUpdated {
			columns = append(columns, table.Column{Title: "Updated", Width: timeWidth})
			fixed += timeWidth
		}
		columnCount := len(columns) + 1
		content := contentWidth(columnCount)
		nameWidth := maxInt(1, content-fixed)
		return append([]table.Column{{Title: "Name", Width: nameWidth}}, columns...)
	case FocusHistory:
		columnCount := 2
		fixed := timeWidth
		if spec.History.ShowSize {
			columnCount++
			fixed += sizeWidth
		}
		if spec.History.ShowComment {
			columnCount++
			fixed += commentWidth
		}
		content := contentWidth(columnCount)
		commandWidth := maxInt(1, content-fixed)
		columns := []table.Column{
			{Title: "Command", Width: commandWidth},
			{Title: "Created", Width: timeWidth},
		}
		if spec.History.ShowSize {
			columns = append(columns, table.Column{Title: "Size", Width: sizeWidth})
		}
		if spec.History.ShowComment {
			columns = append(columns, table.Column{Title: "Comment", Width: commentWidth})
		}
		return columns
	case FocusDockerHubTags:
		fallthrough
	case FocusGitHubTags:
		fallthrough
	default:
		fixed := 0
		columns := []table.Column{}
		if spec.Tag.ShowSize {
			columns = append(columns, table.Column{Title: "Size", Width: sizeWidth})
			fixed += sizeWidth
		}
		if spec.Tag.ShowPushed {
			columns = append(columns, table.Column{Title: "Pushed", Width: timeWidth})
			fixed += timeWidth
		}
		if spec.Tag.ShowLastPulled {
			columns = append(columns, table.Column{Title: "Last Pull", Width: timeWidth})
			fixed += timeWidth
		}
		columnCount := len(columns) + 1
		content := contentWidth(columnCount)
		nameWidth := maxInt(1, content-fixed)
		return append([]table.Column{{Title: "Name", Width: nameWidth}}, columns...)
	}
}

func tableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		BorderBottom(true).
		Foreground(colorTitleText).
		Background(colorSurface2).
		Bold(true)
	styles.Cell = lipgloss.NewStyle().Padding(0, 1)
	styles.Selected = styles.Selected.
		Foreground(colorSelected).
		Background(colorAccent).
		Bold(true)
	return styles
}

func formatCount(value int) string {
	if value < 0 {
		return "-"
	}
	return fmt.Sprintf("%d", value)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Local().Format("2006-01-02 15:04")
}

func formatSize(sizeBytes int64) string {
	if sizeBytes < 0 {
		return "-"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(sizeBytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d B", sizeBytes)
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

func formatHistoryCommand(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func firstNonEmpty(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func listenLogs(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return logMsg(msg)
	}
}

func initClientCmd(host string, auth registry.Auth, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		client, err := registry.NewClientWithLogger(host, auth, logger)
		return initClientMsg{client: client, err: err}
	}
}

func (m *Model) appendLog(entry string) {
	if entry == "" {
		return
	}
	m.logs = append(m.logs, entry)
	if m.logMax > 0 && len(m.logs) > m.logMax {
		m.logs = m.logs[len(m.logs)-m.logMax:]
	}
}

func (m *Model) syncAuthFocus() {
	switch m.authFocus {
	case 0:
		m.usernameInput.Focus()
		m.passwordInput.Blur()
	case 1:
		m.passwordInput.Focus()
		m.usernameInput.Blur()
	default:
		m.usernameInput.Blur()
		m.passwordInput.Blur()
	}
}

func loadImagesCmd(client registry.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		images, err := client.ListImages(ctx)
		return imagesMsg{images: images, err: err}
	}
}

func loadProjectsCmd(client registry.ProjectClient) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		projects, err := client.ListProjects(ctx)
		return projectsMsg{projects: projects, err: err}
	}
}

func loadProjectImagesCmd(client registry.ProjectClient, project string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		images, err := client.ListProjectImages(ctx, project)
		return projectImagesMsg{project: project, images: images, err: err}
	}
}

func loadTagsCmd(client registry.Client, image string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tags, err := client.ListTags(ctx, image)
		return tagsMsg{tags: tags, err: err}
	}
}

func loadHistoryCmd(client registry.Client, image, tag string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		history, err := client.ListTagHistory(ctx, image, tag)
		return historyMsg{history: history, err: err}
	}
}

func loadDockerHubTagsFirstPageCmd(query string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewDockerHubClient(logger)
		page, err := client.SearchTagsPage(ctx, query)
		if err != nil {
			return dockerHubErrorMsg(err)
		}
		return dockerHubTagsMsg{
			tags:      page.Tags,
			image:     page.Image,
			next:      page.Next,
			rateLimit: page.RateLimit,
		}
	}
}

func loadDockerHubTagsNextPageCmd(image, next string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewDockerHubClient(logger)
		page, err := client.NextTagsPage(ctx, image, next)
		if err != nil {
			msg := dockerHubErrorMsg(err)
			msg.appendPage = true
			return msg
		}
		return dockerHubTagsMsg{
			tags:       page.Tags,
			image:      page.Image,
			next:       page.Next,
			rateLimit:  page.RateLimit,
			appendPage: true,
		}
	}
}

func loadGitHubTagsFirstPageCmd(query string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewGitHubContainerClient(logger)
		page, err := client.SearchTagsPage(ctx, query)
		if err != nil {
			return githubTagsMsg{err: err}
		}
		return githubTagsMsg{
			tags:  page.Tags,
			image: page.Image,
			next:  page.Next,
		}
	}
}

func loadGitHubTagsNextPageCmd(image, next string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewGitHubContainerClient(logger)
		page, err := client.NextTagsPage(ctx, image, next)
		if err != nil {
			return githubTagsMsg{err: err, appendPage: true}
		}
		return githubTagsMsg{
			tags:       page.Tags,
			image:      page.Image,
			next:       page.Next,
			appendPage: true,
		}
	}
}

func loadDockerHubHistoryCmd(image, tag string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewDockerHubClient(logger)
		history, err := client.ListTagHistory(ctx, image, tag)
		return historyMsg{history: history, err: err}
	}
}

func loadGitHubHistoryCmd(image, tag string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewGitHubContainerClient(logger)
		history, err := client.ListTagHistory(ctx, image, tag)
		return historyMsg{history: history, err: err}
	}
}

func dockerHubErrorMsg(err error) dockerHubTagsMsg {
	msg := dockerHubTagsMsg{err: err}
	var rateErr *registry.DockerHubRateLimitError
	if errors.As(err, &rateErr) {
		msg.rateLimit = rateErr.RateLimit
		msg.retryAfter = rateErr.RetryAfter
	}
	return msg
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) authUI() registry.AuthUI {
	if m.provider == nil {
		return registry.AuthUI{}
	}
	return m.provider.AuthUI(m.auth)
}

func (m Model) authFieldCount() int {
	ui := m.authUI()
	if ui.ShowRemember {
		return 3
	}
	if ui.ShowUsername || ui.ShowPassword {
		return 2
	}
	return 0
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func lineCount(value string) int {
	if value == "" {
		return 0
	}
	return strings.Count(value, "\n") + 1
}

func truncateLogLine(value string, width int) string {
	if width <= 0 {
		return ""
	}
	line := strings.TrimSpace(strings.ReplaceAll(value, "\n", " "))
	if len(line) <= width {
		return line
	}
	if width <= 3 {
		return line[:width]
	}
	return line[:width-3] + "..."
}

func equalTableColumns(a, b []table.Column) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Title != b[i].Title || a[i].Width != b[i].Width {
			return false
		}
	}
	return true
}

func equalTableRows(a, b []table.Row) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
