package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"

	"github.com/scottbass3/beacon/internal/registry"
)

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
