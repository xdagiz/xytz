package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/types"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
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

func FetchFormats(fm *FormatsManager, cfg *config.Config, url string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if cfg == nil {
			cfg = config.GetDefault()
		}

		ytDlpPath := cfg.YTDLPPath
		if ytDlpPath == "" {
			ytDlpPath = "yt-dlp"
		}

		cmd := exec.Command(ytDlpPath, "-J", url)

		fm.SetCmd(cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		if err := cmd.Start(); err != nil {
			fm.Clear()
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		out, err := io.ReadAll(stdout)
		if closeErr := stdout.Close(); closeErr != nil {
			log.Printf("failed to close formats stdout: %v", closeErr)
		}

		if fm.ClearAndCheckCanceled() {
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

		videoInfo := extractVideoInfo(data)

		formatsAny, ok := data["formats"].([]any)
		if !ok {
			log.Printf("Warning: No formats found in yt-dlp output")
			formatsAny = []any{}
		}

		var (
			videoFormats     []list.Item
			audioFormats     []list.Item
			thumbnailFormats []list.Item
			allFormats       []list.Item
		)

		audioLanguages := make(map[string]bool)
		for _, fAny := range formatsAny {
			f, ok := fAny.(map[string]any)
			if !ok {
				continue
			}

			acodec, ok := f["acodec"].(string)
			if !ok {
				acodec = ""
			}
			if acodec != "none" && acodec != "" {
				lang, ok := f["language"].(string)
				if !ok || lang == "" {
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

			formatID, ok := f["format_id"].(string)
			if !ok || formatID == "" {
				continue
			}
			ext, ok := f["ext"].(string)
			if !ok || ext == "" {
				continue
			}
			resolution, _ := f["resolution"].(string)
			acodec, ok := f["acodec"].(string)
			if !ok {
				acodec = ""
			}
			vcodec, ok := f["vcodec"].(string)
			if !ok {
				vcodec = ""
			}
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
					title = fmt.Sprintf("%dk", int(abr))
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
				ABR:         abr,
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

		formatSizes := make(map[string]float64)
		for _, fAny := range formatsAny {
			f, ok := fAny.(map[string]any)
			if !ok {
				continue
			}

			formatID, _ := f["format_id"].(string)
			if formatID != "" {
				size, _ := f["filesize"].(float64)
				if size == 0 {
					size, _ = f["filesize_approx"].(float64)
				}

				formatSizes[formatID] = size
			}
		}

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

				videoSize := 0.0
				audioSize := 0.0

				videoSize, _ = f["filesize"].(float64)
				if videoSize == 0 {
					videoSize, _ = f["filesize_approx"].(float64)
				}

				audioSize = formatSizes[audioID]

				var sizeStr string
				if videoSize > 0 && audioSize > 0 {
					totalSize := videoSize + audioSize
					sizeStr = bytesToHuman(totalSize)
				} else {
					sizeStr = "unknown size"
				}

				preset := types.FormatItem{
					FormatTitle: title,
					FormatValue: formatID + "+" + audioID,
					Size:        sizeStr,
					Language:    audioLang,
					Resolution:  resolution,
					FormatType:  "video-only+audio-only",
					ABR:         0,
					VideoSize:   videoSize,
					AudioSize:   audioSize,
				}

				videoFormats = append(videoFormats, preset)
			}
		}

		return types.FormatResultMsg{
			VideoFormats:     videoFormats,
			AudioFormats:     audioFormats,
			ThumbnailFormats: thumbnailFormats,
			AllFormats:       allFormats,
			VideoInfo:        videoInfo,
		}
	})
}

func extractVideoInfo(data map[string]any) types.VideoItem {
	videoID, _ := data["id"].(string)
	title, _ := data["title"].(string)
	channel, _ := data["uploader"].(string)

	var viewCount float64
	if vc, ok := data["view_count"]; ok {
		viewCount = parseFloat(vc)
	}

	var duration float64
	if d, ok := data["duration"]; ok {
		duration = parseFloat(d)
	}

	viewsStr := FormatNumber(viewCount)
	durationStr := FormatDuration(duration)

	if len(channel) > 30 {
		channel = channel[:27] + "..."
	}

	desc := fmt.Sprintf("%s • %s views • %s", durationStr, viewsStr, channel)

	return types.VideoItem{
		ID:         videoID,
		VideoTitle: title,
		Desc:       desc,
		Views:      viewCount,
		Duration:   duration,
		Channel:    channel,
	}
}

func CancelFormats(fm *FormatsManager) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if err := fm.Cancel(); err != nil {
			log.Printf("Failed to cancel formats: %v", err)
		}

		return types.CancelFormatsMsg{}
	})
}

func FetchVideoInfo(fm *FormatsManager, cfg *config.Config, url string) tea.Cmd {
	return tea.Cmd(func() tea.Msg {
		if cfg == nil {
			cfg = config.GetDefault()
		}

		ytDlpPath := cfg.YTDLPPath
		if ytDlpPath == "" {
			ytDlpPath = "yt-dlp"
		}

		cmd := exec.Command(ytDlpPath, "-J", url)

		fm.SetCmd(cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to get video info: %v", err)}
		}

		if err := cmd.Start(); err != nil {
			fm.Clear()
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to start yt-dlp: %v", err)}
		}

		out, err := io.ReadAll(stdout)
		if closeErr := stdout.Close(); closeErr != nil {
			log.Printf("failed to close video info stdout: %v", closeErr)
		}

		if fm.ClearAndCheckCanceled() {
			return types.PlayURLResultMsg{URL: url, Err: "Canceled"}
		}

		if err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to read video info: %v", err)}
		}

		if len(out) == 0 {
			return types.PlayURLResultMsg{URL: url, Err: "No video info found"}
		}

		var data map[string]any
		if err := json.Unmarshal(out, &data); err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to parse video info: %v", err)}
		}

		videoInfo := extractVideoInfo(data)
		if videoInfo.ID == "" {
			return types.PlayURLResultMsg{URL: url, Err: "Could not extract video ID from URL"}
		}

		return types.PlayURLResultMsg{
			URL:           url,
			SelectedVideo: videoInfo,
		}
	})
}
