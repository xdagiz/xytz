package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func FetchFormats(url string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			cfg = config.GetDefault()
		}
		ytDlpPath := cfg.YTDLPPath
		if ytDlpPath == "" {
			ytDlpPath = "yt-dlp"
		}
		cmd := exec.Command(ytDlpPath, "-J", url)
		out, err := cmd.Output()
		if err != nil {
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.SearchResultMsg{Err: errMsg}
		}
		var data map[string]any
		if err := json.Unmarshal(out, &data); err != nil {
			errMsg := fmt.Sprintf("JSON parse error: %v", err)
			return types.SearchResultMsg{Err: errMsg}
		}

		formatsAny, _ := data["formats"].([]any)
		var formats []list.Item

		audioLanguages := make(map[string]bool)
		for _, fAny := range formatsAny {
			f, ok := fAny.(map[string]any)
			if !ok {
				continue
			}

			acodec, _ := f["acodec"].(string)
			if acodec != "none" && acodec != "" {
				lang, _ := f["language"].(string)
				if lang == "" {
					lang, _ = f["lang"].(string)
				}
				if lang != "" && lang != "und" {
					audioLanguages[lang] = true
				}
			}
		}

		showLanguage := len(audioLanguages) > 1

		for _, fAny := range formatsAny {
			f, ok := fAny.(map[string]any)
			if !ok {
				continue
			}

			formatID, _ := f["format_id"].(string)
			ext, _ := f["ext"].(string)
			resolution, _ := f["resolution"].(string)
			notes, _ := f["format_note"].(string)
			acodec, _ := f["acodec"].(string)
			vcodec, _ := f["vcodec"].(string)

			if formatID == "" {
				continue
			}

			if ext == "" {
				continue
			}

			if resolution == "" || resolution == "Unknown" {
				resolution = "?"
			}
			if notes == "" {
				notes = "-"
			}

			formatType := ""
			if vcodec != "none" && vcodec != "" {
				if acodec != "none" && acodec != "" {
					formatType = "video+audio"
				} else {
					formatType = "video-only"
				}
			} else if acodec != "none" && acodec != "" {
				formatType = "audio-only"
			} else {
				formatType = "unknown"
			}

			size, _ := f["filesize"].(float64)
			sizeApprox, _ := f["filesize_approx"].(float64)
			if size == 0 {
				size = sizeApprox
			}
			sizeStr := bytesToHuman(size)

			lang := ""
			if showLanguage {
				lang, _ = f["language"].(string)
				if lang == "" {
					lang, _ = f["lang"].(string)
				}
				if lang == "" || lang == "und" {
					lang = "unknown"
				}
			}

			title := fmt.Sprintf("%s (%s %s - %s)", formatID, resolution, formatType, ext)
			if showLanguage && (acodec != "none" && acodec != "") {
				title = fmt.Sprintf("%s [%s]", title, lang)
			}

			formats = append(formats, types.FormatItem{
				FormatTitle: title,
				FormatValue: formatID,
				Size:        sizeStr,
				Language:    lang,
				Resolution:  resolution,
				FormatType:  formatType,
			})
		}

		return types.FormatResultMsg{Formats: formats}
	})
}
