package styles

import (
	"image/color"

	"charm.land/bubbles/v2/list"
	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/tui/theme"
)

var (
	BackgroundBaseColor  color.Color
	TextPrimaryColor     color.Color
	StatusErrorColor     color.Color
	StatusSuccessColor   color.Color
	StatusWarningColor   color.Color
	StatusInfoColor      color.Color
	TextMutedColor       color.Color
	AccentSecondaryColor color.Color
	AccentPrimaryColor   color.Color

	ASCIIStyle lipgloss.Style

	SectionHeaderStyle lipgloss.Style
	StatusBarStyle     lipgloss.Style
	InputStyle         lipgloss.Style
	MutedStyle         lipgloss.Style

	listStyle              lipgloss.Style
	ListTitleStyle         lipgloss.Style
	ListSelectedTitleStyle lipgloss.Style
	ListDescStyle          lipgloss.Style
	ListSelectedDescStyle  lipgloss.Style
	ListDimmedTitle        lipgloss.Style
	ListDimmedDesc         lipgloss.Style

	ListSelectedQueueStyle lipgloss.Style
	QueueSelectedItemStyle lipgloss.Style
	ListContainer          lipgloss.Style
	SpinnerStyle           lipgloss.Style
	ProgressContainer      lipgloss.Style

	SpeedStyle             lipgloss.Style
	TimeRemainingStyle     lipgloss.Style
	ProgressStyle          lipgloss.Style
	DestinationStyle       lipgloss.Style
	CompletionMessageStyle lipgloss.Style
	HelpStyle              lipgloss.Style
	ErrorMessageStyle      lipgloss.Style
	WarningMessageStyle    lipgloss.Style

	autocompleteStyle    lipgloss.Style
	AutocompleteItem     lipgloss.Style
	AutocompleteSelected lipgloss.Style

	sortStyle lipgloss.Style
	SortTitle lipgloss.Style
	SortHelp  lipgloss.Style
	SortItem  lipgloss.Style

	TabActiveStyle   lipgloss.Style
	TabInactiveStyle lipgloss.Style

	FormatContainerStyle       lipgloss.Style
	CustomFormatContainerStyle lipgloss.Style
	FormatTabHelpStyle         lipgloss.Style
	FormatCustomInputStyle     lipgloss.Style
	FormatCustomInputPrompt    lipgloss.Style
	FormatCustomHelpStyle      lipgloss.Style
)

func init() {
	ApplyTheme(theme.CatppuccinMochaTheme())
}

func ApplyTheme(t theme.Theme) {
	TextPrimaryColor = lipgloss.Color(t.TextSecondary)
	BackgroundBaseColor = lipgloss.Color(t.BackgroundBase)
	StatusErrorColor = lipgloss.Color(t.StatusError)
	StatusSuccessColor = lipgloss.Color(t.StatusSuccess)
	StatusWarningColor = lipgloss.Color(t.StatusWarning)
	StatusInfoColor = lipgloss.Color(t.StatusInfo)
	TextMutedColor = lipgloss.Color(t.TextMuted)
	AccentSecondaryColor = lipgloss.Color(t.AccentSecondary)
	AccentPrimaryColor = lipgloss.Color(t.AccentPrimary)

	rebuildStyles()
}

func rebuildStyles() {
	ASCIIStyle = lipgloss.NewStyle().Foreground(AccentPrimaryColor).PaddingBottom(1)
	SectionHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(TextPrimaryColor).
		Padding(1, 0)
	StatusBarStyle = lipgloss.NewStyle().Foreground(TextMutedColor).Padding(1, 2)
	InputStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false).BorderForeground(TextMutedColor)
	MutedStyle = lipgloss.NewStyle().Foreground(TextMutedColor)

	listStyle = lipgloss.NewStyle().Padding(0, 3)
	ListTitleStyle = listStyle.Foreground(TextPrimaryColor)
	ListSelectedTitleStyle = listStyle.Foreground(AccentPrimaryColor).Bold(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(AccentPrimaryColor).
		Padding(0, 0, 0, 2)
	ListDescStyle = listStyle.Foreground(TextMutedColor)
	ListSelectedDescStyle = listStyle.Foreground(TextPrimaryColor)
	ListDimmedTitle = listStyle.Foreground(TextMutedColor).Padding(0, 0, 0, 3)
	ListDimmedDesc = listStyle.Foreground(TextMutedColor)

	ListSelectedQueueStyle = lipgloss.NewStyle().Foreground(AccentSecondaryColor).Bold(true)
	QueueSelectedItemStyle = lipgloss.NewStyle().Foreground(AccentPrimaryColor).Bold(true)
	ListContainer = lipgloss.NewStyle().PaddingBottom(1)
	SpinnerStyle = lipgloss.NewStyle().Foreground(AccentSecondaryColor)
	ProgressContainer = lipgloss.NewStyle().PaddingBottom(1)

	SpeedStyle = lipgloss.NewStyle().Foreground(StatusSuccessColor).Italic(true)
	TimeRemainingStyle = lipgloss.NewStyle().Foreground(StatusSuccessColor).Italic(true)
	ProgressStyle = lipgloss.NewStyle().Foreground(TextPrimaryColor)
	DestinationStyle = lipgloss.NewStyle().Foreground(TextMutedColor)
	CompletionMessageStyle = lipgloss.NewStyle().Foreground(StatusSuccessColor)
	HelpStyle = lipgloss.NewStyle().Foreground(TextMutedColor).Faint(true)
	ErrorMessageStyle = lipgloss.NewStyle().Foreground(StatusErrorColor)
	WarningMessageStyle = lipgloss.NewStyle().Foreground(StatusWarningColor)

	autocompleteStyle = lipgloss.NewStyle().PaddingLeft(1)
	AutocompleteItem = autocompleteStyle.Foreground(TextPrimaryColor)
	AutocompleteSelected = autocompleteStyle.Foreground(AccentPrimaryColor)

	sortStyle = lipgloss.NewStyle().PaddingLeft(1)
	SortTitle = sortStyle.Foreground(TextPrimaryColor).PaddingTop(1).Bold(true)
	SortHelp = sortStyle.Foreground(TextMutedColor).Italic(true)
	SortItem = sortStyle.Foreground(AccentPrimaryColor).PaddingLeft(1).Italic(true)

	TabActiveStyle = lipgloss.NewStyle().Foreground(BackgroundBaseColor).Background(AccentPrimaryColor)
	TabInactiveStyle = lipgloss.NewStyle().Foreground(TextPrimaryColor)

	FormatContainerStyle = lipgloss.NewStyle().PaddingLeft(1)
	CustomFormatContainerStyle = FormatContainerStyle.PaddingLeft(3)
	FormatTabHelpStyle = lipgloss.NewStyle().Foreground(TextMutedColor)
	FormatCustomInputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false).
		BorderForeground(TextMutedColor).
		MarginTop(1)
	FormatCustomInputPrompt = lipgloss.NewStyle().Foreground(AccentSecondaryColor)
	FormatCustomHelpStyle = lipgloss.NewStyle().Foreground(TextMutedColor).PaddingTop(1)
}

func NewListDelegate() list.DefaultDelegate {
	dl := list.NewDefaultDelegate()
	dl.Styles.NormalTitle = ListTitleStyle
	dl.Styles.SelectedTitle = ListSelectedTitleStyle
	dl.Styles.NormalDesc = ListDescStyle
	dl.Styles.SelectedDesc = ListSelectedDescStyle
	dl.Styles.DimmedTitle = ListDimmedTitle
	dl.Styles.DimmedDesc = ListDimmedDesc

	return dl
}
