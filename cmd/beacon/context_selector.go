package main

import (
	"errors"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/scottbass3/beacon/internal/config"
)

type contextItem struct {
	ctx config.Context
}

func (i contextItem) Title() string {
	if i.ctx.Name != "" {
		return i.ctx.Name
	}
	return i.ctx.Registry
}

func (i contextItem) Description() string {
	return i.ctx.Registry
}

func (i contextItem) FilterValue() string {
	return i.Title()
}

type contextSelectorModel struct {
	list   list.Model
	choice *config.Context
	err    error
}

func newContextSelectorModel(contexts []config.Context) contextSelectorModel {
	items := make([]list.Item, 0, len(contexts))
	for _, ctx := range contexts {
		items = append(items, contextItem{ctx: ctx})
	}

	delegate := list.NewDefaultDelegate()
	lst := list.New(items, delegate, 0, 0)
	lst.Title = "Select Context"
	lst.SetShowStatusBar(false)
	lst.SetFilteringEnabled(true)
	lst.SetShowHelp(true)
	lst.Styles.Title = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	lst.Styles.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	lst.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	return contextSelectorModel{list: lst}
}

func (m contextSelectorModel) Init() tea.Cmd {
	return nil
}

func (m contextSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		height := msg.Height - 2
		if height < 4 {
			height = 4
		}
		m.list.SetSize(msg.Width, height)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.err = errors.New("selection canceled")
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(contextItem); ok {
				choice := item.ctx
				m.choice = &choice
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m contextSelectorModel) View() string {
	return m.list.View()
}

func selectContextTUI(contexts []config.Context) (config.Context, error) {
	model := newContextSelectorModel(contexts)
	program := tea.NewProgram(model, tea.WithAltScreen())
	result, err := program.Run()
	if err != nil {
		return config.Context{}, err
	}

	finalModel, ok := result.(contextSelectorModel)
	if !ok {
		return config.Context{}, errors.New("context selection failed")
	}
	if finalModel.choice == nil {
		if finalModel.err != nil {
			return config.Context{}, finalModel.err
		}
		return config.Context{}, errors.New("selection canceled")
	}

	return *finalModel.choice, nil
}
