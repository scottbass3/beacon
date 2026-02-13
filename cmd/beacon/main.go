package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/contextstore"
	"github.com/scottbass3/beacon/internal/registry"
	"github.com/scottbass3/beacon/internal/tui"
)

func main() {
	var registryHost string
	var configPath string
	var debug bool
	flag.StringVar(&registryHost, "registry", "", "Registry host (e.g. https://registry.example.com)")
	flag.StringVar(&configPath, "config", "", "Path to config file (defaults to $XDG_CONFIG_HOME/beacon/config.json)")
	flag.BoolVar(&debug, "debug", false, "Enable request logging")
	flag.Parse()

	logCh := make(chan string, 256)
	logger := registry.RequestLogger(nil)
	if debug {
		logger = makeRequestLogger(logCh)
	} else {
		close(logCh)
		logCh = nil
	}

	auth, host, contexts, currentContext, resolvedConfigPath, err := resolveRegistry(registryHost, configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	program := tea.NewProgram(
		tui.NewModel(host, auth, logger, debug, logCh, contexts, currentContext, resolvedConfigPath),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if err := program.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveRegistry(registryHost, configPath string) (registry.Auth, string, []tui.ContextOption, string, string, error) {
	store := contextstore.New(configPath)
	contextConfigs, err := store.Ensure()
	if err != nil {
		return registry.Auth{}, "", nil, "", store.Path(), err
	}

	contexts := make([]tui.ContextOption, 0, len(contextConfigs))
	for _, ctx := range contextConfigs {
		contexts = append(contexts, toContextOption(ctx))
	}

	if registryHost != "" {
		return registry.Auth{
			Kind: "registry_v2",
			RegistryV2: registry.RegistryV2Auth{
				Anonymous: true,
			},
		}, registryHost, contexts, "", store.Path(), nil
	}

	if len(contextConfigs) == 0 {
		return registry.Auth{}, "", contexts, "", store.Path(), nil
	}

	ctx := contextConfigs[0]
	current := ctx.Name
	return toContextOption(ctx).Auth, ctx.Host, contexts, current, store.Path(), nil
}

func toContextOption(ctx contextstore.Context) tui.ContextOption {
	auth := ctx.Auth
	auth.Normalize()
	return tui.ContextOption{
		Name: ctx.Name,
		Host: ctx.Host,
		Auth: auth,
	}
}

func makeRequestLogger(ch chan<- string) registry.RequestLogger {
	return func(log registry.RequestLog) {
		entry := formatRequestLog(log)
		select {
		case ch <- entry:
		default:
		}
	}
}

func formatRequestLog(log registry.RequestLog) string {
	var b strings.Builder
	b.WriteString(log.Method)
	b.WriteString(" ")
	b.WriteString(log.URL)
	if log.Status > 0 {
		b.WriteString(" -> ")
		b.WriteString(fmt.Sprintf("%d", log.Status))
	}
	if len(log.Headers) == 0 {
		return b.String()
	}

	b.WriteString(" | ")
	keys := make([]string, 0, len(log.Headers))
	for key := range log.Headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for i, key := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(strings.Join(log.Headers[key], ","))
	}
	return b.String()
}
