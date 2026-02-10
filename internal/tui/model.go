package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/scottbass3/beacon/internal/registry"
)

type Focus int

const (
	FocusProjects Focus = iota
	FocusImages
	FocusTags
	FocusHistory
	FocusDockerHubTags
)

const (
	defaultTableHeight = 10
	minTableHeight     = 4
	maxLogLines        = 25
	maxFilterWidth     = 40
)

type Model struct {
	width  int
	height int

	status  string
	focus   Focus
	context string

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

	commandActive    bool
	commandInput     textinput.Model
	commandMatches   []string
	commandIndex     int
	commandError     string
	contexts         []ContextOption
	contextNameIndex map[string]int

	debug  bool
	logCh  <-chan string
	logs   []string
	logMax int
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
	tags  []registry.Tag
	image string
	err   error
}

type projectInfo struct {
	Name       string
	ImageCount int
}

type initClientMsg struct {
	client registry.Client
	err    error
}

type logMsg string

var (
	colorPrimary  = lipgloss.Color("62")
	colorMuted    = lipgloss.Color("241")
	colorAccent   = lipgloss.Color("204")
	colorSelected = lipgloss.Color("229")
)

var (
	titleStyle     = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	labelStyle     = lipgloss.NewStyle().Foreground(colorMuted)
	helpStyle      = lipgloss.NewStyle().Foreground(colorMuted)
	filterStyle    = lipgloss.NewStyle().Foreground(colorAccent)
	emptyStyle     = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	logTitleStyle  = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	authTitleStyle = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	authErrorStyle = lipgloss.NewStyle().Foreground(colorAccent)
)

type ContextOption struct {
	Name string
	Host string
	Auth registry.Auth
}

