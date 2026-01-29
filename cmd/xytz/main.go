package main

import (
	"github.com/xdagiz/xytz/internal/app"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	zone "github.com/lrstanley/bubblezone"
)

func main() {
	zone.NewGlobal()
	defer zone.Close()

	m := app.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.Program = p

	logger, _ := tea.LogToFile("debug.log", "debug")
	defer logger.Close()

	if _, err := p.Run(); err != nil {
		log.Fatal("unable to run the app")
		os.Exit(1)
	}
}
