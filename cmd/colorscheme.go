package cmd

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/fang"
)

func fangColorScheme(_ lipgloss.LightDarkFunc) fang.ColorScheme {
	return fang.ColorScheme{
		Base:           lipgloss.Color("#cdd6f4"),
		Title:          lipgloss.Color("#cba6f7"),
		Description:    lipgloss.Color("#a6adc8"),
		Codeblock:      lipgloss.Color("#313244"),
		Program:        lipgloss.Color("#89b4fa"),
		DimmedArgument: lipgloss.Color("#6c7086"),
		Comment:        lipgloss.Color("#6c7086"),
		Flag:           lipgloss.Color("#f5c2e7"),
		FlagDefault:    lipgloss.Color("#585b70"),
		Command:        lipgloss.Color("#a6e3a1"),
		QuotedString:   lipgloss.Color("#f9e2af"),
		Argument:       lipgloss.Color("#bac2de"),
		Help:           lipgloss.Color("#6c7086"),
		Dash:           lipgloss.Color("#585b70"),
		ErrorHeader: [2]color.Color{
			lipgloss.Color("#1e1e2e"),
			lipgloss.Color("#f38ba8"),
		},
		ErrorDetails: lipgloss.Color("#f38ba8"),
	}
}
