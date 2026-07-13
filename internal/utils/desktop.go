package utils

import (
	"os/exec"
	"runtime"

	log "charm.land/log/v2"
	"github.com/atotto/clipboard"
)

func OpenURL(url string) {
	go func() {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "darwin":
			cmd = exec.Command("open", url)
		default:
			cmd = exec.Command("xdg-open", url)
		}

		if err := cmd.Start(); err != nil {
			log.Warn("failed to open URL", "err", err)
			return
		}
		if err := cmd.Wait(); err != nil {
			log.Warn("failed to open URL", "err", err)
		}
	}()
}

func CopyToClipboard(text string) error {
	return clipboard.WriteAll(text)
}
