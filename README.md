# xytz - YouTube from your terminal

A Beautiful TUI for searching and downloading/playing videos from YouTube and other sites.

<a href="https://trendshift.io/repositories/20867?utm_source=trendshift-badge&amp;utm_medium=badge&amp;utm_campaign=badge-trendshift-20867" target="_blank" rel="noopener noreferrer"><img src="https://trendshift.io/api/badge/trendshift/repositories/20867/daily?language=Go" alt="xdagiz%2Fxytz | Trendshift" width="250" height="55"/></a>

[Demo](https://github.com/user-attachments/assets/4e3f98c7-554f-4b9e-adac-52511ae69f32)

<br />

[![Stars](https://img.shields.io/github/stars/xdagiz/xytz?style=flat-square)](https://github.com/xdagiz/xytz/stargazers)
[![Awesome](https://awesome.re/badge-flat.svg)](https://github.com/rothgar/awesome-tuis#multimedia)
[![Downloads](https://img.shields.io/github/downloads/xdagiz/xytz/total?style=flat-square)](https://github.com/xdagiz/xytz/releases)
<br />
[![License: MIT](https://img.shields.io/github/license/xdagiz/xytz?style=flat-square)](https://github.com/xdagiz/xytz/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/xdagiz/xytz?display_name=tag&sort=semver&style=flat-square)](https://github.com/xdagiz/xytz/releases)
[![AUR](https://img.shields.io/aur/version/xytz-bin?style=flat-square&label=AUR-)](https://aur.archlinux.org/packages/xytz-bin)

## Features

- Search YouTube from your terminal
- Download from any `yt-dlp` supported site by pasting a URL
- Browse channel videos with `/channel @username`
- Find channels with `/channels <query>`
- Browse and download playlist videos with `/playlist <id>`
- Pick from available video/audio formats with quality indicators
- Select multiple videos and download them in one batch
- Resume unfinished downloads with `/resume`
- Save videos for later with `/later`
- Play videos with mpv without downloading using `/play <url>`
- Persistent search history (navigate with ↑/↓)
- Inline thumbnail previews in the terminal
- Switch themes at runtime with `/theme <name>`
- Load cookies from your browser or a file for authenticated content
- Fetch and download Spotify tracks or whole albums (multi-disc aware, tagged, with cover art)
- Works on Linux, macOS, and Windows

## Installation

### Requirements

- **yt-dlp**: Core video downloader
  - Installation: https://github.com/yt-dlp/yt-dlp#installation
- **ffmpeg** - Required for full features
  - Installation: https://ffmpeg.org/download.html
- **mpv** (optional) - For playing videos directly without downloading
  - Installation: https://mpv.io/installation/
- **ffplay** (optional, bundled with ffmpeg) - Fallback playback backend
  - Used automatically when mpv is not installed
  - Set `player: ffplay` in config to use it explicitly (see Configuration below)

### Installer Script (Linux/MacOS)

```bash
curl -fsSL https://raw.githubusercontent.com/xdagiz/xytz/main/install.sh | bash
```

### Homebrew (MacOS/Linux)

```bash
brew tap xdagiz/homebrew-tap
brew install --cask xytz
```

### AUR (Arch Linux)

```bash
paru -S xytz-bin # or yay -S xytz-bin
```

### Scoop (Windows)

```pwsh
scoop bucket add xdagiz https://github.com/xdagiz/scoop-bucket.git
scoop install xdagiz/xytz
```

### Go Install

```bash
go install github.com/xdagiz/xytz@latest
```

### Nix (Flakes)

```bash
# Run without installing
nix run github:xdagiz/xytz

# Build in the current repo
nix build

# Enter a development shell (Go, gopls, yt-dlp, ffmpeg, mpv)
nix develop
```

### Build from Source

**Requirements:**

- Go 1.25+

```bash
# Clone the repository
git clone https://github.com/xdagiz/xytz.git
cd xytz

# Build
go build -o xytz .

# Move to your PATH (optional)
sudo mv xytz /usr/local/bin/
```

## Configuration

The config file location varies by operating system:

| OS      | Config Location                                                                                                      |
| ------- | -------------------------------------------------------------------------------------------------------------------- |
| Linux   | `~/.config/xytz/config.yaml` (or `$XDG_CONFIG_HOME/xytz/config.yaml`)                                                |
| macOS   | `~/.config/xytz/config.yaml` if `XDG_CONFIG_HOME` is set, otherwise `~/Library/Application Support/xytz/config.yaml` |
| Windows | `%APPDATA%/xytz/config.yaml`                                                                                         |

On first run, xytz creates the config file with default values if it doesn't exist.

### Default Configuration

```yaml
search_limit: 25 # Number of search results
default_download_path: ~/Videos # Download destination
spotify_download_path: ~/Music # Spotify download destination
default_quality: best # Default format selection (480p, 720p, 1080p, 4k...)
sort_by_default: relevance # Default sort: relevance, date, views, rating
theme: catppuccin-mocha # Preset theme name
video_format: mp4 # The format which videos are downloaded
audio_format: mp3 # The format which audio files are downloaded
embed_subtitles: false # Embed subtitles in downloads
embed_metadata: true # Embed metadata in downloads
embed_chapters: true # Embed chapters in downloads
embed_thumbnail: false # Embed thumbnail in downloads
ffmpeg_path: "" # Custom ffmpeg path (optional)
yt_dlp_path: "" # Custom yt-dlp path (optional)
cookies_browser: "" # Browser for cookies: chrome, firefox, etc (optional)
cookies_file: "" # Path to cookies.txt file for authentication (optional)
thumbnail_preview: true # Enable thumbnail preview in video list
list_compact_mode: false # Compact list rows in video, channel, and format lists
thumbnail_timeout_ms: 2500 # Timeout for fetching thumbnails (ms)
thumbnail_protocol: auto # Thumbnail protocol: kitty, sixel, iterm2, halfblocks, auto (default: auto)
thumbnail_quality: max # Thumbnail quality: max, high, medium, low (optional, default: max)
js_runtime: "" # JS runtime for yt-dlp: deno, node, bun, quickjs (optional)
js_runtime_path: "" # Custom path to JS runtime executable (optional)
player: mpv # Playback backend: mpv, ffplay (falls back to ffplay if mpv isn't installed)
background_playback: false # Keep player running when leaving the player view (background playback)
check_for_updates: true # Check for new releases in the TUI (banner)
```

## Usage

xytz supports command-line arguments for quick access to search, channels, and playlists. Run `xytz --help` to see all available flags.

### Examples

```bash
# Direct video search
xytz -q "golang tutorial"

# Browse a specific channel
xytz -u @username

# Browse a playlist
xytz -p PLplaylistId

# Search for a channel
xytz -c "linux"

# Custom search results and sorting
xytz -n 50 -s date

# Combined: Search with custom options
xytz -q "rust programming" -n 10 -s views

# Check for and apply the latest release, then exit
xytz --update
```

## Contributing

Contributions are welcome. Sync your fork with the upstream repository before submitting pull requests.

### Commit Style

```
<type>(<scope>): <description>

[optional body]
[optional footer]
```

### Pull Request Guidelines

- Keep changes focused and minimal
- Make sure all tests pass before submitting
- Update documentation if needed
- Follow the existing code style

## Troubleshooting

### yt-dlp not found

Make sure yt-dlp is installed and available in your PATH:

```bash
yt-dlp --version
```

If installed in a non-standard location, set `yt_dlp_path` in your config.

### ffmpeg not found

ffmpeg is required for most features. Install it and make sure it's in your PATH, or set `ffmpeg_path` in your config.

### Downloads failing

- Check your internet connection
- Check the video is available in your region
- Check you have enough disk space
- Check the download path is writable
- Make sure you have `yt-dlp` and `ffmpeg` installed

### Not seeing enough formats

Update `yt-dlp` to the latest version.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - Download engine
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling library
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

## Star History

[![RepoStars](https://repostars.dev/api/embed?repo=xdagiz%2Fxytz&theme=noir)](https://repostars.dev/?repos=xdagiz%2Fxytz&theme=noir)

By [xdagiz](https://github.com/xdagiz)
