package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func FetchLatestVersion() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/xdagiz/xytz/releases/latest", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "xytz-tui")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release ReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	return strings.TrimPrefix(release.TagName, "v"), nil
}
