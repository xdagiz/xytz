package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/paths"
	"github.com/xdagiz/xytz/internal/tui"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/search"
	"github.com/xdagiz/xytz/internal/version"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/spf13/cobra"
)

var (
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
		Run: func(cmd *cobra.Command, args []string) {
			helpFlag, err := cmd.Flags().GetBool("help")
			if err != nil {
				log.Printf("Error getting help flag: %v", err)
				os.Exit(1)
			}

			if helpFlag {
				cmd.Help()
				return
			}

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

func startApp(cmd *cobra.Command) {
	location := config.Location{ConfigFlag: configPath}

	resolved, err := config.ParseConfig(location)
	if err != nil {
		log.Printf("Warning: failed to load config: %v (using defaults)", err)
		resolved.Config = config.GetDefault()
	}

	runtimeCtx := appctx.NewAppContext(resolved.Config)
	runtimeCtx.ConfigLocation = location
	runtimeCtx.ConfigPath = resolved.EffectivePath

	searchLimitSet := cmd.Flags().Changed("number")
	sortBySet := cmd.Flags().Changed("sort-by")
	cookiesBrowserSet := cmd.Flags().Changed("cookies-from-browser")
	cookiesSet := cmd.Flags().Changed("cookies")

	opts := &search.CLIOptions{
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

	zone.NewGlobal()
	defer zone.Close()

	m := tui.NewModel(tui.WithContext(runtimeCtx), tui.WithOptions(opts))
	p := tea.NewProgram(m)
	m.Program = p

	logDir := paths.GetDataDir()
	if err := paths.EnsureDirExists(logDir); err != nil {
		log.Printf("Warning: Could not create log directory: %v", err)
		logDir = "."
	}

	logPath := filepath.Join(logDir, "debug.log")

	logger, err := tea.LogToFile(logPath, "debug")
	if err != nil {
		log.Printf("Warning: Could not create debug log file: %v", err)
	} else {
		defer logger.Close()
	}

	if _, err := p.Run(); err != nil {
		log.Fatalf("unable to run the app: %v", err)
	}

	m.Ctx.CancelManagers()

	saveConfigOptions(m, cmd.Flags().Changed("sort-by"))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cfg := config.GetDefault()
	rootCmd.Version = version.GetVersion()
	rootCmd.AddCommand(completionCmd)

	rootCmd.PersistentFlags().StringVar(
		&configPath,
		"config",
		"",
		"Path to config file (default: $XYTZ_CONFIG or XDG config path)",
	)

	rootCmd.Flags().IntVarP(&searchLimit, "number", "n", cfg.SearchLimit, "Number of search results")

	rootCmd.Flags().StringVarP(&sortBy, "sort-by", "s", cfg.SortByDefault, "Default sort option (relevance, date, views, rating)")

	rootCmd.Flags().BoolP("help", "h", false, "Help for xytz")

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
		log.Printf("Failed to save config on exit: model context is nil")
		return
	}

	cfgPath := m.Ctx.ConfigPath
	if cfgPath == "" {
		log.Printf("Failed to save config on exit: resolved config path is empty")
		return
	}

	cfg := m.Ctx.Config
	if cfg == nil {
		log.Printf("Failed to save config on exit: config is nil")
		return
	}

	diskCfg, err := config.LoadStrictFromPath(cfgPath)
	if err != nil {
		log.Printf("Could not load existing config from %s, using in-memory config: %v", cfgPath, err)
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
		log.Printf("Failed to save config on exit: %v", err)
	}
}
