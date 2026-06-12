package cmd

import (
	"context"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"
	"github.com/charmbracelet/fang"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/spf13/cobra"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/paths"
	"github.com/xdagiz/xytz/internal/tui"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/version"
)

var (
	debug              bool
	searchLimit        int
	sortBy             string
	query              string
	channel            string
	channels           string
	playlists          string
	playlist           string
	cookiesFromBrowser string
	cookies            string
	configPath         string

	rootCmd = &cobra.Command{
		Use:   "xytz",
		Short: "xytz - YouTube from your terminal",
		Long: `xytz is a TUI YouTube app that allows you to search,
browse, and download videos directly from your terminal.`,
		Example: `
# Launch the TUI
xytz

# Search directly from the CLI
xytz --query "never gonna give you up"

# Load a channel's videos
xytz --channel "UCXuqSBlHAE6Xw-yeJA0Tunw"

# Customize search results
xytz --number 20 --sort-by date

# Use a different config file
xytz --config ~/.config/xytz/config.yml
`,
		Run: func(cmd *cobra.Command, args []string) {
			startApp(cmd)
		},
	}

	completionCmd = &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args: cobra.MatchAll(
			cobra.ExactArgs(1),
			cobra.OnlyValidArgs,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "powershell":
				return rootCmd.GenPowerShellCompletion(os.Stdout)
			default:
				return nil
			}
		},
	}
)

func setLogLevel() {
	switch os.Getenv("LOG_LEVEL") {
	case "debug", "":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	}
}

func startApp(cmd *cobra.Command) {
	location := config.Location{ConfigFlag: configPath}

	resolved, err := config.ParseConfig(location)
	if err != nil {
		log.Warn("failed to load config, using defaults", "err", err)
		resolved.Config = config.GetDefault()
	}

	runtimeCtx := appctx.NewAppContext(resolved.Config)
	runtimeCtx.ConfigLocation = location
	runtimeCtx.ConfigPath = resolved.EffectivePath

	searchLimitSet := cmd.Flags().Changed("number")
	sortBySet := cmd.Flags().Changed("sort-by")
	cookiesBrowserSet := cmd.Flags().Changed("cookies-from-browser")
	cookiesSet := cmd.Flags().Changed("cookies")

	opts := &config.CLIOptions{
		SearchLimit:        searchLimit,
		SearchLimitSet:     searchLimitSet,
		SortBy:             sortBy,
		SortBySet:          sortBySet,
		Query:              query,
		ChannelQuery:       channels,
		Channel:            channel,
		PlaylistsQuery:     playlists,
		Playlist:           playlist,
		CookiesFromBrowser: cookiesFromBrowser,
		CookiesBrowserSet:  cookiesBrowserSet,
		Cookies:            cookies,
		CookiesSet:         cookiesSet,
	}

	if debug {
		logDir := paths.GetDataDir()
		if err := paths.EnsureDirExists(logDir); err != nil {
			log.Warn("could not create log directory", "err", err)
			logDir = "."
		}

		f, err := os.OpenFile(filepath.Join(logDir, "debug.log"),
			os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o666)
		if err == nil {
			log.SetOutput(f)
			log.SetTimeFormat(time.Kitchen)
			log.SetReportCaller(true)
			setLogLevel()
			log.Info("logging to debug.log")
			defer f.Close()
		} else {
			log.Warn("could not create debug.log, falling back to stderr", "err", err)
		}
	} else {
		log.SetOutput(os.Stderr)
		log.SetLevel(log.FatalLevel)
	}

	zone.NewGlobal()
	defer zone.Close()

	m := tui.NewModel(tui.WithContext(runtimeCtx), tui.WithOptions(opts))
	p := tea.NewProgram(m)
	m.Program = p

	if _, err := p.Run(); err != nil {
		log.Fatal("unable to run the app", "err", err)
	}

	m.Ctx.CancelManagers()

	saveConfigOptions(m, cmd.Flags().Changed("sort-by"))
}

func Execute() {
	if err := fang.Execute(
		context.Background(),
		rootCmd,
		fang.WithVersion(version.GetVersion()),
		fang.WithColorSchemeFunc(fangColorScheme),
		fang.WithoutCompletions(),
		fang.WithoutManpage(),
	); err != nil {
		os.Exit(1)
	}
}

func init() {
	cfg := config.GetDefault()
	rootCmd.AddCommand(completionCmd)

	rootCmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		"",
		"Path to config file (default: $XYTZ_CONFIG or XDG config path)",
	)

	rootCmd.PersistentFlags().BoolVar(
		&debug,
		"debug",
		false,
		"write debug output to debug.log",
	)

	rootCmd.Flags().IntVarP(&searchLimit, "number", "n", cfg.SearchLimit, "Number of search results")

	rootCmd.Flags().StringVarP(&sortBy, "sort-by", "s", cfg.SortByDefault, "Default sort option (relevance, date, views, rating)")

	rootCmd.Flags().StringVarP(&query, "query", "q", "", "Direct search with a query")
	rootCmd.Flags().StringVarP(&channel, "channel", "u", "", "Load videos for a channel")
	rootCmd.Flags().StringVarP(&channels, "channels", "c", "", "Direct channel search")
	rootCmd.Flags().StringVarP(&playlists, "playlists", "l", "", "Direct playlist search")
	rootCmd.Flags().StringVarP(&playlist, "playlist", "p", "", "Load videos for a playlist")

	rootCmd.Flags().StringVarP(&cookiesFromBrowser, "cookies-from-browser", "", cfg.CookiesBrowser, "The name of the browser to load cookies from")
	rootCmd.Flags().StringVarP(&cookies, "cookies", "", cfg.CookiesFile, "Netscape formatted file to read cookies from")
}

func saveConfigOptions(m *tui.Model, sortBySet bool) {
	if m == nil || m.Ctx == nil {
		log.Warn("failed to save config on exit: model context is nil")
		return
	}

	cfgPath := m.Ctx.ConfigPath
	if cfgPath == "" {
		log.Warn("failed to save config on exit: resolved config path is empty")
		return
	}

	cfg := m.Ctx.Config
	if cfg == nil {
		log.Warn("failed to save config on exit: config is nil")
		return
	}

	diskCfg, err := config.LoadStrictFromPath(cfgPath)
	if err != nil {
		log.Warn("could not load existing config, using in-memory config", "path", cfgPath, "err", err)
		diskCfg = cfg
	}

	for _, opt := range m.Search.DownloadOptions {
		switch opt.ConfigField {
		case "EmbedSubtitles":
			diskCfg.EmbedSubtitles = opt.Enabled
		case "EmbedMetadata":
			diskCfg.EmbedMetadata = opt.Enabled
		case "EmbedChapters":
			diskCfg.EmbedChapters = opt.Enabled
		}
	}

	if !sortBySet {
		diskCfg.SortByDefault = string(m.Search.SortBy)
	}

	if err := diskCfg.SaveToPath(cfgPath); err != nil {
		log.Warn("failed to save config on exit", "err", err)
	}
}
