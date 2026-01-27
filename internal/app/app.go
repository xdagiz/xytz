package app

import (
	"xytz/internal/models"
	"xytz/internal/styles"
	"xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Program      *tea.Program
	Search       models.SearchModel
	State        types.State
	Width        int
	Height       int
	Spinner      spinner.Model
	LoadingType  string
	CurrentQuery string
	Videos       []list.Item
	VideoList    models.VideoListModel
	FormatList   models.FormatListModel
	Download     models.DownloadModel
	ErrMsg       string
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.Search.Init(), m.Spinner.Tick, m.Download.Init())
}

func NewModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = s.Style.Foreground(styles.PinkColor)

	return &Model{
		Search:     models.NewSearchModel(),
		State:      types.StateSearchInput,
		Spinner:    s,
		VideoList:  models.NewVideoListModel(),
		FormatList: models.NewFormatListModel(),
		Download:   models.NewDownloadModel(),
	}
}
