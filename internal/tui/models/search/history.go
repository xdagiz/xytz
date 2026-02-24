package search

import (
	"log"

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
		log.Printf("Failed to load history: %v", err)
		h.items = []string{}
	} else {
		h.items = history
	}
}

func (h *HistoryNavigator) Add(query string) {
	if err := utils.AddToHistory(query); err != nil {
		log.Printf("Failed to save history: %v", err)
	}
	h.index = -1
	h.originalQuery = ""
	h.Load()
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
