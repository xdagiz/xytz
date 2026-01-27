package utils

import (
	"bufio"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
)

type ProgressParser struct {
	regex *regexp.Regexp
}

func NewProgressParser() *ProgressParser {
	return &ProgressParser{
		regex: regexp.MustCompile(`(\d+(?:\.\d+)?)%`),
	}
}

func (p *ProgressParser) ReadPipe(pipe io.Reader, sendProgress func(float64, string, string)) {
	reader := bufio.NewReader(pipe)
	var lineBuilder strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil {
			if lineBuilder.Len() > 0 {
				line := lineBuilder.String()
				percent, speed, eta := p.ParseLine(line)
				if strings.Contains(line, "[download]") || percent > 0 || speed != "" || eta != "" {
					sendProgress(percent, speed, eta)
				}
			}
			break
		}

		switch r {
		// TODO: test this on windows and remove if not needed
		case '\r':
			if lineBuilder.Len() > 0 {
				line := lineBuilder.String()
				percent, speed, eta := p.ParseLine(line)
				if strings.Contains(line, "[download]") || percent > 0 || speed != "" || eta != "" {
					log.Printf("Progress parsed (\\r): %.2f%%, speed: %s, eta: %s, line: %s", percent, speed, eta, line)
					sendProgress(percent, speed, eta)
				}
				lineBuilder.Reset()
			}
		case '\n':
			if lineBuilder.Len() > 0 {
				line := lineBuilder.String()
				percent, speed, eta := p.ParseLine(line)
				if strings.Contains(line, "[download]") || percent > 0 || speed != "" || eta != "" {
					sendProgress(percent, speed, eta)
				}
				lineBuilder.Reset()
			}
		default:
			lineBuilder.WriteRune(r)
		}
	}
}

func (p *ProgressParser) ParseLine(line string) (percent float64, speed, eta string) {
	percentPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`),
		regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%`),
	}

	for _, pattern := range percentPatterns {
		percentMatch := pattern.FindStringSubmatch(line)
		if len(percentMatch) > 1 {
			if p, err := strconv.ParseFloat(percentMatch[1], 64); err == nil {
				percent = p
				break
			}
		}
	}

	speedPattern := regexp.MustCompile(`(\d+(?:\.\d+)?[KMG]?i?B/s)`)
	speedMatch := speedPattern.FindStringSubmatch(line)
	if len(speedMatch) > 1 {
		speed = speedMatch[1]
	}

	etaPattern := regexp.MustCompile(`ETA\s+(\d+:\d+(?::\d+)?)`)
	etaMatch := etaPattern.FindStringSubmatch(line)
	if len(etaMatch) > 1 {
		eta = etaMatch[1]
	}

	return percent, speed, eta
}
