package utils

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	rePercent1    = regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	rePercent2    = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`)
	reSpeed       = regexp.MustCompile(`(\d+(?:\.\d+)?[KMG]?i?B/s)`)
	reETA         = regexp.MustCompile(`ETA\s+(\d+:\d+(?::\d+)?)`)
	reDestination = regexp.MustCompile(`Destination:\s*(.+)`)
	reFormat      = regexp.MustCompile(`(?:format|format_id)\s+(\d+)`)
)

type ProgressParser struct{}

func NewProgressParser() *ProgressParser {
	return &ProgressParser{}
}

func (p *ProgressParser) ReadPipe(pipe io.Reader, sendProgress func(float64, string, string, string, string)) {
	reader := bufio.NewReader(pipe)
	var lineBuilder strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			if lineBuilder.Len() > 0 {
				line := lineBuilder.String()
				percent, speed, eta, status, destination := p.ParseLine(line)
				if strings.Contains(line, "[download]") || percent > 0 || speed != "" || eta != "" {
					sendProgress(percent, speed, eta, status, destination)
				}
			}

			break
		}

		switch r {
		// TODO: test this on windows and remove if not needed
		case '\r':
			if lineBuilder.Len() > 0 {
				line := lineBuilder.String()
				percent, speed, eta, status, destination := p.ParseLine(line)
				if strings.Contains(line, "[download]") || percent > 0 || speed != "" || eta != "" {
					sendProgress(percent, speed, eta, status, destination)
				}

				lineBuilder.Reset()
			}

		case '\n':
			if lineBuilder.Len() > 0 {
				line := lineBuilder.String()
				percent, speed, eta, status, destination := p.ParseLine(line)
				if strings.Contains(line, "[download]") || percent > 0 || speed != "" || eta != "" {
					sendProgress(percent, speed, eta, status, destination)
				}
				lineBuilder.Reset()
			}

		default:
			lineBuilder.WriteRune(r)
		}
	}
}

func (p *ProgressParser) ParseLine(line string) (percent float64, speed, eta, status, destination string) {
	currentDestination := ""
	currentFormat := ""

	percentPatterns := []*regexp.Regexp{rePercent1, rePercent2}

	for _, pattern := range percentPatterns {
		percentMatch := pattern.FindStringSubmatch(line)
		if len(percentMatch) > 1 {
			if pr, err := strconv.ParseFloat(percentMatch[1], 64); err == nil {
				percent = pr
				break
			}
		}
	}

	speedMatch := reSpeed.FindStringSubmatch(line)
	if len(speedMatch) > 1 {
		speed = speedMatch[1]
	}

	etaMatch := reETA.FindStringSubmatch(line)
	if len(etaMatch) > 1 {
		eta = etaMatch[1]
	}

	if strings.Contains(line, "[download] Destination:") {
		if match := reDestination.FindStringSubmatch(line); len(match) > 1 {
			currentDestination = strings.TrimSpace(match[1])
		}

		if ext := extractFormatFromDestination(line); ext != "" {
			currentFormat = ext
		}
	}

	if match := reFormat.FindStringSubmatch(line); len(match) > 1 {
		currentFormat = "format " + match[1]
	}

	if percent > 0 {
		if currentFormat != "" {
			status = "[download] " + currentFormat
		} else {
			status = "[download]"
		}
	}

	return percent, speed, eta, status, currentDestination
}

func extractFormatFromDestination(line string) string {
	videoExtensions := map[string]bool{
		".mp4":  true,
		".webm": true,
		".mkv":  true,
	}
	audioExtensions := map[string]bool{
		".m4a":  true,
		".mp3":  true,
		".ogg":  true,
		".wav":  true,
		".flac": true,
		".aac":  true,
	}

	for ext := range videoExtensions {
		if strings.Contains(line, ext) {
			return "video"
		}
	}

	for ext := range audioExtensions {
		if strings.Contains(line, ext) {
			return "audio"
		}
	}

	return ""
}
