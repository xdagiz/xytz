package utils

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"charm.land/bubbles/v2/list"
	log "charm.land/log/v2"
	"github.com/xdagiz/xytz/internal/config"
)

type Cancellable interface {
	ClearAndCheckCanceled() bool
}

type RunResult struct {
	Items            []list.Item
	Stdout           []byte
	StderrLines      []string
	SkippedLiveShort int
	Canceled         bool
	Err              error
}

func RunYTDLP(mgr Cancellable, ytDlpPath string, args []string, parse func(string) (list.Item, error)) RunResult {
	cmd := exec.Command(ytDlpPath, args...)

	type cmdSetter interface {
		SetCmd(*exec.Cmd)
	}
	if cs, ok := mgr.(cmdSetter); ok {
		cs.SetCmd(cmd)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return RunResult{Err: fmt.Errorf("failed to get stdout pipe: %w", err)}
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return RunResult{Err: fmt.Errorf("failed to get stderr pipe: %w", err)}
	}
	defer stderr.Close()

	if err := cmd.Start(); err != nil {
		return RunResult{Err: fmt.Errorf("failed to start yt-dlp: %w", err)}
	}

	var (
		items       []list.Item
		stdoutBytes []byte
		stderrLines []string
		skipped     int
		stderrWg    sync.WaitGroup
	)

	stderrWg.Go(func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrLines = append(stderrLines, line)
			log.Debug("yt-dlp stderr", "line", line)
		}
	})

	if parse != nil {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			item, err := parse(line)
			if err != nil {
				if err == ErrSkippedLiveShort {
					skipped++
					continue
				}

				log.Error("failed to parse yt-dlp output", "err", err)
				continue
			}

			items = append(items, item)
		}

		if err := scanner.Err(); err != nil {
			log.Error("scanner error", "err", err)
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
		return RunResult{Canceled: true}
	}

	return RunResult{
		Items:            items,
		Stdout:           stdoutBytes,
		StderrLines:      stderrLines,
		SkippedLiveShort: skipped,
		Err:              cmdErr,
	}
}

func AppendCookieArgs(args []string, cfg *config.Config, cookiesBrowser, cookiesFile string) []string {
	if cfg == nil {
		cfg = config.GetDefault()
	}

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

			return "Channel not found"
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
