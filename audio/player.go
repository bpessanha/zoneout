package audio

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

type AudioPlayer struct {
	whitenoiseDir string
	availableMP3s []string
	currentMP3    string
	isPlaying     bool
	isPaused      bool
	loopEnabled   bool
	currentCmd    *exec.Cmd
	mu            sync.Mutex
}

func NewAudioPlayer(whitenoiseDir string) (*AudioPlayer, error) {
	ap := &AudioPlayer{
		whitenoiseDir: whitenoiseDir,
		loopEnabled:   true,
	}

	// Scan for MP3 files
	if err := ap.ScanWhitenoiseDirectory(); err != nil {
		return nil, fmt.Errorf("failed to scan whitenoise directory: %w", err)
	}

	return ap, nil
}

func (ap *AudioPlayer) ScanWhitenoiseDirectory() error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.availableMP3s = []string{}

	entries, err := os.ReadDir(ap.whitenoiseDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if strings.HasSuffix(strings.ToLower(name), ".mp3") {
				fullPath := filepath.Join(ap.whitenoiseDir, name)
				ap.availableMP3s = append(ap.availableMP3s, fullPath)
			}
		}
	}

	return nil
}

func (ap *AudioPlayer) GetAvailableMP3s() []string {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Return a copy
	result := make([]string, len(ap.availableMP3s))
	copy(result, ap.availableMP3s)
	return result
}

func (ap *AudioPlayer) PlayMP3(filePath string) error {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	// Stop current playback if any
	if ap.currentCmd != nil && ap.currentCmd.Process != nil {
		ap.currentCmd.Process.Kill()
		ap.currentCmd = nil
	}

	// Use appropriate audio player based on OS
	var cmd *exec.Cmd
	if runtime.GOOS == "darwin" {
		// macOS - use afplay with looping
		cmd = exec.Command("afplay", filePath)
	} else if runtime.GOOS == "linux" {
		// Linux - try ffplay first with looping
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loop", "0", filePath)
	} else {
		// Windows and others
		cmd = exec.Command("ffplay", "-nodisp", "-autoexit", "-loop", "0", filePath)
	}

	if err := cmd.Start(); err != nil {
		// Try alternative player if first one fails
		if runtime.GOOS != "darwin" {
			cmd = exec.Command("play", "-q", filePath)
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("failed to play MP3: no suitable audio player found")
			}
		} else {
			return fmt.Errorf("failed to play MP3 with afplay: %v", err)
		}
	}

	ap.currentCmd = cmd
	ap.currentMP3 = filePath
	ap.isPlaying = true

	// Run the command in a background goroutine to monitor it
	go func() {
		ap.currentCmd.Wait()
		ap.mu.Lock()
		defer ap.mu.Unlock()
		if ap.currentCmd != nil && ap.isPlaying {
			ap.isPlaying = false
		}
	}()

	return nil
}

func (ap *AudioPlayer) SwitchMP3(filePath string) error {
	return ap.PlayMP3(filePath)
}

func (ap *AudioPlayer) Pause() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentCmd != nil && ap.currentCmd.Process != nil {
		ap.currentCmd.Process.Signal(os.Interrupt)
		ap.isPaused = true
		ap.isPlaying = false
	}
}

func (ap *AudioPlayer) Resume() {
	ap.mu.Lock()
	currentMP3 := ap.currentMP3
	isPaused := ap.isPaused
	ap.mu.Unlock()

	// For system audio players, we'll need to restart the file
	if isPaused && currentMP3 != "" {
		ap.PlayMP3(currentMP3)
	}
}

func (ap *AudioPlayer) Stop() {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	if ap.currentCmd != nil && ap.currentCmd.Process != nil {
		ap.currentCmd.Process.Kill()
		ap.currentCmd = nil
	}
	ap.isPlaying = false
	ap.isPaused = false
}

func (ap *AudioPlayer) IsPlaying() bool {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	return ap.isPlaying
}

func (ap *AudioPlayer) GetCurrentMP3() string {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	return ap.currentMP3
}

func (ap *AudioPlayer) SetLoop(enabled bool) {
	ap.mu.Lock()
	defer ap.mu.Unlock()

	ap.loopEnabled = enabled
}

// PlaySoundEffect plays a short MP3 sound effect file without interrupting current playback
func (ap *AudioPlayer) PlaySoundEffect(filePath string) {
	// Play sound effect in a background goroutine to avoid blocking
	go func() {
		var cmd *exec.Cmd
		if runtime.GOOS == "darwin" {
			// macOS - use afplay
			cmd = exec.Command("afplay", filePath)
		} else if runtime.GOOS == "linux" {
			// Linux - use ffplay
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", filePath)
		} else {
			// Windows - use ffplay
			cmd = exec.Command("ffplay", "-nodisp", "-autoexit", filePath)
		}

		if err := cmd.Start(); err != nil {
			// Silently fail if sound effect can't be played
			return
		}

		// Wait for the sound to finish playing
		cmd.Wait()
	}()
}

func (ap *AudioPlayer) Close() error {
	ap.Stop()
	return nil
}
