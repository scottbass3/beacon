package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/config"
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
	)
	if err := program.Start(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resolveRegistry(registryHost, configPath string) (registry.Auth, string, []tui.ContextOption, string, string, error) {
	path := configPath
	if path == "" {
		path = config.DefaultPath()
	}

	cfg, err := config.Ensure(path)
	if err != nil {
		return registry.Auth{}, "", nil, "", path, err
	}

	contexts := make([]tui.ContextOption, 0, len(cfg.Contexts))
	for _, ctx := range cfg.Contexts {
		contexts = append(contexts, toContextOption(ctx))
	}

	if registryHost != "" {
		return registry.Auth{
			Kind: "registry_v2",
			RegistryV2: registry.RegistryV2Auth{
				Anonymous: true,
			},
		}, registryHost, contexts, "", path, nil
	}

	if len(cfg.Contexts) == 0 {
		return registry.Auth{}, "", contexts, "", path, nil
	}

	ctx := cfg.Contexts[0]
	current := ctx.Name
	return toContextOption(ctx).Auth, ctx.Registry, contexts, current, path, nil
}

func toContextOption(ctx config.Context) tui.ContextOption {
	auth := registry.Auth{Kind: ctx.Kind}
	switch strings.ToLower(ctx.Kind) {
	case "registry_v2", "registry", "v2":
		auth.RegistryV2.Anonymous = ctx.Anonymous
		auth.RegistryV2.Service = ctx.Service
	case "harbor":
		auth.Harbor.Anonymous = ctx.Anonymous
		auth.Harbor.Service = ctx.Service
	}

	return tui.ContextOption{
		Name: ctx.Name,
		Host: ctx.Registry,
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
