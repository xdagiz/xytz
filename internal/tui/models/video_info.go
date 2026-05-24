package models

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/styles"
	"github.com/xdagiz/xytz/internal/utils"
)

func VideoInfoView(title, channel, url, upload_date string, duration, views float64, size, siteName string) string {
	s := strings.Builder{}
	s.WriteString(styles.SectionHeaderStyle.Render(title))
	s.WriteRune('\n')
	s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("⏱  %s", utils.FormatDuration(duration))))
	s.WriteRune('\n')
	s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("👁  %s views", utils.FormatNumber(views))))
	s.WriteRune('\n')
	s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("🗓  %s", utils.FormatUploadDate(upload_date, ""))))
	s.WriteRune('\n')
	s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📺 %s", channel)))
	if siteName != "" {
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf(" (%s)", siteName)))
	}
	s.WriteRune('\n')
	if size != "" {
		s.WriteString(styles.MutedStyle.Render(fmt.Sprintf("📦 %s", size)))
		s.WriteRune('\n')
	}
	s.WriteString(lipgloss.NewStyle().Foreground(styles.TextPrimaryColor).Italic(true).Render(fmt.Sprintf("🔗 %s", url)))
	s.WriteRune('\n')
	return s.String()
}
