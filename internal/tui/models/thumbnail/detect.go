package thumbnail

import (
	"os"
	"strings"
)

func detectProtocolFromEnvironment() string {
	termProgram := os.Getenv("TERM_PROGRAM")
	termName := strings.ToLower(os.Getenv("TERM"))

	if termProgram == "Apple_Terminal" || termProgram == "Terminal" {
		return "halfblocks"
	}

	switch {
	case os.Getenv("KITTY_WINDOW_ID") != "":
		return "kitty"
	case strings.Contains(termName, "kitty"):
		return "kitty"
	case os.Getenv("GHOSTTY_RESOURCES_DIR") != "":
		return "kitty"
	case os.Getenv("WEZTERM_EXECUTABLE") != "":
		return "kitty"
	}

	switch termProgram {
	case "ghostty", "WezTerm", "rio":
		return "kitty"
	}

	if termProgram == "iTerm.app" ||
		strings.Contains(strings.ToLower(os.Getenv("LC_TERMINAL")), "iterm") ||
		os.Getenv("ITERM_SESSION_ID") != "" {
		return "iterm2"
	}

	switch {
	case strings.Contains(termName, "sixel"):
		return "sixel"
	case strings.Contains(termName, "mlterm"):
		return "sixel"
	case strings.Contains(termName, "foot"):
		return "sixel"
	case strings.Contains(termName, "wezterm"):
		return "sixel"
	case strings.Contains(termName, "xterm") && os.Getenv("XTERM_VERSION") != "":
		return "sixel"
	}

	if termProgram == "mintty" {
		return "sixel"
	}

	return "halfblocks"
}
