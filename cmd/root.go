package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	log "charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/fang"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/spf13/cobra"

	"github.com/xdagiz/xytz/internal/config"
	"github.com/xdagiz/xytz/internal/paths"
	"github.com/xdagiz/xytz/internal/tui"
	appctx "github.com/xdagiz/xytz/internal/tui/context"
	"github.com/xdagiz/xytz/internal/tui/models/thumbnail"
	"github.com/xdagiz/xytz/internal/types"
	"github.com/xdagiz/xytz/internal/updater"
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
	updateFlag         bool

	newUpdater = func() updater.UpdateService {
		return updater.New()
	}

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
xytz --config ~/.config/xytz/config.yaml
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if updateFlag {
				if err := validateUpdateExclusive(cmd); err != nil {
					return err
				}
				os.Exit(runUpdate())
			}
			return startApp(cmd)
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

func startApp(cmd *cobra.Command) error {
	if cmd.Flags().Changed("number") && searchLimit <= 0 {
		return fmt.Errorf("--number must be greater than 0")
	}
	if cmd.Flags().Changed("sort-by") && !types.IsValidSortBy(sortBy) {
		return fmt.Errorf("--sort-by must be one of relevance, date, views, rating")
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
			log.SetColorProfile(colorprofile.NoTTY)
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
		if os.Getenv("LOG_LEVEL") != "" {
			setLogLevel()
		} else {
			log.SetLevel(log.FatalLevel)
		}
	}

	location := config.Location{ConfigFlag: configPath}
	resolved, err := config.Load(location)
	if err != nil {
		if resolved.Config == nil {
			return fmt.Errorf("unable to load config: %w", err)
		}
		fmt.Fprintf(os.Stderr, "warning: config has errors, some values may fall back to defaults: %v\n", err)
	}

	thumbnail.ConfigureTermImgProtocol(resolved.Config.ThumbnailPreview, resolved.Config.ThumbnailProtocol)

	opts := buildCLIOptions(cmd)
	if warning, err := validateSearchConflicts(opts); err != nil {
		return err
	} else if warning != "" {
		fmt.Fprintln(os.Stderr, "warning: "+warning)
	}
	runtime := config.ResolveRuntimeOptions(resolved.Config, opts)
	appCtx := appctx.New(resolved.Config, resolved.Path, runtime)

	zone.NewGlobal()
	defer zone.Close()

	m := tui.NewModel(appCtx, tui.WithOptions(opts))
	p := tea.NewProgram(m)
	m.Program = p

	if _, err := p.Run(); err != nil {
		log.Fatal("unable to run the app", "err", err)
	}

	m.Ctx.CancelManagers()

	saveConfigOptions(m, cmd.Flags().Changed("sort-by"))
	return nil
}

func normalizeSwitchArgs(args []string) []string {
	out := make([]string, 0, len(args))
	rest := args
	skipNext := false

	for len(rest) > 0 {
		arg := rest[0]
		rest = rest[1:]

		if skipNext {
			out = append(out, arg)
			skipNext = false
			continue
		}

		if arg == "--" {
			out = append(out, rest...)
			break
		}

		if takesNextValue(arg) {
			out = append(out, arg)
			skipNext = true
			continue
		}

		target, attached, ok := splitModeSwitch(arg)
		if !ok {
			out = append(out, arg)
			continue
		}

		if attached != "" {
			out = append(out, target+"="+attached)
			continue
		}

		if len(rest) > 0 && (rest[0] == "-" || !strings.HasPrefix(rest[0], "-")) {
			out = append(out, target+"="+rest[0])
			rest = rest[1:]
			continue
		}

		out = append(out, target)
	}

	return out
}

func takesNextValue(arg string) bool {
	switch arg {
	case "-q", "--query", "-u", "--channel", "-p", "--playlist", "-n", "--number", "-s", "--sort-by", "--cookies", "--cookies-from-browser", "--config":
		return true
	default:
		return false
	}
}

func splitModeSwitch(arg string) (string, string, bool) {
	if arg == "--channels" || arg == "-c" {
		return "--channels", "", true
	}

	if arg == "--playlists" || arg == "-l" {
		return "--playlists", "", true
	}

	if rest, ok := strings.CutPrefix(arg, "--channels="); ok {
		return "--channels", rest, true
	}

	if rest, ok := strings.CutPrefix(arg, "--playlists="); ok {
		return "--playlists", rest, true
	}

	if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
		switch arg[1] {
		case 'c':
			if arg[2] == '=' {
				return "--channels", arg[3:], true
			}
			return "--channels", arg[2:], true
		case 'l':
			if arg[2] == '=' {
				return "--playlists", arg[3:], true
			}
			return "--playlists", arg[2:], true
		}
	}

	return "", "", false
}

