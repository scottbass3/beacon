package tui

import lipgloss "charm.land/lipgloss/v2"

// Raw ANSI color codes shared with table_columns.go (which uses lipgloss v1 for bubbles).
const (
	rawColorPrimary   = "75"
	rawColorAccent    = "208"
	rawColorMuted     = "241"
	rawColorSelected  = "16"
	rawColorBorder    = "238"
	rawColorSurface   = "235"
	rawColorSurface2  = "232"
	rawColorTitleText = "255"
	rawColorSuccess   = "42"
	rawColorDanger    = "196"
)

var (
	colorPrimary   = lipgloss.Color(rawColorPrimary)
	colorAccent    = lipgloss.Color(rawColorAccent)
	colorMuted     = lipgloss.Color(rawColorMuted)
	colorBorder    = lipgloss.Color(rawColorBorder)
	colorSurface   = lipgloss.Color(rawColorSurface)
	colorSurface2  = lipgloss.Color(rawColorSurface2)
	colorTitleText = lipgloss.Color(rawColorTitleText)
	colorSuccess   = lipgloss.Color(rawColorSuccess)
	colorDanger    = lipgloss.Color(rawColorDanger)
)

var (
	titleStyle          = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	statusStyle         = lipgloss.NewStyle().Foreground(colorMuted)
	statusLoadingStyle  = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	metaLabelStyle      = lipgloss.NewStyle().Foreground(colorMuted).MarginRight(1)
	metaValueStyle      = lipgloss.NewStyle().Foreground(colorTitleText).MarginRight(3)
	metaSeparatorStyle  = lipgloss.NewStyle().Foreground(colorBorder)
	modeInputStyle      = lipgloss.NewStyle().Foreground(colorAccent).Padding(0, 1)
	shortcutKeyStyle    = lipgloss.NewStyle().Foreground(colorAccent)
	shortcutLabelStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	shortcutSepStyle    = lipgloss.NewStyle().Foreground(colorBorder)
	shortcutPrefixStyle = lipgloss.NewStyle().Foreground(colorMuted)
	helpHeadingStyle    = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	helpItemStyle       = lipgloss.NewStyle().Foreground(colorTitleText)
	helpFooterStyle     = lipgloss.NewStyle().Foreground(colorMuted)
	emptyStyle          = lipgloss.NewStyle().Foreground(colorMuted).Italic(true)

	mainSectionStyle      = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)
	mainSectionTitleStyle = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	mainSectionDivStyle   = lipgloss.NewStyle().Foreground(colorBorder)
	topSectionStyle       = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Padding(0, 1)
	logTitleStyle         = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	logBoxStyle           = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Background(colorSurface).Padding(0, 1)

	modalPanelStyle        = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorBorder).Background(colorSurface).Padding(1, 2)
	modalTitleStyle        = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	modalLabelStyle        = lipgloss.NewStyle().Foreground(colorMuted)
	modalErrorStyle        = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	modalInputStyle        = lipgloss.NewStyle().Background(colorSurface2).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorMuted).BorderBackground(colorSurface).Padding(0, 1)
	modalInputFocusStyle   = lipgloss.NewStyle().Background(colorSurface2).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorAccent).BorderBackground(colorSurface).Padding(0, 1)
	modalFocusStyle        = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
	modalButtonStyle       = lipgloss.NewStyle().Foreground(colorMuted).Background(colorSurface2).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorMuted).BorderBackground(colorSurface).Padding(0, 1)
	modalButtonFocusStyle  = lipgloss.NewStyle().Foreground(colorSurface2).Background(colorAccent).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorAccent).BorderBackground(colorSurface).Bold(true).Padding(0, 1)
	modalDangerButtonStyle = lipgloss.NewStyle().Foreground(colorDanger).Background(colorSurface2).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorDanger).BorderBackground(colorSurface).Padding(0, 1)
	modalDangerFocusStyle  = lipgloss.NewStyle().Foreground(colorSurface2).Background(colorDanger).BorderStyle(lipgloss.RoundedBorder()).BorderForeground(colorDanger).BorderBackground(colorSurface).Bold(true).Padding(0, 1)
	modalOptionStyle       = lipgloss.NewStyle().Foreground(colorTitleText).Background(colorSurface2).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorMuted).BorderBackground(colorSurface).Padding(0, 1)
	modalOptionFocusStyle  = lipgloss.NewStyle().Foreground(colorSurface2).Background(colorAccent).BorderStyle(lipgloss.NormalBorder()).BorderForeground(colorAccent).BorderBackground(colorSurface).Bold(true).Padding(0, 1)
	modalOptionMutedStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	modalOptionErrorStyle  = lipgloss.NewStyle().Foreground(colorDanger).Faint(true)
	modalHelpStyle         = lipgloss.NewStyle().Foreground(colorMuted)
	modalDividerStyle      = lipgloss.NewStyle().Foreground(colorBorder)
)
