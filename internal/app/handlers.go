package app

import (
	"xytz/internal/types"
)

func HandleBack(m *Model) *Model {
	switch m.State {
	case types.StateVideoList:
		m.State = types.StateSearchInput
		m.ErrMsg = ""
	case types.StateFormatList:
		m.State = types.StateVideoList
		m.ErrMsg = ""
		// case types.StateDownload:
		// 	m.State = types.StateFormatList
	}
	return m
}
