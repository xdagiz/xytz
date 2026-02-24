package slash

import (
	"strings"

	"github.com/sahilm/fuzzy"
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
		Name:        "playlist",
		Description: "List videos of a playlist",
		Usage:       "/playlist <id>",
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