func NewModel(registryHost string, auth registry.Auth, logger registry.RequestLogger, debug bool, logCh <-chan string, contexts []ContextOption, currentContext string) Model {
	status := "Registry not configured"
	if registryHost != "" {
		status = fmt.Sprintf("Registry: %s", registryHost)
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

	commandInput := textinput.New()
	commandInput.Prompt = ":"
	commandInput.Placeholder = "context <name> | dockerhub"
	commandInput.CharLimit = 64
	commandInput.Blur()

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
	if provider.NeedsAuthPrompt(auth) {
		username.Focus()
	}

	contextIndex := make(map[string]int, len(contexts))
	for i, ctx := range contexts {
		contextIndex[strings.ToLower(ctx.Name)] = i
	}

	return Model{
		status: status,
		focus: func() Focus {
			if provider.TableSpec().SupportsProjects {
				return FocusProjects
			}
			return FocusImages
		}(),
		context:          currentContext,
		registryHost:     registryHost,
		auth:             auth,
		provider:         provider,
		authRequired:     provider.NeedsAuthPrompt(auth),
		authFocus:        0,
		usernameInput:    username,
		passwordInput:    password,
		remember:         remember,
		filterInput:      filter,
		table:            tbl,
		dockerHubInput:   dockerHubInput,
		commandInput:     commandInput,
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
	if m.registryHost != "" && !m.authRequired {
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
		if !m.commandActive && (msg.String() == ":" || (len(msg.Runes) == 1 && msg.Runes[0] == ':')) {
			return m.enterCommandMode()
		}
		if m.commandActive {
			return m.handleCommandKey(msg)
		}
		if m.authRequired && m.registryClient == nil {
			return m.handleAuthKey(msg)
		}
		if m.dockerHubActive {
			return m.handleDockerHubKey(msg)
		}
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncTable()
	case imagesMsg:
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
		if msg.err != nil {
			m.status = fmt.Sprintf("Error searching Docker Hub: %v", msg.err)
			m.syncTable()
			return m, nil
		}
		m.dockerHubTags = msg.tags
		m.dockerHubImage = msg.image
		m.focus = FocusDockerHubTags
		m.status = fmt.Sprintf("Docker Hub: %s (%d tags)", msg.image, len(msg.tags))
		m.clearFilter()
		m.syncTable()
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
	if m.authRequired && m.registryClient == nil {
		return m.renderAuth()
	}
	if m.dockerHubActive {
		header := m.renderDockerHubHeader()
		body := m.renderBody()
		footer := helpStyle.Render("Keys: s search  / filter  : command  r refresh  esc back  j/k or up/down move  q quit")
		sections := []string{header, body, footer}
		if m.debug {
			sections = append(sections, m.renderLogs())
		}
		return strings.Join(sections, "\n\n")
	}
	header := m.renderHeader()
	body := m.renderBody()
	footer := helpStyle.Render("Keys: q quit  esc up/clear  / filter  : command  r refresh  enter open  j/k or up/down move")
	sections := []string{header, body, footer}
	if m.debug {
		sections = append(sections, m.renderLogs())
	}
	return strings.Join(sections, "\n\n")
}

func (m Model) renderHeader() string {
	lines := []string{
		titleStyle.Render("Beacon"),
		labelStyle.Render(fmt.Sprintf("Status: %s", m.status)),
		labelStyle.Render(fmt.Sprintf("Layer: %s", focusLabel(m.focus))),
	}
	if m.context != "" {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Context: %s", m.context)))
	}
	if path := m.breadcrumb(); path != "" {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Path: %s", path)))
	}
	if m.filterActive {
		lines = append(lines, filterStyle.Render(m.filterInput.View()))
	} else if value := m.filterInput.Value(); value != "" {
		lines = append(lines, filterStyle.Render("Filter: "+value))
	}
	if m.commandActive || m.commandInput.Value() != "" {
		lines = append(lines, filterStyle.Render(m.commandInput.View()))
		if len(m.commandMatches) > 0 {
			lines = append(lines, labelStyle.Render("Commands: "+strings.Join(m.commandMatches, " ")))
		}
		if m.commandError != "" {
			lines = append(lines, authErrorStyle.Render(m.commandError))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderDockerHubHeader() string {
	lines := []string{
		titleStyle.Render("Beacon"),
		labelStyle.Render("Mode: Docker Hub"),
		labelStyle.Render(fmt.Sprintf("Status: %s", m.status)),
	}
	if m.context != "" {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Context: %s", m.context)))
	}
	if m.dockerHubImage != "" {
		lines = append(lines, labelStyle.Render(fmt.Sprintf("Image: %s", m.dockerHubImage)))
	}
	searchLine := m.dockerHubInput.View()
	if m.dockerHubInputFocus {
		searchLine = filterStyle.Render(searchLine)
	}
	lines = append(lines, searchLine)
	if m.filterActive {
		lines = append(lines, filterStyle.Render(m.filterInput.View()))
	} else if value := m.filterInput.Value(); value != "" {
		lines = append(lines, filterStyle.Render("Filter: "+value))
	}
	if m.commandActive || m.commandInput.Value() != "" {
		lines = append(lines, filterStyle.Render(m.commandInput.View()))
		if len(m.commandMatches) > 0 {
			lines = append(lines, labelStyle.Render("Commands: "+strings.Join(m.commandMatches, " ")))
		}
		if m.commandError != "" {
			lines = append(lines, authErrorStyle.Render(m.commandError))
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderAuth() string {
	lines := []string{
		authTitleStyle.Render("Beacon"),
		labelStyle.Render(fmt.Sprintf("Registry: %s", m.registryHost)),
		labelStyle.Render("Authentication required"),
	}
	if m.authError != "" {
		lines = append(lines, authErrorStyle.Render(m.authError))
	}

	username := m.usernameInput.View()
	password := m.passwordInput.View()
	remember := ""
	if m.authUI().ShowRemember {
		remember = "[ ] Remember"
		if m.remember {
			remember = "[x] Remember"
		}
	}

	if m.authFocus == 0 {
		username = filterStyle.Render(username)
	}
	if m.authFocus == 1 {
		password = filterStyle.Render(password)
	}
	if m.authFocus == 2 && m.authUI().ShowRemember {
		remember = filterStyle.Render(remember)
	}

	help := "Keys: tab/shift+tab move  enter submit  q quit"
	if m.authUI().ShowRemember {
		help = "Keys: tab/shift+tab move  space toggle  enter submit  q quit"
	}

	lines = append(lines,
		"",
		labelStyle.Render("Username:"),
		username,
		labelStyle.Render("Password:"),
		password,
		remember,
		"",
		helpStyle.Render(help),
	)

	return strings.Join(lines, "\n")
}

func (m Model) renderBody() string {
	view := m.table.View()
	if len(m.table.Rows()) == 0 {
		return view + "\n" + emptyStyle.Render("(empty)")
	}
	return view
}

func (m Model) renderLogs() string {
	var b strings.Builder
	b.WriteString(logTitleStyle.Render("Requests"))
	b.WriteString("\n")
	b.WriteString(strings.Repeat("-", len("Requests")))
	if len(m.logs) == 0 {
		b.WriteString("\n")
		b.WriteString(emptyStyle.Render("(no requests yet)"))
		return b.String()
	}
	for _, entry := range m.logs {
		b.WriteString("\n")
		b.WriteString(entry)
	}
	return b.String()
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.filterActive {
		switch msg.String() {
		case "esc":
			m.clearFilter()
			m.syncTable()
			return m, nil
		case ":":
			m.clearFilter()
			return m.enterCommandMode()
		case "enter":
			m.filterActive = false
			m.filterInput.Blur()
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
		return m, tea.Quit
	case "esc":
		return m, m.handleEscape()
	case "/":
		m.filterActive = true
		m.filterInput.Focus()
		m.filterInput.CursorEnd()
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
			m.clearFilter()
			return m.enterCommandMode()
		case "enter":
			m.filterActive = false
			m.filterInput.Blur()
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
		return m, tea.Quit
	case "esc":
		return m.exitDockerHubMode()
	case ":":
		return m.enterCommandMode()
	case "enter":
		query := strings.TrimSpace(m.dockerHubInput.Value())
		if query == "" {
			m.status = "Enter an image name to search Docker Hub"
			return m, nil
		}
		m.status = fmt.Sprintf("Searching Docker Hub for %s...", query)
		return m, loadDockerHubTagsCmd(query, m.logger)
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
		return m, nil
	case "r":
		return m, m.refreshDockerHub()
	}

	if len(msg.Runes) == 1 && msg.Runes[0] == ':' {
		return m.enterCommandMode()
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

func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
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
	m.commandError = ""
	m.commandMatches = nil
	return m, nil
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
	m.syncTable()

	cmdName, args := parseCommand(input)
	switch cmdName {
	case "context", "ctx":
		if len(args) == 0 {
			m.status = fmt.Sprintf("Usage: :ctx <name>. Available: %s", strings.Join(contextNames(m.contexts), ", "))
			return m, nil
		}
		return m.switchContext(strings.Join(args, " "))
	case "dockerhub", "hub":
		if len(args) > 0 {
			m.dockerHubInput.SetValue(strings.Join(args, " "))
			m.dockerHubInput.CursorEnd()
		}
		return m.enterDockerHubMode()
	default:
		m.status = fmt.Sprintf("Unknown command: %s", cmdName)
		return m, nil
	}
}

func (m Model) switchContext(name string) (tea.Model, tea.Cmd) {
	index, ok := m.contextNameIndex[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		m.commandError = fmt.Sprintf("Unknown context: %s", name)
		return m, nil
	}
	ctx := m.contexts[index]
	if ctx.Host == "" {
		m.commandError = fmt.Sprintf("Context %s has no registry configured", ctx.Name)
		return m, nil
	}

	m.commandActive = false
	m.commandInput.Blur()
	m.commandError = ""
	m.commandMatches = nil

	m.context = ctx.Name
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
	m.dockerHubImage = ""
	m.dockerHubTags = nil
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
	candidates := []string{"context", "ctx", "dockerhub", "hub"}
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

func (m Model) handleAuthKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
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
	m.focus = m.dockerHubPrevFocus
	if m.dockerHubPrevStatus != "" {
		m.status = m.dockerHubPrevStatus
	}
	m.clearFilter()
	m.syncTable()
	return m, nil
}

func (m *Model) refreshCurrent() tea.Cmd {
	if m.dockerHubActive {
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
			return loadProjectsCmd(projectClient)
		}
		m.status = fmt.Sprintf("Refreshing images from %s...", m.registryHost)
		return loadImagesCmd(m.registryClient)
	case FocusImages:
		if m.registryClient == nil {
			m.status = "Registry not configured"
			return nil
		}
		if m.hasSelectedProject {
			if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
				m.status = fmt.Sprintf("Refreshing images for %s...", m.selectedProject)
				return loadProjectImagesCmd(projectClient, m.selectedProject)
			}
		}
		m.status = fmt.Sprintf("Refreshing images from %s...", m.registryHost)
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
					return loadProjectImagesCmd(projectClient, m.selectedProject)
				}
			}
			m.status = fmt.Sprintf("Refreshing images from %s...", m.registryHost)
			return loadImagesCmd(m.registryClient)
		}
		m.status = fmt.Sprintf("Refreshing tags for %s...", m.selectedImage.Name)
		return loadTagsCmd(m.registryClient, m.selectedImage.Name)
	case FocusHistory:
		if !m.hasSelectedTag {
			if m.registryClient == nil {
				m.status = "Registry not configured"
				return nil
			}
			m.status = fmt.Sprintf("Refreshing tags for %s...", m.selectedImage.Name)
			return loadTagsCmd(m.registryClient, m.selectedImage.Name)
		}
		m.status = fmt.Sprintf("Refreshing history for %s:%s...", m.selectedImage.Name, m.selectedTag.Name)
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
	m.status = fmt.Sprintf("Searching Docker Hub for %s...", query)
	return loadDockerHubTagsCmd(query, m.logger)
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
		m.selectedProject = selected.Name
		m.hasSelectedProject = true
		m.images = nil
		m.selectedImage = registry.Image{}
		m.hasSelectedImage = false
		m.tags = nil
		m.focus = FocusImages
		if projectClient, ok := m.registryClient.(registry.ProjectClient); ok {
			m.status = fmt.Sprintf("Loading images for %s...", selected.Name)
			m.clearFilter()
			m.syncTable()
			return loadProjectImagesCmd(projectClient, selected.Name)
		}
		m.status = fmt.Sprintf("Project: %s", selected.Name)
		m.clearFilter()
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
		m.focus = FocusTags
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
			return loadProjectsCmd(projectClient)
		}
	}
	m.status = fmt.Sprintf("Connecting to %s...", m.registryHost)
	return loadImagesCmd(m.registryClient)
}

func (m *Model) syncTable() {
	list := m.listView()
	width := m.width
	if width <= 0 {
		width = 80
	}
	columns := makeColumns(m.focus, width, m.effectiveTableSpec())
	rows := normalizeTableRows(toTableRows(list.rows), len(columns))
	// Clear rows before changing columns to avoid mismatched row lengths during UpdateViewport.
	m.table.SetRows(nil)
	m.table.SetColumns(columns)
	m.table.SetRows(rows)
	m.table.SetHeight(m.tableHeight())
	m.table.SetWidth(maxInt(10, width-2))
	cursor := m.table.Cursor()
	if len(list.rows) == 0 {
		m.table.SetCursor(0)
	} else if cursor >= len(list.rows) {
		m.table.SetCursor(len(list.rows) - 1)
	}

	filterWidth := clampInt(width-10, 10, maxFilterWidth)
	m.filterInput.Width = filterWidth
	m.dockerHubInput.Width = filterWidth
	m.commandInput.Width = filterWidth
}

func (m Model) tableHeight() int {
	if m.height <= 0 {
		return defaultTableHeight
	}

	headerLines := 0
	if m.dockerHubActive {
		headerLines = 3
		if m.dockerHubImage != "" {
			headerLines++
		}
		if m.dockerHubInputFocus || m.dockerHubInput.Value() != "" {
			headerLines++
		}
	} else {
		headerLines = 3
		if m.breadcrumb() != "" {
			headerLines++
		}
	}
	if m.filterActive || m.filterInput.Value() != "" {
		headerLines++
	}
	if m.commandActive || m.commandInput.Value() != "" {
		headerLines++
		if len(m.commandMatches) > 0 || m.commandError != "" {
			headerLines++
		}
	}

	footerLines := 1
	logLines := 0
	if m.debug {
		logLines = 2 + minInt(len(m.logs), m.logMax)
	}

	padding := 2
	available := m.height - headerLines - footerLines - logLines - padding
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
	default:
		return filterRows(tagHeaders(spec.Tag), tagRows(m.tags, spec.Tag), filter)
	}
}

func imageHeaders(spec registry.ImageTableSpec) []string {
	headers := []string{"Image"}
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
	return []string{"Project", "Images"}
}

func tagHeaders(spec registry.TagTableSpec) []string {
	headers := []string{"Tag"}
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
	spacing := 3
	padding := 4
	available := maxInt(40, width-padding)

	timeWidth := 16
	countWidth := 6
	pullWidth := 6
	sizeWidth := 10
	commentWidth := 20

	switch focus {
	case FocusProjects:
		columnCount := 2
		spacingTotal := spacing * (columnCount - 1)
		content := maxInt(20, available-spacingTotal)
		nameWidth := maxInt(12, content-countWidth)
		return []table.Column{
			{Title: "Project", Width: nameWidth},
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
		spacingTotal := spacing * (columnCount - 1)
		content := maxInt(20, available-spacingTotal)
		nameWidth := maxInt(12, content-fixed)
		return append([]table.Column{{Title: "Image", Width: nameWidth}}, columns...)
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
		spacingTotal := spacing * (columnCount - 1)
		content := maxInt(20, available-spacingTotal)
		commandWidth := maxInt(12, content-fixed)
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
		spacingTotal := spacing * (columnCount - 1)
		content := maxInt(20, available-spacingTotal)
		nameWidth := maxInt(12, content-fixed)
		return append([]table.Column{{Title: "Tag", Width: nameWidth}}, columns...)
	}
}

func tableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Header = styles.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Foreground(colorMuted).
		Bold(true)
	styles.Selected = styles.Selected.
		Foreground(colorSelected).
		Background(colorPrimary).
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

func loadDockerHubTagsCmd(query string, logger registry.RequestLogger) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		client := registry.NewDockerHubClient(logger)
		tags, image, err := client.SearchTags(ctx, query)
		return dockerHubTagsMsg{tags: tags, image: image, err: err}
	}
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
