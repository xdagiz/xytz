# xytz - YouTube from your terminal

A beautiful TUI app for searching and downloading YouTube videos, built with Go and [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8?style=flat&logo=go)

## ✨ Features

- **Interactive Search** - Search YouTube videos directly from your terminal
- **Channel Browsing** - View all videos from a specific channel with `/channel @username`
- **Playlist Support** - Browse and download videos from playlists with `/playlist <id>`
- **Format Selection** - Choose from available video/audio formats with quality indicators
- **Download Management** - Real-time progress tracking with speed and ETA
- **Resume Downloads** - Resume unfinished downloads with `/resume`
- **Search History** - Persistent search history for quick access
- **Keyboard Navigation** - Vim-style keybindings and intuitive shortcuts
- **Cross-Platform** - Works on Linux and Windows (MacOS not tested)

**Requirements:**

- **yt-dlp** - Required for video search and download functionality
  - Installation: https://github.com/yt-dlp/yt-dlp#installation
- **ffmpeg** (optional) - Required for adding subtitles, and more
  - Installation: https://ffmpeg.org/download.html

## Installation

### 1. Download from Releases (Recommended)

The easiest way to install xytz is to download a pre-built binary from the [Releases](https://github.com/xdagiz/xytz/releases) page.

```bash
curl -LO https://github.com/xdagiz/xytz/releases/latest/download/xytz-v0.7.0-linux-amd64.tar.gz
tar -xzf xytz-v0.7.0-linux-amd64.tar.gz
sudo mv xytz /usr/local/bin/
```

### 2. Using Go Install

If you have Go installed, you can install directly:

```bash
go install github.com/xdagiz/xytz@latest
```

### 3. Build from Source

**Requirements:**

- **Go 1.25+** - For building from source

```bash
# Clone the repository
git clone https://github.com/xdagiz/xytz.git
cd xytz

# Build
go build -o xytz .

# Move to your PATH (optional)
sudo mv xytz /usr/local/bin/
```

### Using Build Script

```bash
# Build for Linux x64
./scripts/build.sh linux/amd64

# Build for Windows x64
./scripts/build.sh windows/amd64

# Build for all platforms
./scripts/build.sh

# Binaries will be in the dist/ directory
```

## Getting Started

Launch xytz by running:

```bash
xytz
```

### Basic Workflow

1. **Search** - Type your query and press `Enter` to search
2. **Select** - Use `↑/↓` or `j/k` to navigate results, `Enter` to select
3. **Choose Format** - Select your preferred video/audio format
4. **Download** - The download starts automatically

## Configuration

xytz uses a YAML configuration file located at `~/.config/xytz/config.yaml`.

### Default Configuration

```yaml
search_limit: 25 # Number of search results
default_download_path: ~/Videos # Download destination
default_format: bestvideo+bestaudio/best # Default format selection
sort_by_default: relevance # Default sort: relevance, date, views, rating
embed_subtitles: false # Embed subtitles in downloads
embed_metadata: true # Embed metadata in downloads
embed_chapters: true # Embed chapters in downloads
ffmpeg_path: "" # Custom ffmpeg path (optional)
yt_dlp_path: "" # Custom yt-dlp path (optional)
```

The configuration file is created automatically on first run with sensible defaults.

## File Structure

```
xytz/
├── main.go             # Application entry point
├── internal/           # Internal packages
│   ├── app/            # Main application logic (Bubble Tea model)
│   ├── config/         # Configuration management
│   ├── models/         # UI component models
│   ├── slash/          # Slash command definitions
│   ├── styles/         # Lipgloss styling
│   ├── types/          # Type definitions and enums
│   ├── utils/          # Utility functions
│   └── version/        # Version information
├── scripts/            # Build scripts
│   └── build.sh
├── go.mod              # Go module definition
├── go.sum              # Go dependencies
└── README.md           # Readme
```

## Contributing

Contributions are welcome! Please follow these steps:

1. **Fork** the repository
2. **Create** a feature branch: `git checkout -b feature/amazing-feature`
3. **Commit** your changes: `git commit -m 'Add amazing feature'`
4. **Push** to the branch: `git push origin feature/amazing-feature`
5. **Open** a Pull Request

### Development Setup

```bash
# Clone your fork
git clone https://github.com/yourusername/xytz.git
cd xytz

# Install dependencies
go mod tidy

# Run in development mode
go run .
```

## Troubleshooting

### yt-dlp not found

Ensure yt-dlp is installed and available in your PATH:

```bash
yt-dlp --version
```

If installed in a non-standard location, set `yt_dlp_path` in your config.

### ffmpeg features unavailable

Features like embedding subtitles require ffmpeg. Install it and ensure it's in your PATH, or set `ffmpeg_path` in your config.

### Downloads failing

- Check your internet connection
- Verify the video is available in your region
- Ensure you have sufficient disk space
- Check the download path is writable

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [yt-dlp](https://github.com/yt-dlp/yt-dlp) - Video download engine
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling library
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components

---

By [xdagiz](https://github.com/xdagiz)
