package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/scottbass3/beacon/internal/registry"
)

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
