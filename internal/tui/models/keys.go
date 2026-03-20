package models

import (
	"charm.land/bubbles/v2/key"
)

type SearchKeys struct {
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Help       key.Binding
	Quit       key.Binding
	CopyURL    key.Binding
	DeleteItem key.Binding
	OpenGitHub key.Binding
}

func (k SearchKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k SearchKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Help, k.Quit, k.CopyURL},
	}
}

type VideoListKeys struct {
	Up          key.Binding
	Down        key.Binding
	Enter       key.Binding
	Space       key.Binding
	Download    key.Binding
	DownloadAll key.Binding
	Play        key.Binding
	CopyURL     key.Binding
	SelectAll   key.Binding
	GoToChannel key.Binding
	Quit        key.Binding
}

func (k VideoListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k VideoListKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Space},
		{k.Download, k.DownloadAll, k.Play, k.CopyURL},
		{k.SelectAll, k.GoToChannel, k.Quit},
	}
}

type FormatListKeys struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	TabNext key.Binding
	TabPrev key.Binding
	CopyURL key.Binding
	Quit    key.Binding
}

func (k FormatListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k FormatListKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.TabNext, k.TabPrev},
		{k.CopyURL, k.Quit},
	}
}

type DownloadKeys struct {
	Pause   key.Binding
	Cancel  key.Binding
	CopyURL key.Binding
	Enter   key.Binding
	Skip    key.Binding
	Retry   key.Binding
	Up      key.Binding
	Down    key.Binding
}

func (k DownloadKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Pause, k.Cancel}
}

func (k DownloadKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Pause, k.Cancel, k.CopyURL},
		{k.Skip, k.Retry, k.Up, k.Down},
	}
}

type ChannelListKeys struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
}

func (k ChannelListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k ChannelListKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Quit},
	}
}

type PlaylistListKeys struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Quit  key.Binding
}

func (k PlaylistListKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k PlaylistListKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter, k.Quit},
	}
}

type PlayerKeys struct {
	Quit key.Binding
}

func (k PlayerKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k PlayerKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit},
	}
}

var SearchModelKeys = SearchKeys{
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
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	CopyURL: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy url"),
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

var VideoListModelKeys = VideoListKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
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
	CopyURL: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy url"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "select all"),
	),
	GoToChannel: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "go to channel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

var FormatListModelKeys = FormatListKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
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
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

var DownloadModelKeys = DownloadKeys{
	Pause: key.NewBinding(
		key.WithKeys("p", " "),
		key.WithHelp("p/space", "pause/resume"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("c", "esc"),
		key.WithHelp("c/esc", "cancel"),
	),
	CopyURL: key.NewBinding(
		key.WithKeys("ctrl+y"),
		key.WithHelp("ctrl+y", "copy url"),
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

var ChannelListModelKeys = ChannelListKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

var PlaylistListModelKeys = PlaylistListKeys{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}

var PlayerModelKeys = PlayerKeys{
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
