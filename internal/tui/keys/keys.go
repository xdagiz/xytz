package keys

import (
	"charm.land/bubbles/v2/key"
	"github.com/xdagiz/xytz/internal/types"
)

type KeyMap struct {
	CurrentState types.State

	HasError            bool
	IsPaused            bool
	IsCompleted         bool
	IsCancelled         bool
	IsQueue             bool
	HasQueueError       bool
	SelectedVideosCount int
	PauseSupported      bool

	Up   key.Binding
	Down key.Binding
	Prev key.Binding
	Next key.Binding

	Enter  key.Binding
	Back   key.Binding
	Cancel key.Binding

	CancelWithC key.Binding
	Quit        key.Binding
	QuitCtrlC   key.Binding
	Help        key.Binding
	TabNext     key.Binding
	TabPrev     key.Binding
	CopyURL     key.Binding

	OpenGitHub   key.Binding
	SearchUp     key.Binding
	SearchDown   key.Binding
	DeleteItem   key.Binding
	SaveForLater key.Binding

	PlayVideo    key.Binding
	Download     key.Binding
	SelectToggle key.Binding
	SelectAll    key.Binding
	DownloadAll  key.Binding
	GoToChannel  key.Binding

	FormatEnter key.Binding

	Pause  key.Binding
	Skip   key.Binding
	Retry  key.Binding
	DLUp   key.Binding
	DLDown key.Binding

	SpotifyDownload      key.Binding
	SpotifyDownloadEnter key.Binding

	PlaylistConfirm key.Binding
	PlaylistCancel  key.Binding
	ToggleFocus     key.Binding

	ACUp     key.Binding
	ACDown   key.Binding
	ACSelect key.Binding
}

func (k *KeyMap) ShortHelp() []key.Binding {
	switch k.CurrentState {
	case types.StateSearchInput:
		return []key.Binding{k.QuitCtrlC, k.OpenGitHub, k.SearchUp, k.SearchDown}

	case types.StateResumeList, types.StateLaterList:
		return []key.Binding{k.Enter, k.DeleteItem, k.Cancel}

	case types.StateLoading:
		return []key.Binding{k.QuitCtrlC, k.CancelWithC}

	case types.StateVideoList:
		if k.HasError {
			return []key.Binding{k.QuitCtrlC, k.Enter}
		}
		if k.SelectedVideosCount > 0 {
			return []key.Binding{k.QuitCtrlC, k.Download, k.Back}
		}
		return []key.Binding{
			k.QuitCtrlC, k.Back, k.PlayVideo, k.Download,
			k.SelectToggle, k.SelectAll, k.DownloadAll,
			k.GoToChannel, k.CopyURL, k.SaveForLater,
		}

	case types.StateFormatList:
		return []key.Binding{
			k.QuitCtrlC, k.Back, k.FormatEnter, k.TabNext, k.CopyURL, k.SaveForLater,
		}

	case types.StateChannelList:
		return []key.Binding{k.QuitCtrlC, k.Back, k.Enter}

	case types.StatePlaylistList:
		return []key.Binding{k.QuitCtrlC, k.Back, k.Enter}

	case types.StateDownload:
		if k.IsCompleted || k.IsCancelled {
			return []key.Binding{k.QuitCtrlC, k.Back, k.Enter}
		}
		dlKeys := []key.Binding{k.QuitCtrlC, k.Cancel, k.CopyURL}
		if k.PauseSupported {
			dlKeys = append([]key.Binding{dlKeys[0], k.Pause}, dlKeys[1:]...)
		}
		if k.HasQueueError {
			dlKeys = append(dlKeys, k.Skip, k.Retry)
		} else if k.IsQueue {
			dlKeys = append(dlKeys, k.Skip)
		}
		return dlKeys

	case types.StatePlaylistOpts:
		return []key.Binding{k.QuitCtrlC, k.Back, k.Up, k.Down, k.PlaylistConfirm}

	case types.StateVideoPlaying:
		return []key.Binding{k.QuitCtrlC, k.Back}

	case types.StateSpotifyTrack:
		return []key.Binding{k.QuitCtrlC, k.Back, k.SpotifyDownload}

	case types.StateSpotifyAlbumList:
		return []key.Binding{k.QuitCtrlC, k.Back, k.Download, k.SelectToggle, k.SelectAll, k.CopyURL}

	case types.StateSpotifyDownload:
		if k.IsCompleted || k.IsCancelled || k.HasError {
			return []key.Binding{k.QuitCtrlC, k.Back, k.SpotifyDownloadEnter}
		}
		sdKeys := []key.Binding{k.QuitCtrlC, k.Cancel}
		if k.PauseSupported {
			sdKeys = []key.Binding{sdKeys[0], k.Pause, sdKeys[1]}
		}
		if k.IsQueue {
			sdKeys = append(sdKeys, k.Skip)
		}
		return sdKeys

	default:
		return []key.Binding{k.QuitCtrlC}
	}
}

