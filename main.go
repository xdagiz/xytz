package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/xdagiz/xytz/internal/app"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func main() {
	zone.NewGlobal()
	defer zone.Close()

	m := app.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.Program = p

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get home directory: %v", err)
		homeDir = "."
	}

	logDir := filepath.Join(homeDir, ".local", "share", "xytz")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Warning: Could not create log directory: %v", err)
		logDir = "."
	}

	logPath := filepath.Join(logDir, "debug.log")

	logger, err := tea.LogToFile(logPath, "debug")
	if err != nil {
		log.Printf("Warning: Could not create debug log file: %v", err)
	} else {
		defer logger.Close()
	}

	if _, err := p.Run(); err != nil {
		log.Fatal("unable to run the app")
		os.Exit(1)
	}
}
