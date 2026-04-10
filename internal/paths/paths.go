package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

func GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".config/xytz"
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "xytz")
		}
		return filepath.Join(homeDir, "AppData", "Roaming", "xytz")

	case "darwin":
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			return filepath.Join(xdgConfigHome, "xytz")
		}
		return filepath.Join(homeDir, "Library", "Application Support", "xytz")

	default:
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome != "" {
			return filepath.Join(xdgConfigHome, "xytz")
		}
		return filepath.Join(homeDir, ".config", "xytz")
	}
}

func GetDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".local/share/xytz"
	}

	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			return filepath.Join(localAppData, "xytz")
		}
		return filepath.Join(homeDir, "AppData", "Local", "xytz")

	case "darwin":
		return filepath.Join(homeDir, "Library", "Application Support", "xytz")

	default:
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome != "" {
			return filepath.Join(xdgDataHome, "xytz")
		}
		return filepath.Join(homeDir, ".local", "share", "xytz")
	}
}

func EnsureDirExists(path string) error {
	return os.MkdirAll(path, 0o755)
}
