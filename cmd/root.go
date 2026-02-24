package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/paths"
	"github.com/xdagiz/xytz/internal/tui"
	"github.com/xdagiz/xytz/internal/tui/models/search"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
	"github.com/spf13/cobra"
)

var (
	searchLimit        int
	sortBy             string
	query              string
	channel            string
	playlist           string
	cookiesFromBrowser string
	cookies            string

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

			startApp()
		},
	}
)

func startApp() {
	opts := &search.CLIOptions{
		SearchLimit:        searchLimit,
		SortBy:             sortBy,
		Query:              query,
		Channel:            channel,
		Playlist:           playlist,
		CookiesFromBrowser: cookiesFromBrowser,
		Cookies:            cookies,
	}

	zone.NewGlobal()
	defer zone.Close()

	m := tui.NewModelWithOptions(opts)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
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
		log.Fatal("unable to run the app")
		os.Exit(1)
	}

	m.SearchManager.Cancel()
	m.FormatsManager.Cancel()
	m.DownloadManager.Cancel()

	saveConfigOptions(m)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Could not load config, using defaults: %v", err)
		cfg = config.GetDefault()
	}

	rootCmd.Flags().IntVarP(&searchLimit, "number", "n", cfg.SearchLimit, "Number of search results")

	rootCmd.Flags().StringVarP(&sortBy, "sort-by", "s", cfg.SortByDefault, "Default sort option (relevance, date, views, rating)")

	rootCmd.Flags().BoolP("help", "h", false, "Help for xytz")

	rootCmd.Flags().StringVarP(&query, "query", "q", "", "Direct search with a query")
	rootCmd.Flags().StringVarP(&channel, "channel", "c", "", "Direct channel search")
	rootCmd.Flags().StringVarP(&playlist, "playlist", "p", "", "Direct playlist search")

	rootCmd.Flags().StringVarP(&cookiesFromBrowser, "cookies-from-browser", "", cfg.CookiesBrowser, "The name of the browser to load cookies from")
	rootCmd.Flags().StringVarP(&cookies, "cookies", "", cfg.CookiesFile, "Netscape formatted file to read cookies from")
}

func saveConfigOptions(m *tui.Model) {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load config on exit: %v", err)
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

	cfg.SortByDefault = string(m.Search.SortBy)

	if err := cfg.Save(); err != nil {
		log.Printf("Failed to save config on exit: %v", err)
	}
}
