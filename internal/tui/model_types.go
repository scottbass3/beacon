package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

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

	contextSelectionState
	contextFormState
	confirmState

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

	selectionState

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

	commandState
	helpActive       bool
	contexts         []ContextOption
	contextNameIndex map[string]int
	tableColumns     []table.Column

	debug  bool
	logCh  <-chan string
	logs   []string
	logMax int

	loadingCount int
}

type contextSelectionState struct {
	contextSelectionActive   bool
	contextSelectionRequired bool
	contextSelectionIndex    int
	contextSelectionError    string
}

type contextFormState struct {
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
}

type confirmState struct {
	confirmAction  confirmAction
	confirmTitle   string
	confirmMessage string
	confirmFocus   int
}

type selectionState struct {
	selectedProject    string
	hasSelectedProject bool
	selectedImage      registry.Image
	hasSelectedImage   bool
	selectedTag        registry.Tag
	hasSelectedTag     bool
}

type commandState struct {
	commandActive              bool
	commandInput               textinput.Model
	commandMatches             []string
	commandIndex               int
	commandError               string
	commandPrevFilterActive    bool
	commandPrevDockerHubSearch bool
	commandPrevGitHubSearch    bool
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

type ContextOption struct {
	Name string
	Host string
	Auth registry.Auth
}
