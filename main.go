package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"zoneout/audio"
	"zoneout/models"
	"zoneout/stats"
	"zoneout/ui"
)

func main() {
	// Create white noise directory if it doesn't exist
	if err := os.MkdirAll("./whitenoise", 0755); err != nil {
		log.Fatalf("Failed to create whitenoise directory: %v", err)
	}

	// Initialize audio player
	audioPlayer, err := audio.NewAudioPlayer("./whitenoise")
	if err != nil {
		log.Fatalf("Failed to initialize audio player: %v", err)
	}
	defer audioPlayer.Stop()

	// Create sounds directory if it doesn't exist
	if err := os.MkdirAll("./sounds", 0755); err != nil {
		log.Fatalf("Failed to create sounds directory: %v", err)
	}

	// Create motd directory if it doesn't exist
	if err := os.MkdirAll("./motd", 0755); err != nil {
		log.Fatalf("Failed to create motd directory: %v", err)
	}

	// Initialize MOTD
	motdManager, err := NewMOTD("./motd")
	if err != nil {
		// MOTD is optional, log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		motdManager = nil
	}

	// Initialize stats
	appStats := stats.NewStats()

	// Initialize Pomodoro state
	pomodoroState := models.NewPomodoro()

	// Set up transition sound effects
	pomodoroState.SetAudioPlayer(audioPlayer, "./sounds/start.mp3", "./sounds/stop.mp3")

	// Create the main model
	mainModel := ui.NewModel(pomodoroState, audioPlayer, appStats, motdManager)

	// Create and run the Bubble Tea program
	p := tea.NewProgram(mainModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
