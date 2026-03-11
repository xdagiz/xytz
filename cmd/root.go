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

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	searchLimit        int
	sortBy             string
	query              string
	channel            string
	channels           string
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
)

func startApp(cmd *cobra.Command) {
	location := config.Location{ConfigFlag: configPath}
	cfgPath := config.ResolveConfigPath(location)
	cfg, err := config.LoadWithLocation(location)
	if err != nil {
		log.Printf("Warning: Could not load config for startup, using defaults: %v", err)
		cfg = config.GetDefault()
	}
	applyConfigDefaults(cmd.Flags(), cfg)
	runtimeCtx := appctx.NewAppContext(cfg)

	opts := &search.CLIOptions{
		SearchLimit:        searchLimit,
		SortBy:             sortBy,
		Query:              query,
		ChannelQuery:       channels,
		Channel:            channel,
		Playlist:           playlist,
		CookiesFromBrowser: cookiesFromBrowser,
		Cookies:            cookies,
	}

	zone.NewGlobal()
	defer zone.Close()

	m := tui.NewModelWithContext(runtimeCtx, opts)
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
		os.Exit(1)
	}

	m.Ctx.CancelManagers()

	saveConfigOptions(m, cfg, cfgPath, cmd.Flags())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cfg := config.GetDefault()

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
	rootCmd.Flags().StringVarP(&playlist, "playlist", "p", "", "Load videos for a playlist")

	rootCmd.Flags().StringVarP(&cookiesFromBrowser, "cookies-from-browser", "", cfg.CookiesBrowser, "The name of the browser to load cookies from")
	rootCmd.Flags().StringVarP(&cookies, "cookies", "", cfg.CookiesFile, "Netscape formatted file to read cookies from")
}

func applyConfigDefaults(flags *pflag.FlagSet, cfg *config.Config) {
	if cfg == nil {
		return
	}
	if !flags.Changed("number") {
		searchLimit = cfg.SearchLimit
	}
	if !flags.Changed("sort-by") {
		sortBy = cfg.SortByDefault
	}
	if !flags.Changed("cookies-from-browser") {
		cookiesFromBrowser = cfg.CookiesBrowser
	}
	if !flags.Changed("cookies") {
		cookies = cfg.CookiesFile
	}
}

func saveConfigOptions(m *tui.Model, cfg *config.Config, cfgPath string, flags *pflag.FlagSet) {
	if cfg == nil {
		log.Printf("Failed to save config on exit: config is nil")
		return
	}

	for _, opt := range m.Search.DownloadOptions {
		switch opt.ConfigField {
		case "EmbedSubtitles":
			cfg.EmbedSubtitles = opt.Enabled
		case "EmbedMetadata":
			cfg.EmbedMetadata = opt.Enabled
		case "EmbedChapters":
			cfg.EmbedChapters = opt.Enabled
		}
	}

	if flags == nil || !flags.Changed("sort-by") {
		cfg.SortByDefault = string(m.Search.SortBy)
	}

	if err := cfg.SaveToPath(cfgPath); err != nil {
		log.Printf("Failed to save config on exit: %v", err)
	}
}
