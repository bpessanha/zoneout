package main

import (
	"embed"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"zoneout/audio"
	"zoneout/models"
	"zoneout/stats"
	"zoneout/ui"
)

//go:embed sounds/* motd/* whitenoise/*
var assetsFS embed.FS

func main() {
	// Create white noise directory if it doesn't exist
	if err := os.MkdirAll("./whitenoise", 0755); err != nil {
		log.Fatalf("Failed to create whitenoise directory: %v", err)
	}

	// Initialize audio player with embedded whitenoise + user files
	audioPlayer, err := audio.NewAudioPlayerWithEmbed("./whitenoise", assetsFS)
	if err != nil {
		log.Fatalf("Failed to initialize audio player: %v", err)
	}
	defer audioPlayer.Stop()

	// Create motd directory if it doesn't exist (for user-provided messages)
	if err := os.MkdirAll("./motd", 0755); err != nil {
		log.Fatalf("Failed to create motd directory: %v", err)
	}

	// Initialize MOTD from embedded + user files
	motdManager, err := NewMOTDWithEmbed("./motd", assetsFS)
	if err != nil {
		// MOTD is optional, log but don't fail
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		motdManager = nil
	}

	// Initialize stats
	appStats := stats.NewStats()

	// Initialize Pomodoro state
	pomodoroState := models.NewPomodoro()

	// Set up transition sound effects from embedded assets
	if err := pomodoroState.SetAudioPlayerWithEmbed(audioPlayer, assetsFS); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load sound effects: %v\n", err)
	}
	defer pomodoroState.Cleanup()

	// Create the main model
	mainModel := ui.NewModel(pomodoroState, audioPlayer, appStats, motdManager)

	// Create and run the Bubble Tea program
	p := tea.NewProgram(mainModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
