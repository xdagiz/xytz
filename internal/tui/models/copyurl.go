package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/utils"
)

func CopyURLCmd(url string) tea.Cmd {
	if strings.TrimSpace(url) == "" {
		return nil
	}

	return func() tea.Msg {
		if err := utils.CopyToClipboard(url); err != nil {
			return types.ShowToastMsg{Message: "Failed to copy url"}
		}

		return types.ShowToastMsg{Message: "url copied to clipboard"}
	}
}