func Execute() {
	rootCmd.SetArgs(normalizeSwitchArgs(os.Args[1:]))
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

	rootCmd.Flags().BoolVar(
		&updateFlag,
		"update",
		false,
		"Check for and apply updates, then exit (no TUI)",
	)

	rootCmd.Flags().IntVarP(&searchLimit, "number", "n", cfg.SearchLimit, "Number of search results")

	rootCmd.Flags().StringVarP(&sortBy, "sort-by", "s", cfg.SortByDefault, "Default sort option (relevance, date, views, rating)")

	rootCmd.Flags().StringVarP(&query, "query", "q", "", "Direct search with a query")
	rootCmd.Flags().StringVarP(&channel, "channel", "u", "", "Load videos for a channel")
	rootCmd.Flags().StringVarP(&channels, "channels", "c", "", "Direct channel search")
	rootCmd.Flags().StringVarP(&playlists, "playlists", "l", "", "Direct playlist search")
	rootCmd.Flags().StringVarP(&playlist, "playlist", "p", "", "Load videos for a playlist")

	rootCmd.Flags().Lookup("channels").NoOptDefVal = " "
	rootCmd.Flags().Lookup("playlists").NoOptDefVal = " "

	rootCmd.Flags().StringVar(&cookiesFromBrowser, "cookies-from-browser", cfg.CookiesBrowser, "The name of the browser to load cookies from")
	rootCmd.Flags().StringVar(&cookies, "cookies", cfg.CookiesFile, "Netscape formatted file to read cookies from")
}

func validateSearchConflicts(opts *config.CLIOptions) (string, error) {
	if opts == nil {
		return "", nil
	}

	channel := strings.TrimSpace(opts.Channel)
	playlist := strings.TrimSpace(opts.Playlist)
	channelsVal := strings.TrimSpace(opts.ChannelQuery)
	playlistsVal := strings.TrimSpace(opts.PlaylistsQuery)
	query := strings.TrimSpace(opts.Query)

	claimants := []string{}
	if channelsVal != "" {
		claimants = append(claimants, fmt.Sprintf("--channels %q", channelsVal))
	} else if opts.ChannelQuerySet && query != "" {
		claimants = append(claimants, "--channels with --query")
	}
	if playlistsVal != "" {
		claimants = append(claimants, fmt.Sprintf("--playlists %q", playlistsVal))
	} else if opts.PlaylistsQuerySet && query != "" {
		claimants = append(claimants, "--playlists with --query")
	}

	scopeCount := 0
	scopeDesc := ""
	if channel != "" {
		scopeCount++
		scopeDesc = fmt.Sprintf("--channel %q", channel)
	}

	if playlist != "" {
		scopeCount++
		scopeDesc = fmt.Sprintf("--playlist %q", playlist)
	}

	if scopeCount > 1 {
		return "", fmt.Errorf("cannot load --channel %q and --playlist %q at once: pick one", channel, playlist)
	}

	if scopeCount == 1 && len(claimants) > 0 {
		return "", fmt.Errorf("cannot combine %s with %s: load the scope or search, not both", scopeDesc, claimants[0])
	}

	if len(claimants) > 1 {
		return "", fmt.Errorf("cannot combine %s with %s: run one search at a time", claimants[0], claimants[1])
	}

	if opts.SortBySet {
		switch config.ResolveSearch(opts).Mode {
		case config.SearchModeChannelSearch, config.SearchModePlaylistSearch, config.SearchModeChannelVideos, config.SearchModePlaylistVideos:
			return "--sort-by only affects video search and is ignored here", nil
		}
	}

	return "", nil
}

func validateUpdateExclusive(cmd *cobra.Command) error {
	if cmd == nil || !cmd.Flags().Changed("update") {
		return nil
	}

	used := []string{}
	for _, name := range []string{"query", "channel", "channels", "playlists", "playlist", "number", "sort-by"} {
		if cmd.Flags().Changed(name) {
			used = append(used, "--"+name)
		}
	}

	if len(used) == 0 {
		return nil
	}

	return fmt.Errorf("cannot combine --update with %s: run --update alone", strings.Join(used, ", "))
}

