package ytdlp

import (
	"strconv"
	"strings"

	"github.com/xdagiz/xytz/internal/types"
)

func ResolveQualityToFormat(quality string, videoFormats []types.FormatItem) string {
	if quality == "" || quality == "best" {
		if len(videoFormats) > 0 {
			return videoFormats[0].FormatValue
		}

		return "bv*+ba/b"
	}

	requestedHeight := parseHeight(quality)
	if requestedHeight == 0 {
		return quality
	}

	var bestMatch types.FormatItem
	found := false

	for _, item := range videoFormats {
		height := parseResolutionHeight(item.Resolution)
		if height > 0 && height <= requestedHeight {
			if !found || height > parseResolutionHeight(bestMatch.Resolution) {
				bestMatch = item
				found = true
			}
		}
	}

	if found {
		return bestMatch.FormatValue
	}

	if len(videoFormats) > 0 {
		return videoFormats[0].FormatValue
	}

	return quality
}

func parseHeight(quality string) int {
	quality = strings.ToLower(quality)
	quality = strings.TrimSuffix(quality, "p")

	height, err := strconv.Atoi(quality)
	if err != nil {
		return 0
	}

	return height
}

func parseResolutionHeight(resolution string) int {
	if resolution == "" || resolution == "?" {
		return 0
	}

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
