package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type shortcutAction int

const (
	shortcutOpenHelp shortcutAction = iota
	shortcutQuit
	shortcutForceQuit
	shortcutOpenCommand
	shortcutOpenFilter
	shortcutRefresh
	shortcutBack
	shortcutExitExternalMode
	shortcutFocusExternalSearch
	shortcutCopyImageTag

	shortcutOpenProjectImages
	shortcutOpenImageTags
	shortcutOpenTagHistory
	shortcutOpenExternalTagHistory

	shortcutTypeCommand
	shortcutCommandAutocomplete
	shortcutCommandPrevSuggestion
	shortcutCommandNextSuggestion
	shortcutCommandCycleSuggestions
	shortcutCommandRun
	shortcutCommandCancel

	shortcutTypeFilter
	shortcutApplyFilter
	shortcutClearFilter

	shortcutTypeExternalQuery
	shortcutSearchExternal

	shortcutCloseHelp

	shortcutMoveUp
	shortcutMoveDown
	shortcutMovePageUp
	shortcutMovePageDown
	shortcutMoveHalfUp
	shortcutMoveHalfDown
	shortcutMoveTop
	shortcutMoveBottom
)

type shortcutDefinition struct {
	Keys        []string
	HelpKeys    string
	HintKeys    string
	Description string
	HintLabel   string
}

