package styles

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor   = lipgloss.Color("#ffffff")
	SecondaryColor = lipgloss.Color("#cdd6f4")
	ErrorColor     = lipgloss.Color("#f38ba8")
	SuccessColor   = lipgloss.Color("#a6e3a1")
	WarningColor   = lipgloss.Color("#f9e2af")
	InfoColor      = lipgloss.Color("#89dceb")
	MutedColor     = lipgloss.Color("#6c7086")
	MaroonColor    = lipgloss.Color("#eba0ac")
	PinkColor      = lipgloss.Color("#f5c2e7")
	MauveColor     = lipgloss.Color("#cba6f7")
)

var (
	ASCIIStyle         = lipgloss.NewStyle().Foreground(SecondaryColor).PaddingBottom(1)
	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(PrimaryColor)
	StatusBarStyle = lipgloss.NewStyle().Foreground(MutedColor)
	InputStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false).BorderForeground(SecondaryColor)
)
