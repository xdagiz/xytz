package formatlist

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/list"

	"github.com/xdagiz/xytz/internal/types"
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
	case height >= 1000:
		return "1080p"
	case height >= 700:
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

func getPreferredAudioFormat(formats []types.YtDlpFormat) (audioID string, audioLang string) {
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
			return audioID, audioLang
		}
	}

	for _, format := range formats {
		if format.Acodec != "none" && format.Acodec != "" && format.Vcodec == "none" {
			audioID = format.ID
			audioLang = format.Language
			return audioID, audioLang
		}
	}

	bestABR := 0.0
	for _, format := range formats {
		if format.Acodec != "none" && format.Acodec != "" && format.ABR > bestABR {
			bestABR = format.ABR
			audioID = format.ID
			audioLang = format.Language
		}
	}

	return audioID, audioLang
}

func bytesToHuman(bytes float64) string {
	if bytes == 0 {
		return "Unknown Size"
	}

	suffixes := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	i := 0
	for bytes >= 1024 && i < len(suffixes)-1 {
		bytes /= 1024
		i++
	}

	return fmt.Sprintf("%.2f %s", bytes, suffixes[i])
}

func formatBitrate(kbps float64) string {
	if kbps == 0 {
		return "0k"
	}

	if kbps >= 1000 {
		return fmt.Sprintf("%.1fM", kbps/1000)
	}

	return fmt.Sprintf("%.0fk", kbps)
}

func BuildFormatItems(formats []types.YtDlpFormat) (videoFormats, audioFormats, thumbnailFormats, allFormats []list.Item) {
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
		} else if format.VideoExt != "" && format.VideoExt != "none" {
			if format.AudioExt != "" && format.AudioExt != "none" {
				formatType = "video+audio"
				isVideoAudio = true
			} else {
				formatType = "video-only"
			}
		} else if format.AudioExt != "" && format.AudioExt != "none" {
			formatType = "audio-only"
			isAudioOnly = true
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

	return videoFormats, audioFormats, thumbnailFormats, allFormats
}
