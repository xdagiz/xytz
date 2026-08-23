package styles

import (
	"fmt"
	"image/color"
	"io"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/xdagiz/xytz/internal/theme"
)

type Styles struct {
	BackgroundBaseColor  color.Color
	TextPrimaryColor     color.Color
	StatusErrorColor     color.Color
	StatusSuccessColor   color.Color
	StatusWarningColor   color.Color
	StatusInfoColor      color.Color
	TextMutedColor       color.Color
	TextSubtleColor      color.Color
	AccentSecondaryColor color.Color
	AccentPrimaryColor   color.Color

	AccentPrimaryStyle lipgloss.Style
	ASCIIStyle         lipgloss.Style

	SectionHeaderStyle lipgloss.Style
	StatusBarStyle     lipgloss.Style
	InputStyle         lipgloss.Style
	MutedStyle         lipgloss.Style

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
	HelpKeyStyle           lipgloss.Style
	ErrorMessageStyle      lipgloss.Style
	WarningMessageStyle    lipgloss.Style

	AutocompleteItem     lipgloss.Style
	AutocompleteSelected lipgloss.Style

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

	VerifiedBadgeStyle lipgloss.Style
}

func New(t theme.Theme) Styles {
	s := Styles{
		TextPrimaryColor:     lipgloss.Color(t.TextSecondary),
		BackgroundBaseColor:  lipgloss.Color(t.BackgroundBase),
		StatusErrorColor:     lipgloss.Color(t.StatusError),
		StatusSuccessColor:   lipgloss.Color(t.StatusSuccess),
		StatusWarningColor:   lipgloss.Color(t.StatusWarning),
		StatusInfoColor:      lipgloss.Color(t.StatusInfo),
		TextMutedColor:       lipgloss.Color(t.TextMuted),
		TextSubtleColor:      lipgloss.Color(t.TextSubtle),
		AccentSecondaryColor: lipgloss.Color(t.AccentSecondary),
		AccentPrimaryColor:   lipgloss.Color(t.AccentPrimary),
	}

	listPad := lipgloss.NewStyle().Padding(0, 3)

	s.ASCIIStyle = lipgloss.NewStyle().Foreground(s.AccentPrimaryColor).PaddingBottom(1)
	s.AccentPrimaryStyle = lipgloss.NewStyle().Foreground(s.AccentPrimaryColor)
	s.SectionHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(s.TextPrimaryColor).
		Padding(1, 0)
	s.StatusBarStyle = lipgloss.NewStyle().Foreground(s.TextMutedColor).Padding(0, 2)
	s.InputStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false).BorderForeground(s.TextMutedColor)
	s.MutedStyle = lipgloss.NewStyle().Foreground(s.TextMutedColor)

	s.ListTitleStyle = listPad.Foreground(s.TextPrimaryColor)
	s.ListSelectedTitleStyle = listPad.Foreground(s.AccentPrimaryColor).Bold(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(s.AccentPrimaryColor).
		Padding(0, 0, 0, 2)
	s.ListDescStyle = listPad.Foreground(s.TextMutedColor)
	s.ListSelectedDescStyle = listPad.Foreground(s.TextPrimaryColor)
	s.ListDimmedTitle = listPad.Foreground(s.TextMutedColor).Padding(0, 0, 0, 3)
	s.ListDimmedDesc = listPad.Foreground(s.TextMutedColor)

	s.ListSelectedQueueStyle = lipgloss.NewStyle().Foreground(s.AccentSecondaryColor).Bold(true)
	s.QueueSelectedItemStyle = lipgloss.NewStyle().Foreground(s.AccentPrimaryColor).Bold(true)
	s.ListContainer = lipgloss.NewStyle().PaddingBottom(1)
	s.SpinnerStyle = lipgloss.NewStyle().Foreground(s.AccentSecondaryColor)
	s.ProgressContainer = lipgloss.NewStyle().PaddingBottom(1)

	s.SpeedStyle = lipgloss.NewStyle().Foreground(s.StatusSuccessColor).Italic(true)
	s.TimeRemainingStyle = lipgloss.NewStyle().Foreground(s.StatusSuccessColor).Italic(true)
	s.ProgressStyle = lipgloss.NewStyle().Foreground(s.TextPrimaryColor)
	s.DestinationStyle = lipgloss.NewStyle().Foreground(s.TextMutedColor)
	s.CompletionMessageStyle = lipgloss.NewStyle().Foreground(s.StatusSuccessColor)
	s.HelpStyle = lipgloss.NewStyle().Foreground(s.TextMutedColor).Faint(true)
	s.HelpKeyStyle = lipgloss.NewStyle().Foreground(s.TextSubtleColor)
	s.ErrorMessageStyle = lipgloss.NewStyle().Foreground(s.StatusErrorColor)
	s.WarningMessageStyle = lipgloss.NewStyle().Foreground(s.StatusWarningColor)

	s.VerifiedBadgeStyle = lipgloss.NewStyle().Foreground(s.StatusInfoColor)

	ac := lipgloss.NewStyle().PaddingLeft(1)
	s.AutocompleteItem = ac.Foreground(s.TextPrimaryColor)
	s.AutocompleteSelected = ac.Foreground(s.AccentPrimaryColor)

	sortPad := lipgloss.NewStyle().PaddingLeft(1)
	s.SortTitle = sortPad.Foreground(s.TextPrimaryColor).PaddingTop(1).Bold(true)
	s.SortHelp = sortPad.Foreground(s.TextMutedColor).Italic(true)
	s.SortItem = sortPad.Foreground(s.AccentPrimaryColor).PaddingLeft(1).Italic(true)

	s.TabActiveStyle = lipgloss.NewStyle().Foreground(s.BackgroundBaseColor).Background(s.AccentPrimaryColor)
	s.TabInactiveStyle = lipgloss.NewStyle().Foreground(s.TextPrimaryColor)

	s.FormatContainerStyle = lipgloss.NewStyle().PaddingLeft(1)
	s.CustomFormatContainerStyle = s.FormatContainerStyle.PaddingLeft(3)
	s.FormatTabHelpStyle = lipgloss.NewStyle().Foreground(s.TextMutedColor)
	s.FormatCustomInputStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false).
		BorderForeground(s.TextMutedColor).
		MarginTop(1)
	s.FormatCustomInputPrompt = lipgloss.NewStyle().Foreground(s.AccentSecondaryColor)
	s.FormatCustomHelpStyle = lipgloss.NewStyle().Foreground(s.TextMutedColor).PaddingTop(1)

	return s
}

