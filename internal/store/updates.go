package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	log "charm.land/log/v2"

	"github.com/xdagiz/xytz/internal/fsutil"
	"github.com/xdagiz/xytz/internal/paths"
)

const UpdateCheckFileName = "update_check"

var GetUpdateCheckFilePath = func() string {
	dataDir := paths.GetDataDir()
	if err := paths.EnsureDirExists(dataDir); err != nil {
		log.Warn("could not create data directory", "err", err)
		return UpdateCheckFileName
	}

	return filepath.Join(dataDir, UpdateCheckFileName)
}

func LoadLastUpdateCheck() time.Time {
	path := GetUpdateCheckFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return time.Time{}
		}

		log.Warn("could not read last update check", "err", err)
		return time.Time{}
	}

	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		log.Warn("could not parse last update check", "err", err)
		return time.Time{}
	}

	return t
}

func RecordUpdateCheck() error {
	path := GetUpdateCheckFilePath()
	return fsutil.WriteFileAtomic(path, []byte(time.Now().UTC().Format(time.RFC3339)), 0o600)
}

func ShouldCheckForUpdates(interval time.Duration) bool {
	last := LoadLastUpdateCheck()
	if last.IsZero() {
		return true
	}

	return time.Since(last) >= interval
}
