package tui

import tea "github.com/charmbracelet/bubbletea"

type tableMouseRegion struct {
	x      int
	y      int
	width  int
	height int
}

func (m Model) tableRenderedBoundsForCursor(cursor int) (int, int) {
	rowCount := len(m.table.Rows())
	if rowCount == 0 {
		return 0, 0
	}
	height := maxInt(1, m.table.Height())
	start := clampInt(cursor-height, 0, cursor)
	end := clampInt(cursor+height, cursor, rowCount)
	return start, end
}

func (m Model) tableRenderedRowCountForCursor(cursor int) int {
	start, end := m.tableRenderedBoundsForCursor(cursor)
	return end - start
}

func (m Model) tableSetYOffsetClamped(offset, cursor int) int {
	renderedCount := m.tableRenderedRowCountForCursor(cursor)
	if renderedCount == 0 {
		return 0
	}
	height := maxInt(1, m.table.Height())
	maxYOffset := maxInt(0, renderedCount-height)
	return clampInt(offset, 0, maxYOffset)
}

func (m Model) tableClampYOffsetAfterSetContent(offset, cursor int) int {
	if offset < 0 {
		offset = 0
	}
	renderedCount := m.tableRenderedRowCountForCursor(cursor)
	if renderedCount == 0 {
		return 0
	}
	if offset > renderedCount-1 {
		height := maxInt(1, m.table.Height())
		return maxInt(0, renderedCount-height)
	}
	return offset
}

func (m *Model) tableSetCursor(cursor int) {
	m.table.SetCursor(cursor)
	m.tableYOffset = m.tableClampYOffsetAfterSetContent(m.tableYOffset, m.table.Cursor())
}

func (m *Model) tableMoveUp(n int) {
	if n <= 0 || len(m.table.Rows()) == 0 {
		return
	}

	height := maxInt(1, m.table.Height())
	nextCursor := clampInt(m.table.Cursor()-n, 0, len(m.table.Rows())-1)
	start, _ := m.tableRenderedBoundsForCursor(nextCursor)
	nextYOffset := m.tableYOffset

	switch {
	case start == 0:
		nextYOffset = clampInt(nextYOffset, 0, nextCursor)
		nextYOffset = m.tableSetYOffsetClamped(nextYOffset, nextCursor)
	case start < height:
		nextYOffset = clampInt(nextYOffset+n, 0, nextCursor)
		nextYOffset = m.tableSetYOffsetClamped(nextYOffset, nextCursor)
	case nextYOffset >= 1:
		nextYOffset = clampInt(nextYOffset+n, 1, height)
	}

	m.table.MoveUp(n)
	m.tableYOffset = m.tableClampYOffsetAfterSetContent(nextYOffset, m.table.Cursor())
}

func (m *Model) tableMoveDown(n int) {
	if n <= 0 || len(m.table.Rows()) == 0 {
		return
	}

	rowCount := len(m.table.Rows())
	height := maxInt(1, m.table.Height())
	nextCursor := clampInt(m.table.Cursor()+n, 0, rowCount-1)
	nextYOffset := m.tableClampYOffsetAfterSetContent(m.tableYOffset, nextCursor)
	start, end := m.tableRenderedBoundsForCursor(nextCursor)

	switch {
	case end == rowCount:
		nextYOffset = clampInt(nextYOffset-n, 1, height)
		nextYOffset = m.tableSetYOffsetClamped(nextYOffset, nextCursor)
	case nextCursor > (end-start)/2:
		nextYOffset = clampInt(nextYOffset-n, 1, nextCursor)
		nextYOffset = m.tableSetYOffsetClamped(nextYOffset, nextCursor)
	case nextYOffset > 1:
	case nextCursor > nextYOffset+height-1:
		nextYOffset = clampInt(nextYOffset+1, 0, 1)
	}

	m.table.MoveDown(n)
	m.tableYOffset = m.tableClampYOffsetAfterSetContent(nextYOffset, m.table.Cursor())
}

func (m *Model) tableGotoTop() {
	m.tableMoveUp(m.table.Cursor())
}

func (m *Model) tableGotoBottom() {
	m.tableMoveDown(len(m.table.Rows()))
}

func (m *Model) reconcileTableViewportState() {
	m.tableYOffset = m.tableClampYOffsetAfterSetContent(m.tableYOffset, m.table.Cursor())
}

func (m Model) tableFirstVisibleRow() int {
	rowCount := len(m.table.Rows())
	if rowCount == 0 {
		return 0
	}
	cursor := clampInt(m.table.Cursor(), 0, rowCount-1)
	start, _ := m.tableRenderedBoundsForCursor(cursor)
	return clampInt(start+m.tableYOffset, 0, rowCount-1)
}

func (m Model) tableMouseRowsRegion() (tableMouseRegion, bool) {
	width := m.table.Width()
	height := m.table.Height()
	if width <= 0 || height <= 0 {
		return tableMouseRegion{}, false
	}
	topLines := lineCount(m.renderTopSection())
	// main section layout:
	// [top section]
	// <blank separator line>
	// main border top
	// title line
	// table header
	// table rows...
	rowsY := topLines + 1 + 1 + mainSectionTitleLines + 1
	// main section has a left border and horizontal padding of 1.
	contentX := 2
	return tableMouseRegion{
		x:      contentX,
		y:      rowsY,
		width:  width,
		height: height,
	}, true
}

func (m Model) tableRowAtMouse(msg tea.MouseMsg) (int, bool) {
	region, ok := m.tableMouseRowsRegion()
	if !ok || len(m.table.Rows()) == 0 {
		return 0, false
	}
	if msg.X < region.x || msg.X >= region.x+region.width {
		return 0, false
	}
	if msg.Y < region.y || msg.Y >= region.y+region.height {
		return 0, false
	}
	row := m.tableFirstVisibleRow() + (msg.Y - region.y)
	if row < 0 || row >= len(m.table.Rows()) {
		return 0, false
	}
	return row, true
}