var shortcutDefinitions = map[shortcutAction]shortcutDefinition{
	shortcutOpenHelp: {
		Keys:        []string{"?", "f1"},
		HelpKeys:    "?/F1",
		HintKeys:    "?",
		Description: "Open help",
		HintLabel:   "help",
	},
	shortcutQuit: {
		Keys:        []string{"q", "ctrl+c"},
		HelpKeys:    "q/Ctrl+C",
		HintKeys:    "q",
		Description: "Quit",
		HintLabel:   "quit",
	},
	shortcutForceQuit: {
		Keys:        []string{"ctrl+c"},
		HelpKeys:    "Ctrl+C",
		HintKeys:    "ctrl+c",
		Description: "Quit",
		HintLabel:   "quit",
	},
	shortcutOpenCommand: {
		Keys:        []string{":"},
		HelpKeys:    ":",
		HintKeys:    ":",
		Description: "Open command input",
		HintLabel:   "command",
	},
	shortcutOpenFilter: {
		Keys:        []string{"/"},
		HelpKeys:    "/",
		HintKeys:    "/",
		Description: "Filter current list",
		HintLabel:   "filter",
	},
	shortcutRefresh: {
		Keys:        []string{"r"},
		HelpKeys:    "r",
		HintKeys:    "r",
		Description: "Refresh current data",
		HintLabel:   "refresh",
	},
	shortcutBack: {
		Keys:        []string{"esc"},
		HelpKeys:    "Esc",
		HintKeys:    "esc",
		Description: "Go back one level",
		HintLabel:   "back",
	},
	shortcutExitExternalMode: {
		Keys:        []string{"esc"},
		HelpKeys:    "Esc",
		HintKeys:    "esc",
		Description: "Exit external mode",
		HintLabel:   "exit",
	},
	shortcutFocusExternalSearch: {
		Keys:        []string{"s"},
		HelpKeys:    "s",
		HintKeys:    "s",
		Description: "Focus search input",
		HintLabel:   "search",
	},
	shortcutCopyImageTag: {
		Keys:        []string{"c"},
		HelpKeys:    "c",
		HintKeys:    "c",
		Description: "Copy selected image:tag",
		HintLabel:   "copy",
	},
	shortcutOpenProjectImages: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Open selected project images",
		HintLabel:   "open",
	},
	shortcutOpenImageTags: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Open selected image tags",
		HintLabel:   "open",
	},
	shortcutOpenTagHistory: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Open selected tag history",
		HintLabel:   "open",
	},
	shortcutOpenExternalTagHistory: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Open selected tag history",
		HintLabel:   "open",
	},
	shortcutTypeCommand: {
		HelpKeys:    "Type",
		HintKeys:    "type",
		Description: "Set command text",
		HintLabel:   "command",
	},
	shortcutCommandAutocomplete: {
		Keys:        []string{"tab"},
		HelpKeys:    "Tab",
		HintKeys:    "tab",
		Description: "Autocomplete command",
		HintLabel:   "complete",
	},
	shortcutCommandPrevSuggestion: {
		Keys: []string{"up"},
	},
	shortcutCommandNextSuggestion: {
		Keys: []string{"down"},
	},
	shortcutCommandCycleSuggestions: {
		HelpKeys:    "Up/Down",
		HintKeys:    "up/down",
		Description: "Cycle command suggestions",
		HintLabel:   "cycle",
	},
	shortcutCommandRun: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Run command",
		HintLabel:   "run",
	},
	shortcutCommandCancel: {
		Keys:        []string{"esc"},
		HelpKeys:    "Esc",
		HintKeys:    "esc",
		Description: "Close command input",
		HintLabel:   "cancel",
	},
	shortcutTypeFilter: {
		HelpKeys:    "Type",
		HintKeys:    "type",
		Description: "Set filter text",
		HintLabel:   "text",
	},
	shortcutApplyFilter: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Apply and close filter input",
		HintLabel:   "apply",
	},
	shortcutClearFilter: {
		Keys:        []string{"esc"},
		HelpKeys:    "Esc",
		HintKeys:    "esc",
		Description: "Clear filter",
		HintLabel:   "clear",
	},
	shortcutTypeExternalQuery: {
		HelpKeys:    "Type",
		HintKeys:    "type",
		Description: "Set image query",
		HintLabel:   "image",
	},
	shortcutSearchExternal: {
		Keys:        []string{"enter"},
		HelpKeys:    "Enter",
		HintKeys:    "enter",
		Description: "Search image tags",
		HintLabel:   "search",
	},
	shortcutCloseHelp: {
		Keys:        []string{"esc", "?", "f1", "enter"},
		HelpKeys:    "Esc/?/F1/Enter",
		HintKeys:    "esc/?",
		Description: "Close help",
		HintLabel:   "close",
	},
	shortcutMoveUp: {
		Keys:        []string{"up", "k"},
		HelpKeys:    "Up/k",
		Description: "Move selection up",
	},
	shortcutMoveDown: {
		Keys:        []string{"down", "j"},
		HelpKeys:    "Down/j",
		Description: "Move selection down",
	},
	shortcutMovePageUp: {
		Keys:        []string{"pgup", "b"},
		HelpKeys:    "PgUp/b",
		Description: "Move one page up",
	},
	shortcutMovePageDown: {
		Keys:        []string{"pgdown", "f", " "},
		HelpKeys:    "PgDn/f/Space",
		Description: "Move one page down",
	},
	shortcutMoveHalfUp: {
		Keys:        []string{"ctrl+u", "u"},
		HelpKeys:    "Ctrl+U/u",
		Description: "Move half page up",
	},
	shortcutMoveHalfDown: {
		Keys:        []string{"ctrl+d", "d"},
		HelpKeys:    "Ctrl+D/d",
		Description: "Move half page down",
	},
	shortcutMoveTop: {
		Keys:        []string{"home", "g"},
		HelpKeys:    "Home/g",
		Description: "Jump to top",
	},
	shortcutMoveBottom: {
		Keys:        []string{"end", "G"},
		HelpKeys:    "End/G",
		Description: "Jump to bottom",
	},
}

type shortcutPage int

const (
	shortcutPageHelp shortcutPage = iota
	shortcutPageCommandInput
	shortcutPageFilterInput
	shortcutPageDockerHubSearchInput
	shortcutPageGitHubSearchInput
	shortcutPageProjects
	shortcutPageImages
	shortcutPageTags
	shortcutPageHistory
	shortcutPageDockerHubTags
	shortcutPageGitHubTags
)

var listHelpActions = []shortcutAction{
	shortcutOpenHelp,
	shortcutOpenCommand,
	shortcutQuit,
	shortcutOpenFilter,
	shortcutMoveUp,
	shortcutMoveDown,
	shortcutMovePageUp,
	shortcutMovePageDown,
	shortcutMoveHalfUp,
	shortcutMoveHalfDown,
	shortcutMoveTop,
	shortcutMoveBottom,
	shortcutRefresh,
}

var listHintActions = []shortcutAction{
	shortcutOpenHelp,
	shortcutOpenCommand,
	shortcutOpenFilter,
	shortcutRefresh,
	shortcutQuit,
}

