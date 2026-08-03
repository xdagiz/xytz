package thumbnail

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/blacktop/go-termimg"

	"github.com/xdagiz/xytz/internal/config"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
)

func TestDetectProtocolFromEnvironment(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "empty environment falls back to halfblocks",
			env:  nil,
			want: "halfblocks",
		},
		{
			name: "kitty via TERM",
			env:  map[string]string{"TERM": "xterm-kitty"},
			want: "kitty",
		},
		{
			name: "kitty via KITTY_WINDOW_ID",
			env:  map[string]string{"KITTY_WINDOW_ID": "1"},
			want: "kitty",
		},
		{
			name: "kitty via GHOSTTY_RESOURCES_DIR",
			env:  map[string]string{"GHOSTTY_RESOURCES_DIR": "/tmp/ghostty"},
			want: "kitty",
		},
		{
			name: "kitty via WEZTERM_EXECUTABLE",
			env:  map[string]string{"WEZTERM_EXECUTABLE": "/usr/bin/wezterm"},
			want: "kitty",
		},
		{
			name: "kitty via TERM_PROGRAM ghostty",
			env:  map[string]string{"TERM_PROGRAM": "ghostty"},
			want: "kitty",
		},
		{
			name: "kitty via TERM_PROGRAM WezTerm",
			env:  map[string]string{"TERM_PROGRAM": "WezTerm"},
			want: "kitty",
		},
		{
			name: "kitty via TERM_PROGRAM rio",
			env:  map[string]string{"TERM_PROGRAM": "rio"},
			want: "kitty",
		},
		{
			name: "iterm2 via TERM_PROGRAM",
			env:  map[string]string{"TERM_PROGRAM": "iTerm.app"},
			want: "iterm2",
		},
		{
			name: "iterm2 via LC_TERMINAL",
			env:  map[string]string{"LC_TERMINAL": "iTerm2"},
			want: "iterm2",
		},
		{
			name: "iterm2 via ITERM_SESSION_ID",
			env:  map[string]string{"ITERM_SESSION_ID": "w0t0p0:ABC123"},
			want: "iterm2",
		},
		{
			name: "sixel via TERM foot",
			env:  map[string]string{"TERM": "foot"},
			want: "sixel",
		},
		{
			name: "sixel via TERM mlterm",
			env:  map[string]string{"TERM": "mlterm"},
			want: "sixel",
		},
		{
			name: "sixel via TERM wezterm",
			env:  map[string]string{"TERM": "wezterm"},
			want: "sixel",
		},
		{
			name: "sixel via xterm with XTERM_VERSION",
			env:  map[string]string{"TERM": "xterm", "XTERM_VERSION": "XTerm(384)"},
			want: "sixel",
		},
		{
			name: "sixel via TERM_PROGRAM mintty",
			env:  map[string]string{"TERM_PROGRAM": "mintty"},
			want: "sixel",
		},
		{
			name: "alacritty is not sixel",
			env:  map[string]string{"TERM": "alacritty", "TERM_PROGRAM": "alacritty"},
			want: "halfblocks",
		},
		{
			name: "apple terminal is halfblocks",
			env:  map[string]string{"TERM_PROGRAM": "Apple_Terminal", "TERM": "xterm-256color"},
			want: "halfblocks",
		},
		{
			name: "generic xterm is halfblocks",
			env:  map[string]string{"TERM": "xterm-256color"},
			want: "halfblocks",
		},
		{
			name: "kitty hint beats sixel hint",
			env:  map[string]string{"TERM_PROGRAM": "WezTerm", "TERM": "wezterm"},
			want: "kitty",
		},
		{
			name: "tmux screen is halfblocks",
			env:  map[string]string{"TERM": "screen-256color", "TMUX": "/tmp/tmux-0"},
			want: "halfblocks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTerminalEnv(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := detectProtocolFromEnvironment(); got != tt.want {
				t.Errorf("detectProtocolFromEnvironment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplyDefaultsAutoUsesEnvironmentDetection(t *testing.T) {
	for _, protocol := range []string{"", "auto"} {
		t.Run("protocol="+strconv.Quote(protocol), func(t *testing.T) {
			clearTerminalEnv(t)
			t.Setenv("TERM", "foot")

			cfg := config.GetDefault()
			cfg.ThumbnailProtocol = protocol
			ctx := appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))

			termimg.ClearFeatureCache()
			_ = NewModel(ctx)
			t.Cleanup(termimg.ClearFeatureCache)

			if got := os.Getenv("TERMIMG_BYPASS_DETECTION"); got != "sixel" {
				t.Errorf("TERMIMG_BYPASS_DETECTION = %q, want %q", got, "sixel")
			}
		})
	}
}

func TestApplyDefaultsExplicitProtocol(t *testing.T) {
	clearTerminalEnv(t)
	t.Setenv("TERM", "foot")

	cfg := config.GetDefault()
	cfg.ThumbnailProtocol = "iterm2"
	ctx := appctx.New(cfg, filepath.Join(t.TempDir(), "config.yaml"), config.ResolveRuntimeOptions(cfg, nil))

	termimg.ClearFeatureCache()
	_ = NewModel(ctx)
	t.Cleanup(termimg.ClearFeatureCache)

	if got := os.Getenv("TERMIMG_BYPASS_DETECTION"); got != "iterm2" {
		t.Errorf("TERMIMG_BYPASS_DETECTION = %q, want %q", got, "iterm2")
	}
}

func clearTerminalEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"TERM", "TERM_PROGRAM", "KITTY_WINDOW_ID", "GHOSTTY_RESOURCES_DIR",
		"WEZTERM_EXECUTABLE", "LC_TERMINAL", "ITERM_SESSION_ID", "XTERM_VERSION",
		"TERM_SESSION_ID", "TMUX",
	} {
		t.Setenv(k, "")
	}
}