func buildCLIOptions(cmd *cobra.Command) *config.CLIOptions {
	return &config.CLIOptions{
		SearchLimit:        searchLimit,
		SearchLimitSet:     cmd.Flags().Changed("number"),
		SortBy:             sortBy,
		SortBySet:          cmd.Flags().Changed("sort-by"),
		Query:              query,
		QuerySet:           cmd.Flags().Changed("query"),
		ChannelQuery:       channels,
		ChannelQuerySet:    cmd.Flags().Changed("channels"),
		Channel:            channel,
		ChannelSet:         cmd.Flags().Changed("channel"),
		PlaylistsQuery:     playlists,
		PlaylistsQuerySet:  cmd.Flags().Changed("playlists"),
		Playlist:           playlist,
		PlaylistSet:        cmd.Flags().Changed("playlist"),
		CookiesFromBrowser: cookiesFromBrowser,
		CookiesBrowserSet:  cmd.Flags().Changed("cookies-from-browser"),
		Cookies:            cookies,
		CookiesSet:         cmd.Flags().Changed("cookies"),
	}
}

func saveConfigOptions(m *tui.Model, sortBySet bool) {
	if m == nil || m.Ctx == nil {
		fmt.Fprintln(os.Stderr, "warning: failed to save config on exit: model context is nil")
		return
	}

	cfgPath := m.Ctx.ConfigPath
	if cfgPath == "" {
		fmt.Fprintln(os.Stderr, "warning: failed to save config on exit: resolved config path is empty")
		return
	}

	cfg := m.Ctx.Config
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "warning: failed to save config on exit: config is nil")
		return
	}

	diskCfg, err := config.LoadStrictFromPath(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			diskCfg = cfg
		} else {
			fmt.Fprintf(os.Stderr, "warning: skipping config save, config file has errors: %v\n", err)
			return
		}
	}

	for _, opt := range m.Search.DownloadOptions {
		switch opt.ConfigField {
		case "EmbedSubtitles":
			diskCfg.EmbedSubtitles = opt.Enabled
		case "EmbedMetadata":
			diskCfg.EmbedMetadata = opt.Enabled
		case "EmbedChapters":
			diskCfg.EmbedChapters = opt.Enabled
		case "EmbedThumbnail":
			diskCfg.EmbedThumbnail = opt.Enabled
		}
	}

	if !sortBySet {
		diskCfg.SortByDefault = string(m.Search.SortBy)
	}

	if err := diskCfg.SaveToPath(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to save config on exit: %v\n", err)
	}
}

const (
	exitCodeOK    = 0
	exitCodeError = 1
	exitCodeHint  = 2
)

var (
	errUpdateCheckTimeout = errors.New("update check timed out")
	errInstallTimeout     = errors.New("update install timed out")
)

func runUpdate() int {
	current := version.NormalizeVersion(version.GetVersion())
	if version.IsDev() {
		fmt.Println("xytz: dev build, updates unavailable")
		return exitCodeOK
	}

	svc := newUpdater()

	if ok, hint := svc.CanSelfUpdate(); !ok {
		fmt.Fprintln(os.Stderr, hint)
		return exitCodeHint
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), 15*time.Second, errUpdateCheckTimeout)
	defer cancel()

	rel, found, err := svc.DetectLatest(ctx)
	if err != nil {
		if errors.Is(context.Cause(ctx), errUpdateCheckTimeout) {
			fmt.Fprintln(os.Stderr, "xytz: update check timed out")
			return exitCodeError
		}

		fmt.Fprintf(os.Stderr, "xytz: update check failed: %v\n", err)
		return exitCodeError
	}

	if !found || version.CompareVersions(rel.Version, current) <= 0 {
		fmt.Printf("xytz is up to date (v%s)\n", current)
		return exitCodeOK
	}

	fmt.Printf("Updating xytz v%s -> %s\n", current, updater.VersionDisplay(rel.Version))
	ictx, icancel := context.WithTimeoutCause(context.Background(), 10*time.Minute, errInstallTimeout)
	defer icancel()

	if err := svc.Install(ictx, rel); err != nil {
		if errors.Is(context.Cause(ictx), errInstallTimeout) {
			fmt.Fprintln(os.Stderr, "xytz: update install timed out")
			return exitCodeError
		}

		fmt.Fprintf(os.Stderr, "xytz: update failed: %v\n", err)
		return exitCodeError
	}

	fmt.Printf("xytz updated to %s\n", updater.VersionDisplay(rel.Version))
	return exitCodeOK
}
