package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type commandDescriptor struct {
	Name    string
	Aliases []string
	Help    []commandHelp
	Run     func(Model, []string) (tea.Model, tea.Cmd)
}

func commandRegistry() []commandDescriptor {
	return []commandDescriptor{
		{
			Name:    "help",
			Aliases: nil,
			Help: []commandHelp{
				{Command: "help", Usage: "Open the help page"},
			},
			Run: runHelpCommand,
		},
		{
			Name:    "context",
			Aliases: []string{"ctx"},
			Help: []commandHelp{
				{Command: "context", Usage: "Open context selection"},
				{Command: "context add", Usage: "Create a new context"},
				{Command: "context edit <name>", Usage: "Edit an existing context"},
				{Command: "context remove <name>", Usage: "Remove a context"},
				{Command: "context <name>", Usage: "Switch to context by name"},
			},
			Run: runContextCommand,
		},
		{
			Name:    "dockerhub",
			Aliases: []string{"dh", "hub"},
			Help: []commandHelp{
				{Command: "dockerhub", Usage: "Open Docker Hub mode"},
				{Command: "dockerhub <image>", Usage: "Search Docker Hub image tags"},
			},
			Run: runDockerHubCommand,
		},
		{
			Name:    "github",
			Aliases: []string{"ghcr"},
			Help: []commandHelp{
				{Command: "github", Usage: "Open GitHub Container Registry mode"},
				{Command: "github <image>", Usage: "Search GHCR image tags"},
				{Command: "ghcr", Usage: "Alias for github"},
				{Command: "ghcr <image>", Usage: "Alias search for GHCR tags"},
			},
			Run: runGitHubCommand,
		},
	}
}

func availableCommands() []commandHelp {
	registry := commandRegistry()
	entries := make([]commandHelp, 0, len(registry)*2)
	for _, cmd := range registry {
		entries = append(entries, cmd.Help...)
	}
	return entries
}

func resolveCommand(name string) (commandDescriptor, bool) {
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return commandDescriptor{}, false
	}
	for _, descriptor := range commandRegistry() {
		if descriptor.Name == needle {
			return descriptor, true
		}
		for _, alias := range descriptor.Aliases {
			if alias == needle {
				return descriptor, true
			}
		}
	}
	return commandDescriptor{}, false
}

func commandSuggestions() []string {
	registry := commandRegistry()
	out := make([]string, 0, len(registry)*2)
	for _, descriptor := range registry {
		out = append(out, descriptor.Name)
		out = append(out, descriptor.Aliases...)
	}
	return out
}

func matchCommands(prefix string) []string {
	candidates := commandSuggestions()
	if prefix == "" {
		return candidates
	}
	prefix = strings.ToLower(prefix)
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.HasPrefix(candidate, prefix) {
			out = append(out, candidate)
		}
	}
	return out
}

func runHelpCommand(m Model, _ []string) (tea.Model, tea.Cmd) {
	return m.openHelp()
}

func runContextCommand(m Model, args []string) (tea.Model, tea.Cmd) {
	return m.runContextCommand(args)
}

func runDockerHubCommand(m Model, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 {
		query := strings.Join(args, " ")
		model, _ := m.enterDockerHubMode()
		next := model.(Model)
		next.dockerHubInput.SetValue(query)
		next.dockerHubInput.CursorEnd()
		return next, next.searchDockerHub(query)
	}
	return m.enterDockerHubMode()
}

func runGitHubCommand(m Model, args []string) (tea.Model, tea.Cmd) {
	if len(args) > 0 {
		query := strings.Join(args, " ")
		model, _ := m.enterGitHubMode()
		next := model.(Model)
		next.githubInput.SetValue(query)
		next.githubInput.CursorEnd()
		return next, next.searchGitHub(query)
	}
	return m.enterGitHubMode()
}
