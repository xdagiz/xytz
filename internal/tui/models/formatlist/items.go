package formatlist

import (
	"fmt"
	"sort"
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

func isAudioOnlyFormat(f types.YtDlpFormat) bool {
	return f.Acodec != "" &&
		f.Acodec != "none" &&
		(f.Vcodec == "" || f.Vcodec == "none")
}

func baseFormatID(id string) string {
	if i := strings.Index(id, "-"); i >= 0 {
		return id[:i]
	}
	return id
}

func isDRCFormat(id string) bool {
	for _, part := range strings.Split(id, "-") {
		if part == "drc" {
			return true
		}
	}
	return false
}

func hasOriginalNote(note string) bool {
	return strings.Contains(strings.ToLower(note), "original")
}

func bestOriginalByABR(formats []types.YtDlpFormat) (audioID string, audioLang string, ok bool) {
	bestABR := -1.0
	for _, f := range formats {
		if !isAudioOnlyFormat(f) {
			continue
		}

		if isDRCFormat(f.ID) {
			continue
		}

		if !hasOriginalNote(f.Note) {
			continue
		}

		if f.ABR > bestABR {
			bestABR = f.ABR
			audioID = f.ID
			audioLang = f.Language
			ok = true
		}
	}

	return audioID, audioLang, ok
}

func bestInBaseByABR(formats []types.YtDlpFormat, base string) (audioID string, audioLang string, ok bool) {
	var candidates []types.YtDlpFormat
	for _, f := range formats {
		if !isAudioOnlyFormat(f) {
			continue
		}

		if baseFormatID(f.ID) != base {
			continue
		}

		if isDRCFormat(f.ID) {
			continue
		}

		candidates = append(candidates, f)
	}

	if len(candidates) == 0 {
		return "", "", false
	}

	if id, lang, found := bestOriginalByABR(candidates); found {
		return id, lang, true
	}

	bestABR := -1.0
	for _, f := range candidates {
		if f.ABR > bestABR {
			bestABR = f.ABR
			audioID = f.ID
			audioLang = f.Language
			ok = true
		}
	}

	return audioID, audioLang, ok
}

func formatItemHeight(item types.FormatItem) int {
	if item.Height > 0 {
		return item.Height
	}
	return resolutionHeight(item.Resolution)
}

func resolutionHeight(resolution string) int {
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return 0
	}

	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}

	return height
}

func bestPreferredAudio(formats []types.YtDlpFormat, allowDRC bool) (audioID string, audioLang string, ok bool) {
	bestPref := 0
	bestABR := -1.0

	for _, format := range formats {
		if !isAudioOnlyFormat(format) {
			continue
		}

		if !allowDRC && isDRCFormat(format.ID) {
			continue
		}

		if format.LanguagePreference <= 0 {
			continue
		}

		if !ok || format.LanguagePreference > bestPref ||
			(format.LanguagePreference == bestPref && format.ABR > bestABR) {
			bestPref = format.LanguagePreference
			bestABR = format.ABR
			audioID = format.ID
			audioLang = format.Language
			ok = true
		}
	}

	return audioID, audioLang, ok
}

func getPreferredAudioFormat(formats []types.YtDlpFormat) (audioID string, audioLang string) {
	if id, lang, ok := bestPreferredAudio(formats, false); ok {
		return id, lang
	}

	if id, lang, ok := bestOriginalByABR(formats); ok {
		return id, lang
	}

	if id, lang, ok := bestPreferredAudio(formats, true); ok {
		return id, lang
	}

	hasFormat140 := false
	hasFormat251 := false

	for _, format := range formats {
		if !isAudioOnlyFormat(format) {
			continue
		}

		formatID := format.ID
		if formatID == "140" {
			hasFormat140 = true
		}
		if formatID == "251" {
			hasFormat251 = true
		}
	}

	if hasFormat140 {
		audioID = "140"
	} else if hasFormat251 {
		audioID = "251"
	}

	for _, format := range formats {
		formatID := format.ID
		if formatID == audioID {
			audioLang = format.Language
			return audioID, audioLang
		}
	}

	if id, lang, ok := bestInBaseByABR(formats, "140"); ok {
		return id, lang
	}

	if id, lang, ok := bestInBaseByABR(formats, "251"); ok {
		return id, lang
	}

	for _, format := range formats {
		if format.Acodec != "none" && format.Acodec != "" &&
			(format.Vcodec == "none" || format.Vcodec == "") {
			audioID = format.ID
			audioLang = format.Language
			return audioID, audioLang
		}
	}

	fallbackABR := 0.0
	for _, format := range formats {
		if format.Acodec != "none" && format.Acodec != "" && format.ABR > fallbackABR {
			fallbackABR = format.ABR
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
			switch {
			case format.VideoExt != "" && format.VideoExt != "none":
				ext = format.VideoExt
			case format.AudioExt != "" && format.AudioExt != "none":
				ext = format.AudioExt
			default:
				continue
			}
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
			Height:      format.Height,
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
			if audioID == "" {
				continue
			}

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
				Height:      format.Height,
				FormatType:  "video-only+audio-only",
				ABR:         0,
				VideoSize:   videoSize,
				AudioSize:   audioSize,
			}

			videoFormats = append(videoFormats, preset)
		}
	}

	sort.SliceStable(videoFormats, func(i, j int) bool {
		hi := videoFormats[i].(types.FormatItem)
		hj := videoFormats[j].(types.FormatItem)
		return formatItemHeight(hi) > formatItemHeight(hj)
	})

	return videoFormats, audioFormats, thumbnailFormats, allFormats
}
