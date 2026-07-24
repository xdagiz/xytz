package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xdagiz/xytz/internal/paths"
	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName    = "config.yaml"
	ConfigAltFileName = "config.yml"
	ConfigEnvVar      = "XYTZ_CONFIG"
)

type Location struct {
	ConfigFlag string
}

type ResolvedConfig struct {
	Config *Config
	Path   string
}

type Config struct {
	SearchLimit         int    `yaml:"search_limit"`
	DefaultDownloadPath string `yaml:"default_download_path"`
	SpotifyDownloadPath string `yaml:"spotify_download_path"`
	DefaultQuality      string `yaml:"default_quality"`
	SortByDefault       string `yaml:"sort_by_default"`
	EmbedSubtitles      bool   `yaml:"embed_subtitles"`
	EmbedMetadata       bool   `yaml:"embed_metadata"`
	EmbedChapters       bool   `yaml:"embed_chapters"`
	EmbedThumbnail      bool   `yaml:"embed_thumbnail"`
	FFmpegPath          string `yaml:"ffmpeg_path"`
	YTDLPPath           string `yaml:"yt_dlp_path"`
	VideoFormat         string `yaml:"video_format"`
	AudioFormat         string `yaml:"audio_format"`
	CookiesBrowser      string `yaml:"cookies_browser"`
	CookiesFile         string `yaml:"cookies_file"`
	ThumbnailPreview    bool   `yaml:"thumbnail_preview"`
	ThumbnailTimeoutMS  int    `yaml:"thumbnail_timeout_ms"`
	ThumbnailProtocol   string `yaml:"thumbnail_protocol"`
	ListCompactMode     bool   `yaml:"list_compact_mode"`
	Theme               string `yaml:"theme,omitempty"`
	JSRuntime           string `yaml:"js_runtime"`
	JSRuntimePath       string `yaml:"js_runtime_path"`
	Player              string `yaml:"player"`
	BackgroundPlayback  bool   `yaml:"background_playback"`
}

var GetConfigDir = func() string {
	return paths.GetConfigDir()
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), ConfigFileName)
}

func LoadStrictFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := GetDefault()
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(false)
	if err := decoder.Decode(cfg); err != nil {
		return cfg, err
	}

	applyOmittedBooleanDefaults(cfg, data)
	if err := cfg.validate(); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func (c *Config) Save() error {
	return c.SaveToPath(GetConfigPath())
}

func (c *Config) SaveToPath(configPath string) error {
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o600)
}

func (c *Config) applyDefaults() {
	defaults := GetDefault()

	if c.SearchLimit == 0 {
		c.SearchLimit = defaults.SearchLimit
	}

	if c.DefaultDownloadPath == "" {
		c.DefaultDownloadPath = defaults.DefaultDownloadPath
	}

	if c.SpotifyDownloadPath == "" {
		c.SpotifyDownloadPath = defaults.SpotifyDownloadPath
	}

	if c.DefaultQuality == "" {
		c.DefaultQuality = defaults.DefaultQuality
	}

	if c.SortByDefault == "" {
		c.SortByDefault = defaults.SortByDefault
	}

	if c.VideoFormat == "" {
		c.VideoFormat = defaults.VideoFormat
	}

	if c.AudioFormat == "" {
		c.AudioFormat = defaults.AudioFormat
	}

	if c.Player == "" {
		c.Player = defaults.Player
	}

	if c.ThumbnailTimeoutMS == 0 {
		c.ThumbnailTimeoutMS = defaults.ThumbnailTimeoutMS
	}
}

func yamlHasTopLevelKey(data []byte, key string) bool {
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return false
	}
	if len(node.Content) == 0 {
		return false
	}

	mapping := node.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return true
		}
	}

	return false
}

func applyOmittedBooleanDefaults(cfg *Config, data []byte) {
	defaults := GetDefault()

	if !yamlHasTopLevelKey(data, "embed_subtitles") {
		cfg.EmbedSubtitles = defaults.EmbedSubtitles
	}

	if !yamlHasTopLevelKey(data, "embed_metadata") {
		cfg.EmbedMetadata = defaults.EmbedMetadata
	}

	if !yamlHasTopLevelKey(data, "embed_chapters") {
		cfg.EmbedChapters = defaults.EmbedChapters
	}

	if !yamlHasTopLevelKey(data, "embed_thumbnail") {
		cfg.EmbedThumbnail = defaults.EmbedThumbnail
	}

	if !yamlHasTopLevelKey(data, "thumbnail_preview") {
		cfg.ThumbnailPreview = defaults.ThumbnailPreview
	}

	if !yamlHasTopLevelKey(data, "thumbnail_protocol") {
		cfg.ThumbnailProtocol = defaults.ThumbnailProtocol
	}

	if !yamlHasTopLevelKey(data, "list_compact_mode") {
		cfg.ListCompactMode = defaults.ListCompactMode
	}

	if !yamlHasTopLevelKey(data, "background_playback") {
		cfg.BackgroundPlayback = defaults.BackgroundPlayback
	}
}

func (c *Config) GetDefaultFormat() string {
	return ResolveQuality(c.DefaultQuality)
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

func (c *Config) GetSpotifyDownloadPath() string {
	return c.ExpandPath(c.SpotifyDownloadPath)
}

func (c *Config) validate() error {
	if c.SearchLimit <= 0 {
		return fmt.Errorf("search_limit must be greater than 0")
	}

	if c.SortByDefault != "" {
		switch c.SortByDefault {
		case "relevance", "date", "views", "rating":
		default:
			return fmt.Errorf("sort_by_default must be one of relevance, date, views, rating")
		}
	}

	if c.VideoFormat != "" {
		switch strings.ToLower(c.VideoFormat) {
		case "mp4", "mkv", "webm", "mov", "avi", "flv":
		default:
			return fmt.Errorf("video_format must be one of: mp4, mkv, webm, mov, avi, flv")
		}
	}

	if c.AudioFormat != "" {
		switch strings.ToLower(c.AudioFormat) {
		case "mp3", "m4a", "opus", "ogg", "flac", "wav", "aac":
		default:
			return fmt.Errorf("audio_format must be one of: mp3, m4a, opus, ogg, flac, wav, aac")
		}
	}

	if c.ThumbnailTimeoutMS < 250 {
		return fmt.Errorf("thumbnail_timeout_ms must be at least 250")
	}

	if c.ThumbnailProtocol != "" {
		c.ThumbnailProtocol = strings.ToLower(c.ThumbnailProtocol)
		switch c.ThumbnailProtocol {
		case "auto", "kitty", "sixel", "iterm2", "halfblocks":
		default:
			return fmt.Errorf("thumbnail_protocol must be one of: auto, kitty, sixel, iterm2, halfblocks")
		}
	}

	if c.Player != "" {
		c.Player = strings.ToLower(c.Player)
		switch c.Player {
		case "mpv", "ffplay":
		default:
			return fmt.Errorf("player must be one of: mpv, ffplay")
		}
	}

	if c.JSRuntime != "" {
		switch c.JSRuntime {
		case "deno", "node", "bun", "quickjs":
		default:
			return fmt.Errorf("js_runtime must be one of: deno, node, bun, quickjs")
		}
	}

	return nil
}
