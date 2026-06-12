package models

import (
	"charm.land/bubbles/v2/key"
)

type globalKeys struct {
	TabNext key.Binding
	TabPrev key.Binding
	CopyURL key.Binding
}

var GlobalModelKeys = globalKeys{
	TabNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	TabPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	),
	CopyURL: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy url"),
	),
}

type searchKeys struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Help       key.Binding
	Quit       key.Binding
	DeleteItem key.Binding
	OpenGitHub key.Binding
}

var SearchModelKeys = searchKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "ctrl+p"),
		key.WithHelp("↑/ctrl+p", "history up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "ctrl+n"),
		key.WithHelp("↓/ctrl+n", "history down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "search"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	DeleteItem: key.NewBinding(
		key.WithKeys("delete", "ctrl+d"),
		key.WithHelp("delete", "delete item"),
	),
	OpenGitHub: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open github"),
	),
}

func (k searchKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

type videoListKeys struct {
	Enter        key.Binding
	Space        key.Binding
	Download     key.Binding
	DownloadAll  key.Binding
	Play         key.Binding
	SelectAll    key.Binding
	GoToChannel  key.Binding
	SaveForLater key.Binding
	Quit         key.Binding
}

var VideoListModelKeys = videoListKeys{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Space: key.NewBinding(
		key.WithKeys("space"),
		key.WithHelp("space", "toggle selection"),
	),
	Download: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "download"),
	),
	DownloadAll: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "download all"),
	),
	Play: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "play"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	),
	GoToChannel: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "go to channel"),
	),
	SaveForLater: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save for later"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k videoListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

type formatListKeys struct {
	Enter        key.Binding
	Quit         key.Binding
	SaveForLater key.Binding
}

var FormatListModelKeys = formatListKeys{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	SaveForLater: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "save for later"),
	),
}

func (k formatListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

type downloadKeys struct {
	Pause  key.Binding
	Cancel key.Binding
	Enter  key.Binding
	Skip   key.Binding
	Retry  key.Binding
	Up     key.Binding
	Down   key.Binding
}

var DownloadModelKeys = downloadKeys{
	Pause: key.NewBinding(
		key.WithKeys("p", "space"),
		key.WithHelp("p/space", "pause/resume"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("c", "esc"),
		key.WithHelp("c/esc", "cancel"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "continue"),
	),
	Skip: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "skip"),
	),
	Retry: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "retry"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "prev item"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "next item"),
	),
}

func (k downloadKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Pause, k.Cancel}
}

type channelListKeys struct {
	Enter key.Binding
	Quit  key.Binding
}

var ChannelListModelKeys = channelListKeys{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k channelListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

type playlistListKeys struct {
	Enter key.Binding
	Quit  key.Binding
}

var PlaylistListModelKeys = playlistListKeys{
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k playlistListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

type PlayerKeys struct {
	Quit key.Binding
}

var PlayerModelKeys = PlayerKeys{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

func (k PlayerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}
