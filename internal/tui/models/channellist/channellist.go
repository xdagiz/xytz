package channellist

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/types"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type Model struct {
	ctx          *appctx.AppContext
	Width        int
	Height       int
	List         list.Model
	CurrentQuery string
	ErrMsg       string
	prefix       string
}

type SelectedMsg struct {
	Channel types.ChannelItem
}

func NewModel(ctx *appctx.AppContext) Model {
	s := textinput.DefaultStyles(true)
	prefix := zone.NewPrefix()
	dl := styles.NewClickableDelegate(prefix, ctx.Styles.NewListDelegate())
	li := list.New([]list.Item{}, dl, 0, 0)
	li.DisableQuitKeybindings()
	li.SetShowStatusBar(false)
	li.SetShowTitle(false)
	li.SetShowHelp(false)
	li.SetStatusBarItemName("channel", "channels")
	s.Cursor.Color = ctx.Styles.AccentPrimaryColor
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(ctx.Styles.TextPrimaryColor)
	li.FilterInput.SetStyles(s)

	m := Model{
		ctx:    ctx,
		List:   li,
		prefix: prefix,
	}

	m.applyListDelegate()
	return m
}

func (m *Model) ApplyTheme() {
	m.applyListDelegate()
	s := textinput.DefaultStyles(true)
	s.Focused.Prompt = lipgloss.NewStyle().Foreground(m.ctx.Styles.TextPrimaryColor)
	s.Cursor.Color = m.ctx.Styles.AccentPrimaryColor
	m.List.FilterInput.SetStyles(s)
}

func (m *Model) applyListDelegate() {
	compact := m.ctx != nil && m.ctx.Config != nil && m.ctx.Config.ListCompactMode
	if compact {
		m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewCompactDelegate()))
		return
	}

	m.List.SetDelegate(styles.NewClickableDelegate(m.prefix, m.ctx.Styles.NewListDelegate()))
}

func (m *Model) ApplyConfig() {
	m.applyListDelegate()
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) View() string {
	var (
		s           strings.Builder
		headerText  string
		headerStyle lipgloss.Style
	)

	if m.ErrMsg != "" {
		headerStyle = m.ctx.Styles.ErrorMessageStyle.PaddingTop(1)
		headerText = fmt.Sprintf("Error: %s", m.ErrMsg)
	} else {
		headerText = fmt.Sprintf("Channels for: %s", m.CurrentQuery)
		headerStyle = m.ctx.Styles.SectionHeaderStyle
	}

	s.WriteString(headerStyle.Render(headerText))
	s.WriteRune('\n')
	s.WriteString(m.ctx.Styles.ListContainer.Render(m.List.View()))

	return s.String()
}

func (m Model) HandleResize(w, h int) Model {
	m.Width = w
	m.Height = h
	m.List.SetSize(w, h-6)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var listCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft && !m.List.SettingFilter() {
			for i := range m.List.Items() {
				if zone.Get(m.prefix + strconv.Itoa(i)).InBounds(msg) {
					if i != m.List.Index() {
						m.List.Select(i)
						return m, nil
					}

					channel, ok := m.SelectedChannel()
					if ok && channel.Name != "" {
						return m, func() tea.Msg {
							return SelectedMsg{Channel: channel}
						}
					}
				}
			}
		}

	case tea.MouseWheelMsg:
		switch msg.Button {
		case tea.MouseWheelUp:
			m.List.CursorUp()
		case tea.MouseWheelDown:
			m.List.CursorDown()
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.List.SettingFilter() {
			break
		}

		switch msg.String() {
		case "esc", "b":
			return m, func() tea.Msg {
				return types.GoBackMsg{From: types.StateChannelList, To: types.StateSearchInput}
			}

		case "enter":
			if len(m.List.Items()) > 0 {
				channel, ok := m.SelectedChannel()
				if ok && channel.Name != "" {
					return m, func() tea.Msg {
						return SelectedMsg{Channel: channel}
					}
				}
			}
		}
	}

	m.List, listCmd = m.List.Update(msg)
	return m, listCmd
}

func (m Model) SelectedChannel() (types.ChannelItem, bool) {
	selectedItem := m.List.SelectedItem()
	if channel, ok := selectedItem.(types.ChannelItem); ok {
		return channel, true
	}

	return types.ChannelItem{}, false
}

func (m *Model) SetItems(channels []types.ChannelItem) {
	items := make([]list.Item, len(channels))
	for i, c := range channels {
		items[i] = c
	}
	m.List.SetItems(items)
}
