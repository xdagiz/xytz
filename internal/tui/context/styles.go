package context

import (
	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/tui/theme"
)

type Styles struct {
	PrimaryText   lipgloss.Style
	MutedText     lipgloss.Style
	ErrorText     lipgloss.Style
	SuccessText   lipgloss.Style
	WarningText   lipgloss.Style
	StatusBarText lipgloss.Style
}

func InitStyles(th theme.Theme) Styles {
	return Styles{
		PrimaryText:   lipgloss.NewStyle().Foreground(lipgloss.Color(th.TextPrimary)),
		MutedText:     lipgloss.NewStyle().Foreground(lipgloss.Color(th.TextMuted)),
		ErrorText:     lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusError)),
		SuccessText:   lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusSuccess)),
		WarningText:   lipgloss.NewStyle().Foreground(lipgloss.Color(th.StatusWarning)),
		StatusBarText: lipgloss.NewStyle().Foreground(lipgloss.Color(th.TextMuted)),
	}
}
