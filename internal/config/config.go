package config

import (
	"os"
	"path/filepath"
	"strings"
	"log"

	"gopkg.in/yaml.v3"
)

const ConfigFileName = "config.yaml"

type Config struct {
	SearchLimit         int    `yaml:"search_limit"`
	DefaultDownloadPath string `yaml:"default_download_path"`
	DefaultFormat       string `yaml:"default_format"`
	SortByDefault       string `yaml:"sort_by_default"`
	EmbedSubtitles      bool   `yaml:"embed_subtitles"`
	EmbedMetadata       bool   `yaml:"embed_metadata"`
	EmbedChapters       bool   `yaml:"embed_chapters"`
	FFmpegPath          string `yaml:"ffmpeg_path"`
	YTDLPPath           string `yaml:"yt_dlp_path"`
}

func GetConfigDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".config/xytz"
	}

	return filepath.Join(homeDir, ".config", "xytz")
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func Load() (*Config, error) {
	configPath := GetConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := GetDefault()
		if err := defaultCfg.Save(); err != nil {
			return defaultCfg, err
		}

		return defaultCfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Warning: Could not read config file %s: %v, using defaults", configPath, err)
		return GetDefault(), nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Printf("Warning: Could not parse config file %s: %v, using defaults", configPath, err)
		return GetDefault(), nil
	}

	cfg.applyDefaults()

	return &cfg, nil
}

func (c *Config) Save() error {
	configPath := GetConfigPath()

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) applyDefaults() {
	defaults := GetDefault()
	if c.SearchLimit == 0 {
		c.SearchLimit = defaults.SearchLimit
	}

	if c.DefaultDownloadPath == "" {
		c.DefaultDownloadPath = defaults.DefaultDownloadPath
	}

	if c.DefaultFormat == "" {
		c.DefaultFormat = defaults.DefaultFormat
	}

	if c.SortByDefault == "" {
		c.SortByDefault = defaults.SortByDefault
	}

	if !c.EmbedSubtitles && defaults.EmbedSubtitles {
		c.EmbedSubtitles = defaults.EmbedSubtitles
	}

	if !c.EmbedMetadata && defaults.EmbedMetadata {
		c.EmbedMetadata = defaults.EmbedMetadata
	}

	if !c.EmbedChapters && defaults.EmbedChapters {
		c.EmbedChapters = defaults.EmbedChapters
	}
}

func (c *Config) ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, path[2:])
		}
	}

	return path
}

func (c *Config) GetDownloadPath() string {
	return c.ExpandPath(c.DefaultDownloadPath)
}
