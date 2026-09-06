package ytdlp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
)

type Cancellable interface {
	ClearAndCheckCanceled() bool
	ResetCanceled()
}

const (
	scannerInitialBuf = 64 * 1024
	scannerMaxToken   = 16 * 1024 * 1024
)

func scanLines(r io.Reader, handle func(line string)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, scannerInitialBuf), scannerMaxToken)
	for scanner.Scan() {
		handle(scanner.Text())
	}
	return scanner.Err()
}

type RunResult[T any] struct {
	Items            []T
	Stdout           []byte
	StderrLines      []string
	SkippedLiveShort int
	Canceled         bool
	Err              error
}

func RunYTDLP[T any](mgr Cancellable, ytDlpPath string, args []string, parse func(string) (T, error)) RunResult[T] {
	mgr.ResetCanceled()

	cmd := exec.Command(ytDlpPath, args...)
	ConfigureProcessGroup(cmd)

	type cmdSetter interface {
		SetCmd(*exec.Cmd)
	}
	if cs, ok := mgr.(cmdSetter); ok {
		cs.SetCmd(cmd)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunResult[T]{Err: fmt.Errorf("failed to get stdout pipe: %w", err)}
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return RunResult[T]{Err: fmt.Errorf("failed to get stderr pipe: %w", err)}
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		return RunResult[T]{Err: fmt.Errorf("failed to start yt-dlp: %w", err)}
	}

	_ = AttachProcessTree(cmd)
	defer ReleaseProcessTree(cmd)

	stopKill := func() {}
	type runController interface {
		MarkStarted()
		RunContext() context.Context
	}
	if rc, ok := mgr.(runController); ok {
		rc.MarkStarted()
		runCtx := rc.RunContext()
		stop := context.AfterFunc(runCtx, func() {
			TerminateProcessAsync(cmd)
		})
		if runCtx.Err() != nil {
			TerminateProcessAsync(cmd)
		}
		stopKill = func() { stop() }
	}
	defer stopKill()

	var (
		items       []T
		stdoutBytes []byte
		stderrLines []string
		skipped     int
		scanErr     error
		stderrWg    sync.WaitGroup
	)

	stderrWg.Go(func() {
		_ = scanLines(stderr, func(line string) {
			stderrLines = append(stderrLines, line)
			log.Debug("yt-dlp stderr", "line", line)
		})
	})

	if parse != nil {
		scanErr = scanLines(stdout, func(line string) {
			line = strings.TrimSpace(line)
			if line == "" {
				return
			}

			item, err := parse(line)
			if err != nil {
				if err == ErrSkippedLiveShort {
					skipped++
					return
				}

				log.Error("failed to parse yt-dlp output", "err", err)
				return
			}

			items = append(items, item)
		})

		if scanErr != nil {
			log.Error("failed to read yt-dlp output", "err", scanErr)
		}
	} else {
		var readErr error
		stdoutBytes, readErr = io.ReadAll(stdout)
		if readErr != nil {
			log.Error("failed to read yt-dlp stdout", "err", readErr)
		}
	}

	stderrWg.Wait()

	var cmdErr error
	if err := cmd.Wait(); err != nil {
		log.Error("yt-dlp command failed", "err", err)
		log.Error("stderr output", "lines", stderrLines)
		cmdErr = err
	}

	if mgr.ClearAndCheckCanceled() {
		return RunResult[T]{Canceled: true}
	}

	if scanErr != nil {
		return RunResult[T]{
			StderrLines:      stderrLines,
			SkippedLiveShort: skipped,
			Err:              fmt.Errorf("failed to read yt-dlp output: %w", scanErr),
		}
	}

	return RunResult[T]{
		Items:            items,
		Stdout:           stdoutBytes,
		StderrLines:      stderrLines,
		SkippedLiveShort: skipped,
		Err:              cmdErr,
	}
}

func AppendCookieArgs(args []string, cfg *config.Config, cookiesBrowser, cookiesFile string) []string {
	if cookiesBrowser == "" {
		cookiesBrowser = cfg.CookiesBrowser
	}

	if cookiesFile == "" {
		cookiesFile = cfg.CookiesFile
	}

	if cookiesBrowser != "" {
		return append(args, "--cookies-from-browser", cookiesBrowser)
	}

	if cookiesFile != "" {
		return append(args, "--cookies", cookiesFile)
	}

	return args
}

func AppendJSRuntimeArgs(args []string, cfg *config.Config) []string {
	if cfg == nil {
		return args
	}

	if cfg.JSRuntime == "" {
		return args
	}

	jsRuntimeArg := cfg.JSRuntime
	if cfg.JSRuntimePath != "" {
		jsRuntimeArg = cfg.JSRuntime + ":" + cfg.JSRuntimePath
	}

	return append(args, "--js-runtimes", jsRuntimeArg)
}

