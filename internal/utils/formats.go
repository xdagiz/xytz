package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	formatsCmd      *exec.Cmd
	formatsMutex    sync.Mutex
	formatsCanceled bool
)

func formatQuality(resolution string) string {
	if resolution == "" || resolution == "?" {
		return resolution
	}

	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return resolution
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return resolution
	}

	switch {
	case height >= 4320:
		return "8k"
	case height >= 2160:
		return "4k"
	case height >= 1440:
		return "2k"
	case height >= 1080:
		return "1080p"
	case height >= 720:
		return "720p"
	case height >= 480:
		return "480p"
	case height >= 360:
		return "360p"
	case height >= 240:
		return "240p"
	case height >= 144:
		return "144p"
	default:
		return resolution
	}
}

func getPreferredAudioFormat(formatsAny []any) (audioID string, audioLang string) {
	hasFormat140 := false
	hasFormat251 := false
	audioID = "140"
	audioLang = ""

	for _, fAny := range formatsAny {
		f, ok := fAny.(map[string]any)
		if !ok {
			continue
		}
		formatID, _ := f["format_id"].(string)
		if formatID == "140" {
			hasFormat140 = true
		}
		if formatID == "251" {
			hasFormat251 = true
		}
	}

	if !hasFormat140 && hasFormat251 {
		audioID = "251"
	}

	for _, fAny := range formatsAny {
		f, ok := fAny.(map[string]any)
		if !ok {
			continue
		}

		formatID, _ := f["format_id"].(string)
		if formatID == audioID {
			audioLang, _ = f["language"].(string)
			if audioLang == "" {
				audioLang, _ = f["lang"].(string)
			}

			break
		}
	}

	return audioID, audioLang
}

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

		formatsMutex.Lock()
		formatsCmd = cmd
		formatsMutex.Unlock()

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		if err := cmd.Start(); err != nil {
			formatsMutex.Lock()
			formatsCmd = nil
			formatsMutex.Unlock()
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		out, err := io.ReadAll(stdout)
		stdout.Close()

		formatsMutex.Lock()
		wasCancelled := formatsCanceled
		formatsCanceled = false
		formatsCmd = nil
		formatsMutex.Unlock()

		if wasCancelled {
			return nil
		}

		if err != nil {
			log.Printf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: fmt.Sprintf("Format fetch error: %v", err)}
		}

		if len(out) == 0 {
			return types.FormatResultMsg{Err: "No formats found"}
		}

		var data map[string]any
		if err := json.Unmarshal(out, &data); err != nil {
			errMsg := fmt.Sprintf("JSON parse error: %v", err)
			return types.SearchResultMsg{Err: errMsg}
		}

		formatsAny, _ := data["formats"].([]any)
		var videoFormats []list.Item
		var audioFormats []list.Item
		var thumbnailFormats []list.Item
		var allFormats []list.Item

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
			acodec, _ := f["acodec"].(string)
			vcodec, _ := f["vcodec"].(string)
			abr, _ := f["abr"].(float64)
			fps, _ := f["fps"].(float64)
			tbr, _ := f["tbr"].(float64)

			if formatID == "" {
				continue
			}

			if ext == "" {
				continue
			}

			if resolution == "" || resolution == "Unknown" {
				resolution = "?"
			}

			formatType := ""
			isVideoAudio := false
			isAudioOnly := false
			isThumbnail := ext == "mhtml"

			if vcodec != "none" && vcodec != "" {
				if acodec != "none" && acodec != "" {
					formatType = "video+audio"
					isVideoAudio = true
				} else {
					formatType = "video-only"
				}
			} else if acodec != "none" && acodec != "" {
				formatType = "audio-only"
				isAudioOnly = true
			} else if isThumbnail {
				formatType = "thumbnail"
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

			title := ext
			if isAudioOnly {
				if abr > 0 {
					title = fmt.Sprintf("%s @%dk", ext, int(abr))
				}
			} else if isThumbnail {
				title = formatQuality(resolution)
			} else {
				quality := formatQuality(resolution)
				if fps > 0 {
					quality = fmt.Sprintf("%s%.0f", quality, fps)
				}
				title = quality
				if tbr > 0 {
					title = fmt.Sprintf("%s @%s", title, formatBitrate(tbr))
				}
				title = fmt.Sprintf("%s %s", title, ext)
			}

			if showLanguage && (acodec != "none" && acodec != "") {
				title = fmt.Sprintf("%s [%s]", title, lang)
			}

			formatItem := types.FormatItem{
				FormatTitle: title,
				FormatValue: formatID,
				Size:        sizeStr,
				Language:    lang,
				Resolution:  resolution,
				FormatType:  formatType,
			}

			allFormats = append(allFormats, formatItem)

			if isVideoAudio {
				if !strings.Contains(title, "144p") && !strings.Contains(title, "240p") {
					videoFormats = append(videoFormats, formatItem)
				}
			} else if isAudioOnly {
				audioFormats = append(audioFormats, formatItem)
			} else if isThumbnail {
				thumbnailFormats = append(thumbnailFormats, formatItem)
			}
		}

		audioID, audioLang := getPreferredAudioFormat(formatsAny)

		for _, fAny := range formatsAny {
			f, ok := fAny.(map[string]any)
			if !ok {
				continue
			}
			formatID, _ := f["format_id"].(string)
			vcodec, _ := f["vcodec"].(string)
			acodec, _ := f["acodec"].(string)
			resolution, _ := f["resolution"].(string)
			fps, _ := f["fps"].(float64)
			tbr, _ := f["tbr"].(float64)

			if vcodec != "none" && vcodec != "" && (acodec == "none" || acodec == "") {
				quality := formatQuality(resolution)
				if quality == "144p" || quality == "240p" {
					continue
				}

				if fps > 0 {
					quality = fmt.Sprintf("%s%.0f", quality, fps)
				}

				title := quality
				if title == resolution || title == "?" {
					title = resolution
				}

				if tbr > 0 {
					title = fmt.Sprintf("%s @%s", title, formatBitrate(tbr))
				}

				title = fmt.Sprintf("%s mp4", title)

				if audioLang != "" && audioLang != "und" {
					title = fmt.Sprintf("%s [%s]", title, audioLang)
				}

				preset := types.FormatItem{
					FormatTitle: title,
					FormatValue: formatID + "+" + audioID,
					Size:        "unknown size",
					Language:    audioLang,
					Resolution:  resolution,
					FormatType:  "video-only+audio-only",
				}

				videoFormats = append(videoFormats, preset)
			}
		}

		return types.FormatResultMsg{
			VideoFormats:     videoFormats,
			AudioFormats:     audioFormats,
			ThumbnailFormats: thumbnailFormats,
			AllFormats:       allFormats,
		}
	})
}

func CancelFormats() tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		formatsMutex.Lock()

		if formatsCmd != nil && formatsCmd.Process != nil {
			formatsCanceled = true
			if err := formatsCmd.Process.Kill(); err != nil {
				log.Printf("Failed to kill formats process: %v", err)
			}
		}

		formatsMutex.Unlock()
		return types.CancelFormatsMsg{}
	})
}
