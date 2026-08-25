package models

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/utils"
)

func SpotifyInfoView(st styles.Styles, title, artist, album, releaseDate string, duration float64) string {
	s := strings.Builder{}
	s.WriteRune('\n')
	s.WriteString(st.SectionHeaderStyle.Render("♪ " + title))
	s.WriteRune('\n')

	if artist != "" {
		s.WriteString(st.MutedStyle.Render("🎙️ " + artist))
		s.WriteRune('\n')
	}

	if album != "" {
		s.WriteString(st.MutedStyle.Render("💿 " + album))
		s.WriteRune('\n')
	}

	if releaseDate != "" {
		s.WriteString(st.MutedStyle.Render("🗓  " + releaseDate))
		s.WriteRune('\n')
	}

	if duration > 0 {
		s.WriteString(st.MutedStyle.Render("⏱  " + utils.FormatDuration(duration)))
		s.WriteRune('\n')
	}

	return s.String()
}

func VideoInfoView(st styles.Styles, title, channel, url, uploadDate string, duration, views float64, size, siteName string) string {
	s := strings.Builder{}
	s.WriteString(st.SectionHeaderStyle.Render(title))
	s.WriteRune('\n')
	s.WriteString(st.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(duration))))
	s.WriteRune('\n')
	s.WriteString(st.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(views))))
	s.WriteRune('\n')
	s.WriteString(st.MutedStyle.Render(fmt.Sprintf("🗓  %s", utils.FormatUploadDate(uploadDate, ""))))
	s.WriteRune('\n')
	s.WriteString(st.MutedStyle.Render(fmt.Sprintf("📺 %s", channel)))
	if siteName != "" {
		s.WriteString(st.MutedStyle.Render(fmt.Sprintf(" (%s)", siteName)))
	}
	s.WriteRune('\n')
	if size != "" {
		s.WriteString(st.MutedStyle.Render(fmt.Sprintf("📦 %s", size)))
		s.WriteRune('\n')
	}
	s.WriteString(lipgloss.NewStyle().Foreground(st.TextPrimaryColor).Italic(true).Render(fmt.Sprintf("🔗 %s", url)))
	s.WriteRune('\n')
	return s.String()
}
