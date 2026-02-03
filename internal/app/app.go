package app

import (
	"github.com/xdagiz/xytz/internal/models"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Program       *tea.Program
	Search        models.SearchModel
	State         types.State
	Width         int
	Height        int
	Spinner       spinner.Model
	LoadingType   string
	CurrentQuery  string
	Videos        []list.Item
	VideoList     models.VideoListModel
	FormatList    models.FormatListModel
	Download      models.DownloadModel
	SelectedVideo types.VideoItem
	ErrMsg        string
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.Search.Init(), m.Spinner.Tick, m.Download.Init())
}

func NewModel() *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = sp.Style.Foreground(styles.PinkColor)

	return &Model{
		State:      types.StateSearchInput,
		Spinner:    sp,
		Search:     models.NewSearchModel(),
		VideoList:  models.NewVideoListModel(),
		FormatList: models.NewFormatListModel(),
		Download:   models.NewDownloadModel(),
	}
}