func isShortcut(msg tea.KeyMsg, action shortcutAction) bool {
	def, ok := shortcutDefinitions[action]
	if !ok || len(def.Keys) == 0 {
		return false
	}
	key := msg.String()
	for _, candidate := range def.Keys {
		if key == candidate {
			return true
		}
	}
	return false
}

func (m Model) shortcutPage(includeHelpOverlay bool) shortcutPage {
	if includeHelpOverlay && m.helpActive {
		return shortcutPageHelp
	}
	if m.commandActive {
		return shortcutPageCommandInput
	}
	if m.filterActive {
		return shortcutPageFilterInput
	}
	if m.dockerHubActive && m.dockerHubInputFocus {
		return shortcutPageDockerHubSearchInput
	}
	if m.githubActive && m.githubInputFocus {
		return shortcutPageGitHubSearchInput
	}
	switch m.focus {
	case FocusProjects:
		return shortcutPageProjects
	case FocusImages:
		return shortcutPageImages
	case FocusTags:
		return shortcutPageTags
	case FocusHistory:
		return shortcutPageHistory
	case FocusDockerHubTags:
		return shortcutPageDockerHubTags
	case FocusGitHubTags:
		return shortcutPageGitHubTags
	default:
		if m.dockerHubActive {
			return shortcutPageDockerHubTags
		}
		if m.githubActive {
			return shortcutPageGitHubTags
		}
		return shortcutPageImages
	}
}

func (m Model) shortcutPageTitle(includeHelpOverlay bool) string {
	switch m.shortcutPage(includeHelpOverlay) {
	case shortcutPageHelp:
		return "Help"
	case shortcutPageCommandInput:
		return "Command Input"
	case shortcutPageFilterInput:
		return "Filter Input"
	case shortcutPageDockerHubSearchInput:
		return "Docker Hub Search"
	case shortcutPageGitHubSearchInput:
		return "GHCR Search"
	case shortcutPageProjects:
		return "Projects"
	case shortcutPageImages:
		return "Images"
	case shortcutPageTags:
		return "Tags"
	case shortcutPageHistory:
		return "History"
	case shortcutPageDockerHubTags:
		return "Docker Hub Tags"
	case shortcutPageGitHubTags:
		return "GHCR Tags"
	default:
		return focusLabel(m.focus)
	}
}

func (m Model) currentPageHelpEntries() []helpEntry {
	return helpEntriesForActions(m.helpActionsForPage(m.shortcutPage(false)))
}

func (m Model) shortcutHintLine() string {
	page := m.shortcutPage(true)
	return hintLineForActions(m.hintPrefixForPage(page), m.hintActionsForPage(page))
}

func (m Model) hintPrefixForPage(page shortcutPage) string {
	switch page {
	case shortcutPageHelp:
		return "Help"
	case shortcutPageCommandInput:
		return "Command"
	case shortcutPageFilterInput:
		return "Filter"
	case shortcutPageDockerHubSearchInput:
		return "Docker Hub search"
	case shortcutPageGitHubSearchInput:
		return "GHCR search"
	default:
		return "Shortcuts"
	}
}

func (m Model) helpActionsForPage(page shortcutPage) []shortcutAction {
	switch page {
	case shortcutPageCommandInput:
		return []shortcutAction{
			shortcutTypeCommand,
			shortcutCommandAutocomplete,
			shortcutCommandCycleSuggestions,
			shortcutCommandRun,
			shortcutCommandCancel,
			shortcutQuit,
		}
	case shortcutPageFilterInput:
		return []shortcutAction{
			shortcutTypeFilter,
			shortcutApplyFilter,
			shortcutClearFilter,
			shortcutOpenCommand,
		}
	case shortcutPageDockerHubSearchInput, shortcutPageGitHubSearchInput:
		return []shortcutAction{
			shortcutTypeExternalQuery,
			shortcutSearchExternal,
			shortcutExitExternalMode,
			shortcutForceQuit,
		}
	case shortcutPageDockerHubTags:
		actions := cloneActions(listHelpActions)
		actions = append(actions,
			shortcutOpenExternalTagHistory,
			shortcutCopyImageTag,
			shortcutFocusExternalSearch,
			shortcutExitExternalMode,
		)
		return actions
	case shortcutPageGitHubTags:
		actions := cloneActions(listHelpActions)
		actions = append(actions,
			shortcutOpenExternalTagHistory,
			shortcutCopyImageTag,
			shortcutFocusExternalSearch,
			shortcutExitExternalMode,
		)
		return actions
	case shortcutPageProjects:
		actions := cloneActions(listHelpActions)
		return append(actions, shortcutOpenProjectImages, shortcutBack)
	case shortcutPageImages:
		actions := cloneActions(listHelpActions)
		return append(actions, shortcutOpenImageTags, shortcutBack)
	case shortcutPageTags:
		actions := cloneActions(listHelpActions)
		return append(actions, shortcutOpenTagHistory, shortcutCopyImageTag, shortcutBack)
	case shortcutPageHistory:
		actions := cloneActions(listHelpActions)
		if m.dockerHubActive || m.githubActive {
			actions = append(actions, shortcutFocusExternalSearch)
		}
		return append(actions, shortcutBack)
	default:
		return []shortcutAction{shortcutCloseHelp, shortcutQuit}
	}
}

