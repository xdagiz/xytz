package models

import (
	"strings"

	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

type FormatAutocompleteKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Select key.Binding
}

func DefaultFormatAutocompleteKeyMap() FormatAutocompleteKeyMap {
	return FormatAutocompleteKeyMap{
		Up: key.NewBinding(
			key.WithKeys("ctrl+p", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("ctrl+n", "down"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter", "tab"),
		),
	}
}

type FormatAutocompleteModel struct {
	Visible      bool
	Filtered     []FormatMatchResult
	SelectedIdx  int
	ScrollOffset int
	Query        string
	Keys         FormatAutocompleteKeyMap
	Width        int
	MaxHeight    int
}

type FormatMatchResult struct {
	Format types.FormatItem
	Score  float64
}

func NewFormatAutocompleteModel() FormatAutocompleteModel {
	return FormatAutocompleteModel{
		Visible:      false,
		Filtered:     []FormatMatchResult{},
		SelectedIdx:  0,
		ScrollOffset: 0,
		Query:        "",
		Keys:         DefaultFormatAutocompleteKeyMap(),
		Width:        60,
		MaxHeight:    20,
	}
}

func (m *FormatAutocompleteModel) UpdateFilteredFormats(query string, allFormats []list.Item) {
	m.Query = query
	m.SelectedIdx = 0

	specialFormats := []types.FormatItem{
		{FormatTitle: "best video codec", FormatValue: "bestvideo", Size: "", Language: "", Resolution: "", FormatType: ""},
		{FormatTitle: "best audio codec", FormatValue: "bestaudio", Size: "", Language: "", Resolution: "", FormatType: ""},
		{FormatTitle: "best video + best audio", FormatValue: "bestvideo*+bestaudio", Size: "", Language: "", Resolution: "", FormatType: ""},
	}

	if query == "" {
		results := make([]FormatMatchResult, len(specialFormats))
		for i, f := range specialFormats {
			results[i] = FormatMatchResult{Format: f, Score: 1000}
		}
		m.Filtered = results
		return
	}

	lastPlus := strings.LastIndex(query, "+")
	var searchQuery string
	if lastPlus >= 0 {
		searchQuery = strings.TrimSpace(query[lastPlus+1:])
	} else {
		searchQuery = strings.TrimSpace(query)
	}

	if searchQuery == "" {
		m.Filtered = []FormatMatchResult{}
		return
	}

	patterns := make([]string, len(allFormats))
	formats := make([]types.FormatItem, len(allFormats))
	for i, item := range allFormats {
		f, ok := item.(types.FormatItem)
		if !ok {
			continue
		}

		patterns[i] = f.FormatTitle + " " + f.FormatValue + f.FormatType
		formats[i] = f
	}

	matches := fuzzy.Find(searchQuery, patterns)

	var results []FormatMatchResult

	addedFormats := make(map[int]bool)

	for _, f := range specialFormats {
		if strings.Contains(strings.ToLower(f.FormatValue), strings.ToLower(searchQuery)) {
			results = append(results, FormatMatchResult{
				Format: f,
				Score:  1001,
			})
		}
	}

	for i, f := range formats {
		if strings.Contains(strings.ToLower(f.FormatValue), strings.ToLower(searchQuery)) ||
			strings.Contains(strings.ToLower(f.FormatTitle), strings.ToLower(searchQuery)) {
			results = append(results, FormatMatchResult{
				Format: f,
				Score:  1000,
			})

			addedFormats[i] = true
		}
	}

	for _, match := range matches {
		if !addedFormats[match.Index] {
			f := formats[match.Index]
			results = append(results, FormatMatchResult{
				Format: f,
				Score:  float64(match.Score),
			})
		}
	}

	m.Filtered = results
}

func (m *FormatAutocompleteModel) Show(query string, allFormats []list.Item) {
	m.Visible = true
	m.UpdateFilteredFormats(query, allFormats)
}

func (m *FormatAutocompleteModel) Hide() {
	m.Visible = false
	m.Filtered = []FormatMatchResult{}
	m.SelectedIdx = 0
	m.ScrollOffset = 0
	m.Query = ""
}

func (m *FormatAutocompleteModel) Next() {
	if len(m.Filtered) == 0 {
		return
	}

	m.SelectedIdx++
	if m.SelectedIdx >= len(m.Filtered) {
		m.SelectedIdx = 0
	}
}

func (m *FormatAutocompleteModel) Prev() {
	if len(m.Filtered) == 0 {
		return
	}

	m.SelectedIdx--
	if m.SelectedIdx < 0 {
		m.SelectedIdx = len(m.Filtered) - 1
	}
}

func (m *FormatAutocompleteModel) updateScrollOffset(height int) {
	if len(m.Filtered) == 0 {
		return
	}

	visibleItems := min(height-2, m.MaxHeight)

	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}

	if m.ScrollOffset > len(m.Filtered)-visibleItems {
		m.ScrollOffset = max(0, len(m.Filtered)-visibleItems)
	}

	if m.SelectedIdx < m.ScrollOffset {
		m.ScrollOffset = m.SelectedIdx
	}

	if m.SelectedIdx >= m.ScrollOffset+visibleItems {
		m.ScrollOffset = m.SelectedIdx - visibleItems + 1
	}
}

func (m *FormatAutocompleteModel) Update(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Visible {
			return false, nil
		}

		switch {
		case key.Matches(msg, m.Keys.Up):
			m.Prev()
			return true, nil
		case key.Matches(msg, m.Keys.Down):
			m.Next()
			return true, nil
		case key.Matches(msg, m.Keys.Select):
			return true, nil
		}
	}

	return false, nil
}

func (m *FormatAutocompleteModel) HandleResize(width, height int) {
	m.Width = width - 4
}

func (m *FormatAutocompleteModel) View(width, height int) string {
	if !m.Visible || len(m.Filtered) == 0 {
		return ""
	}

	if height < 4 {
		return ""
	}

	m.updateScrollOffset(height)

	var b strings.Builder

	visibleItems := min(height-2, m.MaxHeight)

	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}

	if m.ScrollOffset > len(m.Filtered)-visibleItems {
		m.ScrollOffset = max(0, len(m.Filtered)-visibleItems)
	}

	for i := range visibleItems {
		idx := m.ScrollOffset + i
		if idx >= len(m.Filtered) {
			break
		}

		result := m.Filtered[idx]
		isSelected := idx == m.SelectedIdx

		itemText := result.Format.FormatValue
		if result.Format.Resolution != "" {
			itemText += " - " + result.Format.Resolution
		}

		if result.Format.FormatType != "" {
			if strings.Contains(result.Format.Resolution, "audio") && strings.Contains(result.Format.FormatType, "audio") {
			} else {
				itemText += " - " + result.Format.FormatType
			}
		}

		if result.Format.Language != "" {
			itemText += " [" + result.Format.Language + "]"
		}

		if len(itemText) > width-4 {
			itemText = itemText[:width-7] + "..."
		}

		var itemStyle string
		if isSelected {
			itemStyle = styles.AutocompleteSelected.Render("> " + itemText)
		} else {
			itemStyle = styles.AutocompleteItem.Render("  " + itemText)
		}

		b.WriteString(itemStyle)

		if i < visibleItems-1 && idx < len(m.Filtered)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m *FormatAutocompleteModel) SelectedFormat() *types.FormatItem {
	if m.SelectedIdx >= 0 && m.SelectedIdx < len(m.Filtered) {
		return &m.Filtered[m.SelectedIdx].Format
	}

	return nil
}
