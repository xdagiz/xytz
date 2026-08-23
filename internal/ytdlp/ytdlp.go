package ytdlp

import (
	"bufio"
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

		if strings.Contains(line, "HTTP Error 404") || strings.Contains(line, "Requested entity was not found") {
			if strings.Contains(searchURL, "/playlist?list=") {
				return "Playlist not found"
			}

			if strings.Contains(searchURL, "/@") || strings.Contains(searchURL, "/channel/") || strings.Contains(searchURL, "/c/") {
				return "Channel not found"
			}

			return "Not found"
		}

		if strings.Contains(line, "Private playlist") || strings.Contains(line, "This playlist is private") {
			return "This playlist is private"
		}

		if strings.Contains(line, "Playlist does not exist") {
			return "Playlist does not exist"
		}
	}

	return ""
}