func MapSearchErrorFromStderr(stderrLines []string, searchURL string) string {
	for _, line := range stderrLines {
		if strings.Contains(line, "[Errno 101]") || strings.Contains(line, "[Errno -3]") {
			return "Please Check Your Internet connection"
		}

		if strings.Contains(line, "Private video") {
			return "Private video. It cannot be viewed or downloaded."
		}

		if strings.Contains(line, "Private playlist") || strings.Contains(line, "This playlist is private") || strings.Contains(line, "is private") {
			return "This playlist is private"
		}

		if strings.Contains(line, "HTTP Error 404") || strings.Contains(line, "Requested entity was not found") ||
			strings.Contains(line, "does not exist and the URL redirected") || strings.Contains(line, "Failed to resolve url") {
			if strings.Contains(searchURL, "/playlist?list=") {
				return "Playlist not found"
			}

			if strings.Contains(searchURL, "/@") || strings.Contains(searchURL, "/channel/") || strings.Contains(searchURL, "/c/") {
				return "Channel not found"
			}

			return "Not found"
		}

		if strings.Contains(line, "Playlist does not exist") {
			return "Playlist does not exist"
		}

		if strings.Contains(line, "does not have a") && strings.Contains(line, "tab") {
			return "No videos here. This channel has no videos tab."
		}

		if strings.Contains(line, "Unable to find selected tab") || strings.Contains(line, "Unable to recognize tab page") {
			return "Update needed. YouTube changed its layout, please update yt-dlp and try again."
		}

		if strings.Contains(line, "Incomplete yt initial data") {
			return "Incomplete response. YouTube returned partial data, please try again."
		}

		if strings.Contains(line, "require authentication") {
			return "Private content. Sign in with cookies to access it."
		}

		lower := strings.ToLower(line)

		if strings.Contains(lower, "not available in your region") {
			return "Not available in your region. This content is region restricted."
		}

		if strings.Contains(lower, "unsupported url") {
			return "Site not supported. This URL is not supported by yt-dlp."
		}

		if strings.Contains(lower, "unable to download webpage") || strings.Contains(lower, "unable to download api") {
			return "Network failure. Please check your connection and try again."
		}

		if strings.Contains(line, "HTTP Error 429") || strings.Contains(lower, "too many requests") {
			return "Rate limited. YouTube is limiting requests, please wait and try again."
		}

		if strings.Contains(lower, "sign in to confirm") || strings.Contains(lower, "not a bot") ||
			strings.Contains(lower, "login required") || strings.Contains(lower, "log in to") {
			return "Sign in required. Load cookies from your browser to continue."
		}

		if strings.Contains(lower, "confirm your age") || strings.Contains(lower, "age-restricted") ||
			strings.Contains(lower, "age_verification_required") || strings.Contains(lower, "age_check_required") {
			return "Age restricted. Sign in with an age-verified account to access it."
		}

		if strings.Contains(lower, "available in your country") || strings.Contains(lower, "geo restriction") ||
			strings.Contains(lower, "not available from your location") {
			return "Not available in your country. The uploader restricted this video by region."
		}

		if strings.Contains(lower, "captcha") {
			return "Captcha required. YouTube is asking for verification, please try again later."
		}

		if strings.Contains(lower, "try again later") || strings.Contains(lower, "rate-limit") {
			return "Rate limited. YouTube limited this session for up to an hour, please wait and try again."
		}

		if strings.Contains(lower, "drm") {
			return "DRM protected. This video cannot be downloaded."
		}

		if strings.Contains(lower, "members only") || strings.Contains(lower, "members-only") {
			return "Members-only video. It requires a channel membership."
		}

		if strings.Contains(lower, "purchase") || strings.Contains(lower, "rent this") || strings.Contains(lower, "to rent") {
			return "Paid content. This video must be purchased or rented."
		}

		if strings.Contains(line, "Failed to extract any player response") || strings.Contains(line, "Cannot identify player") ||
			strings.Contains(line, "No player clients") || strings.Contains(line, "Invalid URL") {
			return "Update needed. YouTube changed something, please update yt-dlp and try again."
		}

		if strings.Contains(lower, "video unavailable") || strings.Contains(lower, "not available.") ||
			strings.Contains(lower, "has been removed") || strings.Contains(lower, "no longer available") ||
			strings.Contains(lower, "has been deleted") || strings.Contains(lower, "account has been terminated") {
			return "Video unavailable. It may have been removed or deleted."
		}
	}

	return ""
}

func FriendlyYTDLError(stderrLines []string, searchURL string, err error) string {
	if mapped := MapSearchErrorFromStderr(stderrLines, searchURL); mapped != "" {
		return mapped
	}
	if line := lastYTDLErrorLine(stderrLines); line != "" {
		return line
	}
	if err != nil {
		return "Request failed. Please try again."
	}
	return ""
}

func lastYTDLErrorLine(stderrLines []string) string {
	msg := ""
	for _, line := range stderrLines {
		if idx := strings.Index(line, "ERROR:"); idx >= 0 {
			msg = strings.TrimSpace(line[idx+len("ERROR:"):])
		}
	}
	if msg == "" {
		return ""
	}
	if strings.HasPrefix(msg, "[") {
		if end := strings.Index(msg, "]"); end > 0 {
			msg = strings.TrimSpace(msg[end+1:])
		}
	}
	if idx := strings.Index(msg, ":"); idx > 0 && !strings.Contains(msg[:idx], " ") && !strings.Contains(msg[:idx], "/") {
		msg = strings.TrimSpace(msg[idx+1:])
	}
	runes := []rune(msg)
	if len(runes) > 160 {
		msg = string(runes[:157]) + "..."
	}
	return msg
}
