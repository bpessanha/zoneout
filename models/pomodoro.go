package models

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"time"
	"zoneout/audio"
)

type Mode int

const (
	ModeIdle Mode = iota
	ModeFocus
	ModeBreak
)

type Session struct {
	FocusDuration time.Duration
	BreakDuration time.Duration
	TotalSessions int
}

type Pomodoro struct {
	CurrentMode        Mode
	Session            Session
	CurrentSession     int
	RemainingTime      time.Duration
	TotalTime          time.Duration
	IsRunning          bool
	IsPaused           bool
	LastTickTime       time.Time
	CompletedSessions  int
	audioPlayer        *audio.AudioPlayer
	startSoundPath     string
	stopSoundPath      string
	startSoundTempPath string // For cleanup
	stopSoundTempPath  string // For cleanup
}

func NewPomodoro() *Pomodoro {
	return &Pomodoro{
		CurrentMode: ModeIdle,
		Session: Session{
			FocusDuration: 25 * time.Minute,
			BreakDuration: 5 * time.Minute,
			TotalSessions: 3,
		},
		CurrentSession:    0,
		RemainingTime:     25 * time.Minute,
		TotalTime:         25 * time.Minute,
		IsRunning:         false,
		IsPaused:          false,
		CompletedSessions: 0,
		startSoundPath:    "",
		stopSoundPath:     "",
	}
}

// SetAudioPlayer sets the audio player for playing transition sounds
func (p *Pomodoro) SetAudioPlayer(player *audio.AudioPlayer, startSoundPath, stopSoundPath string) {
	p.audioPlayer = player
	p.startSoundPath = startSoundPath
	p.stopSoundPath = stopSoundPath
}

// SetAudioPlayerWithEmbed sets the audio player and extracts embedded sounds to temp files
func (p *Pomodoro) SetAudioPlayerWithEmbed(player *audio.AudioPlayer, assetsFS embed.FS) error {
	p.audioPlayer = player

	// Extract start.mp3
	startPath, err := extractSoundToTemp(assetsFS, "sounds/start.mp3")
	if err != nil {
		return fmt.Errorf("failed to extract start sound: %w", err)
	}
	p.startSoundPath = startPath
	p.startSoundTempPath = startPath

	// Extract stop.mp3
	stopPath, err := extractSoundToTemp(assetsFS, "sounds/stop.mp3")
	if err != nil {
		// Clean up start sound if stop sound fails
		os.Remove(startPath)
		return fmt.Errorf("failed to extract stop sound: %w", err)
	}
	p.stopSoundPath = stopPath
	p.stopSoundTempPath = stopPath

	return nil
}

// Cleanup removes temporary sound files
func (p *Pomodoro) Cleanup() {
	if p.startSoundTempPath != "" {
		os.Remove(p.startSoundTempPath)
		p.startSoundTempPath = ""
	}
	if p.stopSoundTempPath != "" {
		os.Remove(p.stopSoundTempPath)
		p.stopSoundTempPath = ""
	}
}

// PlayStartSound plays the start sound effect
func (p *Pomodoro) PlayStartSound() {
	if p.audioPlayer != nil && p.startSoundPath != "" {
		p.audioPlayer.PlaySoundEffect(p.startSoundPath)
	}
}

// PlayStopSound plays the stop sound effect
func (p *Pomodoro) PlayStopSound() {
	if p.audioPlayer != nil && p.stopSoundPath != "" {
		p.audioPlayer.PlaySoundEffect(p.stopSoundPath)
	}
}

// Helper function to extract embedded sound to temp file
func extractSoundToTemp(assetsFS embed.FS, path string) (string, error) {
	data, err := fs.ReadFile(assetsFS, path)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded file: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "zoneout-sound-*.mp3")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	tmpFile.Close()
	return tmpFile.Name(), nil
}

func (p *Pomodoro) Start() {
	if p.CurrentMode == ModeIdle {
		p.CurrentMode = ModeFocus
		p.CurrentSession = 1
		p.RemainingTime = p.Session.FocusDuration
		p.TotalTime = p.Session.FocusDuration
		p.PlayStartSound() // Mode changed to FOCUS
	}
	p.IsRunning = true
	p.IsPaused = false
	p.LastTickTime = time.Now()
	if p.CurrentMode != ModeIdle {
		p.PlayStartSound() // Status changed to Running
	}
}

func (p *Pomodoro) Pause() {
	p.IsRunning = false
	p.IsPaused = true
	p.PlayStopSound() // Status changed to Paused
}

func (p *Pomodoro) Resume() {
	if p.IsPaused {
		p.IsRunning = true
		p.IsPaused = false
		p.LastTickTime = time.Now()
		p.PlayStartSound() // Status changed to Running
	}
}

func (p *Pomodoro) Stop() {
	p.IsRunning = false
	p.IsPaused = false
	p.CurrentMode = ModeIdle
	p.CurrentSession = 0
	p.RemainingTime = p.Session.FocusDuration
	p.TotalTime = p.Session.FocusDuration
	p.CompletedSessions = 0
}

func (p *Pomodoro) Tick(delta time.Duration) bool {
	if !p.IsRunning {
		return false
	}

	p.RemainingTime -= delta
	if p.RemainingTime <= 0 {
		return p.NextPhase()
	}
	return false
}

func (p *Pomodoro) NextPhase() bool {
	// Check if we're done with the current phase
	if p.CurrentMode == ModeFocus {
		p.CompletedSessions++
		// Switch to break
		p.CurrentMode = ModeBreak
		p.RemainingTime = p.Session.BreakDuration
		p.TotalTime = p.Session.BreakDuration
		p.PlayStopSound() // Mode changed to BREAK
		return false
	} else if p.CurrentMode == ModeBreak {
		// Check if we've completed all sessions after the break
		if p.CurrentSession >= p.Session.TotalSessions {
			p.Stop()
			return true // All done
		}
		// Switch to next focus session
		p.CurrentSession++
		p.CurrentMode = ModeFocus
		p.RemainingTime = p.Session.FocusDuration
		p.TotalTime = p.Session.FocusDuration
		p.PlayStartSound() // Mode changed to FOCUS
		return false
	}
	return false
}

func (p *Pomodoro) GetModeString() string {
	switch p.CurrentMode {
	case ModeFocus:
		return "FOCUS"
	case ModeBreak:
		return "BREAK"
	default:
		return "IDLE"
	}
}

func (p *Pomodoro) FormatTime() string {
	minutes := int(p.RemainingTime.Minutes())
	seconds := int(p.RemainingTime.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
