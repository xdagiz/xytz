package main

import (
	"log"
	"os"
	"xytz/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := app.NewModel()
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	logger, _ := tea.LogToFile("debug.log", "debug")
	defer logger.Close()

	if _, err := p.Run(); err != nil {
		log.Fatal("unable to run the app")
		os.Exit(1)
	}
}