type compactDelegate struct {
	list.DefaultDelegate
	muted color.Color
}

func (d compactDelegate) Height() int {
	return 1
}

func (d compactDelegate) Spacing() int {
	return 1
}

func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	title := item.FilterValue()
	desc := ""

	if di, ok := item.(interface {
		Title() string
		Description() string
	}); ok {
		title = di.Title()
		desc = di.Description()
	}

	isSelected := index == m.Index()

	if isSelected {
		fmt.Fprint(w, d.Styles.SelectedTitle.Render(title))
	} else {
		fmt.Fprint(w, d.Styles.NormalTitle.Render(title))
	}

	if desc != "" {
		mutedStyle := lipgloss.NewStyle().Foreground(d.muted)
		fmt.Fprint(w, mutedStyle.Render(" • "))
		fmt.Fprint(w, mutedStyle.Render(desc))
	}
}

type ClickableDelegate struct {
	inner  list.ItemDelegate
	prefix string
}

func NewClickableDelegate(prefix string, inner list.ItemDelegate) *ClickableDelegate {
	return &ClickableDelegate{inner: inner, prefix: prefix}
}

func (d *ClickableDelegate) Height() int {
	return d.inner.Height()
}

func (d *ClickableDelegate) Spacing() int {
	return d.inner.Spacing()
}

func (d *ClickableDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return d.inner.Update(msg, m)
}

func (d *ClickableDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var buf strings.Builder
	d.inner.Render(&buf, m, index, item)
	rendered := zone.Mark(d.prefix+strconv.Itoa(index), buf.String())
	fmt.Fprint(w, rendered)
}

func (s Styles) NewListDelegate() list.DefaultDelegate {
	dl := list.NewDefaultDelegate()
	dl.Styles.NormalTitle = s.ListTitleStyle
	dl.Styles.SelectedTitle = s.ListSelectedTitleStyle
	dl.Styles.NormalDesc = s.ListDescStyle
	dl.Styles.SelectedDesc = s.ListSelectedDescStyle
	dl.Styles.DimmedTitle = s.ListDimmedTitle
	dl.Styles.DimmedDesc = s.ListDimmedDesc
	return dl
}

func (s Styles) NewCompactDelegate() compactDelegate {
	d := compactDelegate{
		DefaultDelegate: list.NewDefaultDelegate(),
		muted:           s.TextMutedColor,
	}
	d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(s.TextPrimaryColor).Padding(0, 0, 0, 3)
	d.Styles.SelectedTitle = s.ListSelectedTitleStyle
	d.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(s.TextMutedColor)
	d.Styles.DimmedDesc = lipgloss.NewStyle().Foreground(s.TextMutedColor)
	return d
}
