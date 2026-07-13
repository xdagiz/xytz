package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func Load(location Location) (ResolvedConfig, error) {
	path, explicit, err := resolveConfigPath(location)
	if err != nil {
		return ResolvedConfig{}, err
	}

	if !fileExists(path) {
		if explicit {
			return ResolvedConfig{}, fmt.Errorf("config file not found: %s", path)
		}

		defaultCfg := GetDefault()
		if err := defaultCfg.SaveToPath(path); err != nil {
			return ResolvedConfig{}, fmt.Errorf("failed creating default config %s: %w", path, err)
		}

		return ResolvedConfig{Config: defaultCfg, Path: path}, nil
	}

	cfg, err := LoadStrictFromPath(path)
	if err != nil {
		return ResolvedConfig{Config: cfg, Path: path}, fmt.Errorf("failed loading config %s: %w", path, err)
	}

	return ResolvedConfig{Config: cfg, Path: path}, nil
}

func resolveConfigPath(location Location) (path string, explicit bool, err error) {
	if location.ConfigFlag != "" {
		return location.ConfigFlag, true, nil
	}

	if envPath := os.Getenv(ConfigEnvVar); envPath != "" {
		return envPath, true, nil
	}

	defaultPath, err := discoverDefaultConfigPath()
	if err != nil {
		return "", false, err
	}

	return defaultPath, false, nil
}

func discoverDefaultConfigPath() (string, error) {
	configDir := GetConfigDir()
	if configDir == "" {
		return "", errors.New("empty config directory")
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", err
	}

	yamlPath := filepath.Join(configDir, ConfigFileName)
	ymlPath := filepath.Join(configDir, ConfigAltFileName)

	if !fileExists(yamlPath) && fileExists(ymlPath) {
		if err := os.Rename(ymlPath, yamlPath); err != nil {
			return "", fmt.Errorf("failed migrating %s to %s: %w", ymlPath, yamlPath, err)
		}
	}

	return yamlPath, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