func (m Model) hintActionsForPage(page shortcutPage) []shortcutAction {
	switch page {
	case shortcutPageHelp:
		return []shortcutAction{shortcutCloseHelp, shortcutQuit}
	case shortcutPageCommandInput:
		return []shortcutAction{
			shortcutCommandAutocomplete,
			shortcutCommandCycleSuggestions,
			shortcutCommandRun,
			shortcutCommandCancel,
			shortcutQuit,
		}
	case shortcutPageFilterInput:
		return []shortcutAction{
			shortcutTypeFilter,
			shortcutApplyFilter,
			shortcutClearFilter,
			shortcutOpenCommand,
		}
	case shortcutPageDockerHubSearchInput, shortcutPageGitHubSearchInput:
		return []shortcutAction{
			shortcutTypeExternalQuery,
			shortcutSearchExternal,
			shortcutExitExternalMode,
			shortcutForceQuit,
		}
	case shortcutPageDockerHubTags:
		actions := cloneActions(listHintActions)
		actions = append(actions,
			shortcutFocusExternalSearch,
			shortcutOpenExternalTagHistory,
			shortcutCopyImageTag,
			shortcutExitExternalMode,
		)
		return actions
	case shortcutPageGitHubTags:
		actions := cloneActions(listHintActions)
		actions = append(actions,
			shortcutFocusExternalSearch,
			shortcutOpenExternalTagHistory,
			shortcutCopyImageTag,
			shortcutExitExternalMode,
		)
		return actions
	case shortcutPageProjects:
		actions := cloneActions(listHintActions)
		return append(actions, shortcutOpenProjectImages, shortcutBack)
	case shortcutPageImages:
		actions := cloneActions(listHintActions)
		return append(actions, shortcutOpenImageTags, shortcutBack)
	case shortcutPageTags:
		actions := cloneActions(listHintActions)
		return append(actions, shortcutOpenTagHistory, shortcutCopyImageTag, shortcutBack)
	case shortcutPageHistory:
		actions := cloneActions(listHintActions)
		if m.dockerHubActive || m.githubActive {
			actions = append(actions, shortcutFocusExternalSearch)
		}
		return append(actions, shortcutBack)
	default:
		return []shortcutAction{shortcutOpenHelp, shortcutQuit}
	}
}

func helpEntriesForActions(actions []shortcutAction) []helpEntry {
	entries := make([]helpEntry, 0, len(actions))
	for _, action := range actions {
		def, ok := shortcutDefinitions[action]
		if !ok || def.HelpKeys == "" || def.Description == "" {
			continue
		}
		entries = append(entries, helpEntry{Keys: def.HelpKeys, Action: def.Description})
	}
	return entries
}

func hintLineForActions(prefix string, actions []shortcutAction) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		def, ok := shortcutDefinitions[action]
		if !ok || def.HintLabel == "" {
			continue
		}
		keys := def.HintKeys
		if keys == "" {
			keys = def.HelpKeys
		}
		if keys == "" {
			continue
		}
		parts = append(parts, keys+" "+def.HintLabel)
	}
	if len(parts) == 0 {
		return prefix
	}
	if prefix == "" {
		return strings.Join(parts, "   ")
	}
	return prefix + ": " + strings.Join(parts, "   ")
}

func cloneActions(actions []shortcutAction) []shortcutAction {
	if len(actions) == 0 {
		return nil
	}
	out := make([]shortcutAction, len(actions))
	copy(out, actions)
	return out
}