func (k *KeyMap) FullHelp() [][]key.Binding {
	sections := [][]key.Binding{
		{k.Up, k.Down, k.Prev, k.Next},
	}

	switch k.CurrentState {
	case types.StateVideoList:
		sections = append(sections, []key.Binding{
			k.PlayVideo, k.Download, k.SelectToggle, k.SelectAll,
			k.DownloadAll, k.GoToChannel, k.CopyURL, k.SaveForLater,
			k.Back, k.Quit,
		})

	case types.StateSpotifyAlbumList:
		sections = append(sections, []key.Binding{
			k.Download, k.SelectToggle, k.SelectAll, k.CopyURL,
			k.Back, k.Quit,
		})

	case types.StateFormatList:
		sections = append(sections, []key.Binding{
			k.FormatEnter, k.TabNext, k.CopyURL, k.SaveForLater,
			k.Back, k.Quit,
		})

	default:
		sections = append(sections, []key.Binding{k.Back, k.Cancel, k.Quit})
	}

	return sections
}

var Keys = &KeyMap{
	Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Prev: key.NewBinding(
		key.WithKeys("h", "k", "left", "shift+tab"),
		key.WithHelp("shift+tab/←", "prev tab"),
	),
	Next: key.NewBinding(
		key.WithKeys("l", "j", "right", "tab"),
		key.WithHelp("tab/→", "next tab"),
	),

	Enter:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "enter")),
	Back:        key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", "back")),
	Cancel:      key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	CancelWithC: key.NewBinding(key.WithKeys("esc", "c"), key.WithHelp("esc/c", "cancel")),
	Quit:        key.NewBinding(key.WithKeys("q", "esc", "ctrl+c"), key.WithHelp("q", "quit")),
	QuitCtrlC:   key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
	Help:        key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	TabNext:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	TabPrev:     key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
	CopyURL:     key.NewBinding(key.WithKeys("ctrl+y"), key.WithHelp("ctrl+y", "copy url")),

	OpenGitHub:   key.NewBinding(key.WithKeys("ctrl+o"), key.WithHelp("ctrl+o", "open github")),
	SearchUp:     key.NewBinding(key.WithKeys("up", "ctrl+p"), key.WithHelp("↑/ctrl+p", "history up")),
	SearchDown:   key.NewBinding(key.WithKeys("down", "ctrl+n"), key.WithHelp("↓/ctrl+n", "history down")),
	DeleteItem:   key.NewBinding(key.WithKeys("delete", "ctrl+d"), key.WithHelp("delete", "delete item")),
	SaveForLater: key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save for later")),

	PlayVideo:    key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "play")),
	Download:     key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "download")),
	SelectToggle: key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle selection")),
	SelectAll:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
	DownloadAll:  key.NewBinding(key.WithKeys("ctrl+d"), key.WithHelp("ctrl+d", "download all")),
	GoToChannel:  key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "go to channel")),

	FormatEnter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),

	Pause:  key.NewBinding(key.WithKeys("p", "space"), key.WithHelp("p/space", "pause/resume")),
	Skip:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "skip")),
	Retry:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "retry")),
	DLUp:   key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "prev item")),
	DLDown: key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "next item")),

	SpotifyDownload:      key.NewBinding(key.WithKeys("enter", "d"), key.WithHelp("d/enter", "download")),
	SpotifyDownloadEnter: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),

	PlaylistConfirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	PlaylistCancel:  key.NewBinding(key.WithKeys("esc", "b"), key.WithHelp("esc/b", "back")),
	ToggleFocus:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch focus")),

	ACUp:     key.NewBinding(key.WithKeys("ctrl+p", "up")),
	ACDown:   key.NewBinding(key.WithKeys("ctrl+n", "down")),
	ACSelect: key.NewBinding(key.WithKeys("enter", "tab")),
}
