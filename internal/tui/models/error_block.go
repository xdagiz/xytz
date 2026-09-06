package models

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/xdagiz/xytz/internal/styles"
)

func ErrorTitleReason(err string) (string, string, bool) {
	if strings.HasPrefix(err, "No results matching") {
		return "No matches found", "Nothing in this scope matched your filter.", true
	}

	if strings.Contains(err, "No results found") {
		return "No results found", "Nothing matched your search.", true
	}

	if strings.Contains(err, "No channels found") {
		return "No channels found", "No channels matched this query.", true
	}

	if strings.Contains(err, "No playlists found") {
		return "No playlists found", "No playlists matched this query.", true
	}

	if strings.Contains(err, "Private video") {
		return "Private video", "It cannot be viewed or downloaded.", false
	}

	if idx := strings.Index(err, ". "); idx > 0 {
		return err[:idx], strings.TrimSpace(err[idx+2:]), false
	}

	lower := strings.ToLower(err)
	if strings.Contains(lower, "channel not found") {
		return "Channel not found", "This channel does not exist or has no public videos.", false
	}

	if strings.Contains(lower, "playlist not found") || strings.Contains(lower, "playlist does not exist") {
		return "Playlist not found", "This playlist does not exist or is unavailable.", false
	}

	if strings.Contains(lower, "private") {
		return "Private playlist", "This playlist is private and cannot be viewed.", false
	}

	if strings.Contains(err, "Internet") || strings.Contains(lower, "connection") {
		return "No internet connection", "Please check your connection.", false
	}

	if strings.Contains(lower, "not found") {
		return "Not found", err, false
	}

	if strings.Contains(lower, "yt-dlp not found") {
		return "yt-dlp missing", err, false
	}

	if strings.Contains(lower, "failed to run yt-dlp") || strings.Contains(lower, "search failed") {
		return "Search failed", err, false
	}

	if strings.Contains(lower, "manager not available") {
		return "Unavailable", err, false
	}

	return "Request failed", err, false
}

func EntityReason(err string) (string, bool, bool) {
	if strings.HasPrefix(err, "No results matching") {
		return "matches nothing in this scope.", true, true
	}

	if strings.Contains(err, "No results found") {
		return "returned no results. Try different wording.", true, true
	}

	if strings.Contains(err, "No channels found") {
		return "matched no channels. Try different wording.", true, true
	}

	if strings.Contains(err, "No playlists found") {
		return "matched no playlists. Try different wording.", true, true
	}

	if strings.Contains(err, "Private video") || strings.Contains(err, "Private content") {
		return "", false, false
	}

	lower := strings.ToLower(err)
	if strings.Contains(lower, "channel not found") {
		return "does not exist or has no public videos.", false, true
	}

	if strings.Contains(lower, "playlist not found") || strings.Contains(lower, "playlist does not exist") {
		return "does not exist or is unavailable.", false, true
	}

	if strings.Contains(lower, "private") {
		return "is private and cannot be viewed.", false, true
	}

	if strings.Contains(err, "has no videos tab") {
		return "has no videos tab.", false, true
	}

	return "", false, false
}

type ErrorContent struct {
	Title        string
	Reason       string
	Entity       string
	EntityReason string
	Help         string
	Warning      bool
}

func DescribeError(err, entity string) ErrorContent {
	title, reason, warn := ErrorTitleReason(err)
	content := ErrorContent{Title: title, Reason: reason, Warning: warn, Help: cookiesHelp(title)}
	if predicate, pwarn, ok := EntityReason(err); ok {
		if entity = strings.TrimSpace(entity); entity != "" {
			content.Entity = entity
			content.EntityReason = predicate
			content.Warning = pwarn
		}
	}
	return content
}

func ErrorBlockView(st styles.Styles, content ErrorContent) string {
	accent := st.StatusErrorColor
	if content.Warning {
		accent = st.StatusWarningColor
	}

	titleStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
	var body string
	if content.Entity != "" && content.EntityReason != "" {
		entityStyle := lipgloss.NewStyle().Foreground(st.TextPrimaryColor).Bold(true)
		messageStyle := lipgloss.NewStyle().Foreground(accent)
		body = titleStyle.Render("⚠ ") + entityStyle.Render(content.Entity) + messageStyle.Render(" "+content.EntityReason)
	} else {
		body = titleStyle.Render("⚠ " + content.Title)
		if content.Reason != "" {
			body += "\n" + st.MutedStyle.Render(content.Reason)
		}
	}
	if content.Help != "" {
		body += "\n" + st.MutedStyle.Render(content.Help)
	}

	bar := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(accent).Padding(0, 0, 0, 2)
	return bar.Render(body)
}

func cookiesHelp(title string) string {
	switch title {
	case "Sign in required", "Private content", "Age restricted":
		return "Add cookies: --cookies-from-browser chrome\nor --cookies cookies.txt, or config cookies_browser / cookies_file"
	default:
		return ""
	}
}
