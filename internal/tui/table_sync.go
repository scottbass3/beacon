package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/scottbass3/beacon/internal/registry"
)

func (m *Model) syncTable() {
	list := m.listView()
	width := m.width
	if width <= 0 {
		width = defaultRenderWidth
	}
	filterWidth := clampInt(width-10, 10, maxFilterWidth)
	m.filterInput.Width = filterWidth
	m.dockerHubInput.Width = filterWidth
	m.githubInput.Width = filterWidth
	m.commandInput.Width = filterWidth

	tableWidth := maxInt(10, m.mainSectionContentWidth())
	columns := makeColumns(m.focus, tableWidth, m.effectiveTableSpec())
	rows := normalizeTableRows(toTableRows(list.rows), len(columns))
	columnsChanged := !equalTableColumns(m.tableColumns, columns)
	if columnsChanged {
		// Clear rows only when column shape changes to avoid transient empty-frame flicker.
		// This still protects bubbles/table from row/column length mismatches.
		if len(m.table.Rows()) > 0 {
			m.table.SetRows(nil)
		}
		m.table.SetColumns(columns)
		m.tableColumns = append(m.tableColumns[:0], columns...)
	}

	if columnsChanged || !equalTableRows(m.table.Rows(), rows) {
		m.table.SetRows(rows)
	}

	tableHeight := m.tableHeight()
	if m.table.Height() != tableHeight {
		m.table.SetHeight(tableHeight)
	}
	if m.table.Width() != tableWidth {
		m.table.SetWidth(tableWidth)
	}
	m.table.SetStyles(tableStyles())
	cursor := m.table.Cursor()
	if len(list.rows) == 0 {
		m.tableSetCursor(0)
	} else if cursor >= len(list.rows) {
		m.tableSetCursor(len(list.rows) - 1)
	}
	m.reconcileTableViewportState()
}

func (m Model) tableHeight() int {
	if m.height <= 0 {
		return defaultTableHeight
	}
	topLines := lineCount(m.renderTopSection())
	sectionSeparators := 1 // top section + main section
	debugLines := 0
	if m.debug {
		// Requests section: top/bottom border + title + fixed visible rows.
		debugLines = maxVisibleLogs + 3
		sectionSeparators++ // main section + debug section
	}
	// bubbles/table height controls only row viewport height; header + header border
	// plus the bordered main section and title consume extra terminal lines.
	available := m.height - topLines - mainSectionTitleLines - mainSectionBorderLines - debugLines - tableChromeLines - sectionSeparators
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
	case FocusGitHubTags:
		return "GHCR Tags"
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
	} else if m.githubActive || m.focus == FocusGitHubTags {
		spec.Tag = registry.TagTableSpec{
			ShowSize:       false,
			ShowPushed:     false,
			ShowLastPulled: false,
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
