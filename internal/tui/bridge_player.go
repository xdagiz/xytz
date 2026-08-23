package tui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/xdagiz/xytz/internal/player"
	"github.com/xdagiz/xytz/internal/types"
)

func playURL(pm *player.PlayerManager, program *tea.Program, url, ytdlFormat, backendPreference string, video types.VideoItem) tea.Cmd {
	return func() tea.Msg {
		res := pm.Start(url, ytdlFormat, backendPreference, video, func(v types.VideoItem, u string) {
			if program != nil {
				program.Send(types.PlayVideoMsg{SelectedVideo: v, IsPlayerExit: true, URL: u})
			}
		})
		if res.ErrMsg != "" {
			return types.PlayVideoMsg{ErrMsg: res.ErrMsg}
		}
		return types.PlayerStartedMsg{SelectedVideo: video}
	}
}
