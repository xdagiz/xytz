package search

import (
	"strings"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/utils"
)

type HistoryNavigator struct {
	items         []string
	index         int
	originalQuery string
}

func NewHistoryNavigator() HistoryNavigator {
	h := HistoryNavigator{index: -1}
	h.Load()

	return h
}

func (h *HistoryNavigator) Load() {
	history, err := utils.LoadHistory()
	if err != nil {
		log.Error("failed to load history", "err", err)
		h.items = []string{}
	} else {
		h.items = history
	}
}

func (h *HistoryNavigator) Add(query string) {
	if err := utils.AddToHistory(query); err != nil {
		log.Error("failed to save history", "err", err)
	}
	h.index = -1
	h.originalQuery = ""
	h.Load()
}

func (h *HistoryNavigator) AddLocal(query string) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}

	var newHistory []string
	for _, entry := range h.items {
		if entry != query {
			newHistory = append(newHistory, entry)
		}
	}

	newHistory = append([]string{query}, newHistory...)
	if len(newHistory) > 1000 {
		newHistory = newHistory[:1000]
	}

	h.items = newHistory
	h.index = -1
	h.originalQuery = ""
}

func (h *HistoryNavigator) Navigate(dir int, getCurrentValue func() string, setValue func(string)) {
	if h.index == -1 {
		h.originalQuery = getCurrentValue()
	}

	newIndex := h.index + dir

	if newIndex < 0 {
		h.index = -1
		setValue(h.originalQuery)
	} else if newIndex >= len(h.items) {
		h.index = len(h.items) - 1
	} else {
		h.index = newIndex
		setValue(h.items[newIndex])
	}
}

func (h *HistoryNavigator) TrackEdit(oldValue, newValue string) {
	if h.index >= 0 && h.index < len(h.items) {
		expectedValue := h.items[h.index]
		if oldValue != newValue && newValue != expectedValue {
			h.index = -1
			h.originalQuery = ""
		}
	}
}

func (h *HistoryNavigator) Reset() {
	h.index = -1
	h.originalQuery = ""
}
