package utils

import (
	"bufio"
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

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
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

func getPreferredAudioFormat(formats []YtDlpFormat) (audioID string, audioLang string) {
	hasFormat140 := false
	hasFormat251 := false
	audioID = "140"
	audioLang = ""

	for _, format := range formats {
		formatID := format.ID
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

	for _, format := range formats {
		formatID := format.ID
		if formatID == audioID {
			audioLang = format.Language
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

		args := []string{"-J", url}
		if cfg.JSRuntime != "" {
			jsRuntimeArg := cfg.JSRuntime
			if cfg.JSRuntimePath != "" {
				jsRuntimeArg = cfg.JSRuntime + ":" + cfg.JSRuntimePath
			}
			args = append([]string{"--js-runtimes", jsRuntimeArg}, args...)
		}
		if cfg.CookiesBrowser != "" {
			args = append([]string{"--cookies-from-browser", cfg.CookiesBrowser}, args...)
		} else if cfg.CookiesFile != "" {
			args = append([]string{"--cookies", cfg.CookiesFile}, args...)
		}

		cmd := exec.Command(ytDlpPath, args...)

		fm.SetCmd(cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		if err := cmd.Start(); err != nil {
			fm.Clear()
			errMsg := fmt.Sprintf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		var (
			stderrLines []string
			stderrWg    sync.WaitGroup
		)
		stderrWg.Add(1)
		go func() {
			defer stderrWg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				stderrLines = append(stderrLines, line)
				log.Printf("yt-dlp stderr: %s", line)
			}
		}()

		out, err := io.ReadAll(stdout)
		if closeErr := stdout.Close(); closeErr != nil {
			log.Printf("failed to close formats stdout: %v", closeErr)
		}
		waitErr := cmd.Wait()
		stderrWg.Wait()
		if closeErr := stderr.Close(); closeErr != nil {
			log.Printf("failed to close formats stderr: %v", closeErr)
		}

		if fm.ClearAndCheckCanceled() {
			return nil
		}

		if waitErr != nil {
			log.Printf("yt-dlp formats command failed: %v, stderr: %v", waitErr, stderrLines)
		}
		if err == nil && waitErr != nil {
			err = waitErr
		}

		if err != nil {
			log.Printf("Format fetch error: %v", err)
			return types.FormatResultMsg{Err: fmt.Sprintf("Format fetch error: %v", err)}
		}

		if len(out) == 0 {
			return types.FormatResultMsg{Err: "No formats found"}
		}

		var data YtDlpVideo
		if err := json.Unmarshal(out, &data); err != nil {
			errMsg := fmt.Sprintf("JSON parse error: %v", err)
			return types.FormatResultMsg{Err: errMsg}
		}

		videoInfo := extractVideoInfo(data)

		formats := data.Formats

		var (
			videoFormats     []list.Item
			audioFormats     []list.Item
			thumbnailFormats []list.Item
			allFormats       []list.Item
		)

		audioLanguages := make(map[string]bool)
		for _, format := range formats {
			acodec := format.Acodec
			if acodec != "none" && acodec != "" {
				lang := format.Language
				if lang != "" && lang != "und" {
					audioLanguages[lang] = true
				}
			}
		}

		showLanguage := len(audioLanguages) > 1

		for _, format := range formats {
			formatID := format.ID
			if formatID == "" {
				continue
			}

			ext := format.Ext
			if ext == "" {
				continue
			}
			resolution := format.Resolution
			acodec := format.Acodec
			vcodec := format.Vcodec
			abr := format.ABR
			fps := format.FPS
			tbr := format.TBR

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

			size := format.Filesize
			sizeApprox := format.FilesizeApprox
			if size == 0 {
				size = sizeApprox
			}
			sizeStr := bytesToHuman(size)

			lang := ""
			if showLanguage {
				lang = format.Language
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

		audioID, audioLang := getPreferredAudioFormat(formats)

		formatSizes := make(map[string]float64)
		for _, format := range formats {
			formatID := format.ID
			if formatID != "" {
				size := format.Filesize
				if size == 0 {
					size = format.FilesizeApprox
				}

				formatSizes[formatID] = size
			}
		}

		for _, format := range formats {
			formatID := format.ID
			vcodec := format.Vcodec
			acodec := format.Acodec
			resolution := format.Resolution
			fps := format.FPS
			tbr := format.TBR

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

				videoSize = format.Filesize
				if videoSize == 0 {
					videoSize = format.FilesizeApprox
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

func extractVideoInfo(data YtDlpVideo) types.VideoItem {
	videoID := data.ID
	title := data.Title
	channel := data.Uploader
	uploadDate := data.UploadDate

	viewCount := float64(data.ViewCount)
	duration := float64(data.Duration)

	viewsStr := FormatNumber(viewCount)
	durationStr := FormatDuration(duration)
	formattedUploadDate := FormatUploadDate(uploadDate, "simple")

	if len(channel) > 30 {
		channel = channel[:27] + "..."
	}

	desc := fmt.Sprintf("%s • %s views • %s • %s", channel, viewsStr, durationStr, formattedUploadDate)

	return types.VideoItem{
		ID:         videoID,
		VideoTitle: title,
		Desc:       desc,
		Views:      viewCount,
		Duration:   duration,
		Channel:    channel,
		UploadDate: uploadDate,
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

		cmdArgs := []string{"-J", url}
		if cfg.JSRuntime != "" {
			jsRuntimeArg := cfg.JSRuntime
			if cfg.JSRuntimePath != "" {
				jsRuntimeArg = cfg.JSRuntime + ":" + cfg.JSRuntimePath
			}
			cmdArgs = append([]string{"--js-runtimes", jsRuntimeArg}, cmdArgs...)
		}
		cmd := exec.Command(ytDlpPath, cmdArgs...)

		fm.SetCmd(cmd)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to get video info: %v", err)}
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to get video info: %v", err)}
		}

		if err := cmd.Start(); err != nil {
			fm.Clear()
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to start yt-dlp: %v", err)}
		}

		var (
			stderrLines []string
			stderrWg    sync.WaitGroup
		)
		stderrWg.Add(1)
		go func() {
			defer stderrWg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				line := scanner.Text()
				stderrLines = append(stderrLines, line)
				log.Printf("yt-dlp stderr: %s", line)
			}
		}()

		out, err := io.ReadAll(stdout)
		if closeErr := stdout.Close(); closeErr != nil {
			log.Printf("failed to close video info stdout: %v", closeErr)
		}
		waitErr := cmd.Wait()
		stderrWg.Wait()
		if closeErr := stderr.Close(); closeErr != nil {
			log.Printf("failed to close video info stderr: %v", closeErr)
		}

		if fm.ClearAndCheckCanceled() {
			return types.PlayURLResultMsg{URL: url, Err: "Canceled"}
		}

		if waitErr != nil {
			log.Printf("yt-dlp video info command failed: %v, stderr: %v", waitErr, stderrLines)
		}
		if err == nil && waitErr != nil {
			err = waitErr
		}

		if err != nil {
			return types.PlayURLResultMsg{URL: url, Err: fmt.Sprintf("Failed to read video info: %v", err)}
		}

		if len(out) == 0 {
			return types.PlayURLResultMsg{URL: url, Err: "No video info found"}
		}

		var data YtDlpVideo
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
