package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/scottbass3/beacon/internal/registry"
)

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
