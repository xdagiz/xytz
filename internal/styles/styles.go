package styles

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor   = lipgloss.Color("#ffffff")
	BlackColor     = lipgloss.Color("#1e1e2e")
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
	ASCIIStyle         = lipgloss.NewStyle().Foreground(MauveColor).PaddingBottom(1)
	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(SecondaryColor).
				Padding(1, 0)
	StatusBarStyle = lipgloss.NewStyle().Foreground(MutedColor).MarginTop(1)
	InputStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false).BorderForeground(MutedColor)

	listStyle              = lipgloss.NewStyle().Padding(0, 3)
	ListTitleStyle         = listStyle.Foreground(lipgloss.Color("#bac2de"))
	ListSelectedTitleStyle = listStyle.Foreground(MauveColor).Bold(true)
	ListDescStyle          = listStyle.Foreground(MutedColor)
	ListSelectedDescStyle  = listStyle.Foreground(lipgloss.Color("#cdd6f4"))

	ListContainer = lipgloss.NewStyle().PaddingBottom(1)

	SpinnerStyle = lipgloss.NewStyle().Foreground(PinkColor)

	ProgressContainer = lipgloss.NewStyle().PaddingBottom(1)

	SpeedStyle             = lipgloss.NewStyle().Foreground(MauveColor).Italic(true)
	TimeRemainingStyle     = lipgloss.NewStyle().Foreground(PinkColor).Italic(true)
	ProgressStyle          = lipgloss.NewStyle().Foreground(SecondaryColor)
	DestinationStyle       = lipgloss.NewStyle().Foreground(MutedColor)
	CompletionMessageStyle = lipgloss.NewStyle().Foreground(SuccessColor)
	HelpStyle              = lipgloss.NewStyle().Foreground(MutedColor).Faint(true)
	ErrorMessageStyle      = lipgloss.NewStyle().Foreground(ErrorColor)

	autocompleteStyle = lipgloss.NewStyle().PaddingLeft(1)
	AutocompleteItem  = autocompleteStyle.
				Foreground(SecondaryColor)
	AutocompleteSelected = autocompleteStyle.
				Foreground(MauveColor)

	TabActiveStyle   = lipgloss.NewStyle().Foreground(BlackColor).Background(MauveColor)
	TabInactiveStyle = lipgloss.NewStyle().Foreground(SecondaryColor)
)
