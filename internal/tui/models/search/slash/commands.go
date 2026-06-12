package slash

import (
	"strings"

	"github.com/sahilm/fuzzy"
	"github.com/xdagiz/xytz/internal/tui/theme"
)

type Command struct {
	Name        string
	Description string
	Usage       string
	HasArg      bool
}

var AllCommands = []Command{
	{
		Name:        "channel",
		Description: "List videos from a specific channel using @username",
		Usage:       "/channel <username>",
		HasArg:      true,
	},
	{
		Name:        "channels",
		Description: "Search for YouTube channels",
		Usage:       "/channels <query>",
		HasArg:      true,
	},
	{
		Name:        "playlist",
		Description: "List videos of a playlist",
		Usage:       "/playlist <id>",
		HasArg:      true,
	},
	{
		Name:        "playlists",
		Description: "Search for YouTube playlists",
		Usage:       "/playlists <query>",
		HasArg:      true,
	},
	{
		Name:        "play",
		Description: "Play a video with url",
		Usage:       "/play <url>",
		HasArg:      true,
	},
	{
		Name:        "resume",
		Description: "Resume unfinished download",
		Usage:       "/resume",
		HasArg:      false,
	},
	{
		Name:        "later",
		Description: "Browse and download videos saved for later",
		Usage:       "/later",
		HasArg:      false,
	},
	{
		Name:        "theme",
		Description: "Switch to a preset theme",
		Usage:       "/theme <name>",
		HasArg:      true,
	},
	{
		Name:        "help",
		Description: "Show available commands",
		Usage:       "/help",
		HasArg:      false,
	},
}

type MatchResult struct {
	Command Command
	Score   float64
	Matched bool
}

func ParseCommand(input string) (cmd string, args string, isSlash bool) {
	input = strings.TrimSpace(input)
	if !strings.HasPrefix(input, "/") {
		return "", "", false
	}

	rest := strings.TrimPrefix(input, "/")

	spaceIdx := strings.Index(rest, " ")
	if spaceIdx == -1 {
		return rest, "", true
	}

	cmd = rest[:spaceIdx]
	args = strings.TrimSpace(rest[spaceIdx:])
	return cmd, args, true
}

func FuzzyMatch(query string) []MatchResult {
	query = strings.TrimPrefix(query, "/")

	if query == "" {
		results := make([]MatchResult, len(AllCommands))
		for i, cmd := range AllCommands {
			results[i] = MatchResult{Command: cmd, Score: 0, Matched: true}
		}
		return results
	}

	patterns := make([]string, len(AllCommands))
	for i, cmd := range AllCommands {
		patterns[i] = cmd.Name
	}

	matches := fuzzy.Find(query, patterns)

	var results []MatchResult
	for _, match := range matches {
		if match.Score > 0 {
			cmd := AllCommands[match.Index]
			results = append(results, MatchResult{
				Command: cmd,
				Score:   float64(match.Score),
				Matched: true,
			})
		}
	}

	return results
}

type ThemeMatchResult struct {
	Name  string
	Score float64
}

func FuzzyMatchThemes(query string) []ThemeMatchResult {
	knownThemes := theme.KnownThemes()

	if query == "" {
		results := make([]ThemeMatchResult, len(knownThemes))
		for i, t := range knownThemes {
			results[i] = ThemeMatchResult{Name: t, Score: 1000}
		}
		return results
	}

	matches := fuzzy.Find(query, knownThemes)

	var results []ThemeMatchResult
	for _, match := range matches {
		if match.Score > 0 {
			results = append(results, ThemeMatchResult{
				Name:  knownThemes[match.Index],
				Score: float64(match.Score),
			})
		}
	}

	if len(results) == 0 {
		for i, t := range knownThemes {
			if strings.Contains(strings.ToLower(t), strings.ToLower(query)) {
				results = append(results, ThemeMatchResult{
					Name:  knownThemes[i],
					Score: 500,
				})
			}
		}
	}

	return results
}
