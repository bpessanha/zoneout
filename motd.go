package main

import (
	"embed"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MOTDManager interface {
	GetMessage() string
	NeedsRefresh() bool
	Refresh()
}

type MOTD struct {
	currentMessage string
	loadedAt       time.Time
	messages       []string
}

func NewMOTD(motdDir string) (*MOTD, error) {
	m := &MOTD{
		loadedAt: time.Now(),
	}

	// Load messages from directory
	if err := m.loadMessages(motdDir); err != nil {
		return nil, err
	}

	// Select initial random message
	m.selectRandomMessage()

	return m, nil
}

func NewMOTDWithEmbed(motdDir string, assetsFS embed.FS) (*MOTD, error) {
	m := &MOTD{
		loadedAt: time.Now(),
	}

	// Load messages from embedded assets first
	if err := m.loadMessagesFromEmbed(assetsFS); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load embedded messages: %v\n", err)
		// Not fatal - try user files
	}

	// Load additional messages from user directory
	if err := m.loadMessages(motdDir); err != nil {
		// If no embedded messages and no user messages, return error
		if len(m.messages) == 0 {
			return nil, fmt.Errorf("no messages found (embedded or user): %w", err)
		}
		// Otherwise, just use embedded messages
	}

	// If still no messages, error
	if len(m.messages) == 0 {
		return nil, fmt.Errorf("no valid messages found")
	}

	// Select initial random message
	m.selectRandomMessage()

	return m, nil
}

func (m *MOTD) loadMessages(motdDir string) error {
	// Don't clear messages if we're combining with embedded messages
	// Check if directory exists
	if _, err := os.Stat(motdDir); os.IsNotExist(err) {
		return fmt.Errorf("motd directory does not exist: %s", motdDir)
	}

	entries, err := os.ReadDir(motdDir)
	if err != nil {
		return fmt.Errorf("failed to read motd directory: %w", err)
	}

	foundAny := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			filePath := filepath.Join(motdDir, entry.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}

			// Split by lines and add non-empty lines
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					m.messages = append(m.messages, trimmed)
					foundAny = true
				}
			}
		}
	}

	if !foundAny && len(m.messages) == 0 {
		return fmt.Errorf("no valid messages found in motd directory")
	}

	return nil
}

func (m *MOTD) loadMessagesFromEmbed(assetsFS embed.FS) error {
	// Initialize messages if not already done
	if m.messages == nil {
		m.messages = []string{}
	}

	// Read all files from embedded motd directory
	entries, err := fs.ReadDir(assetsFS, "motd")
	if err != nil {
		return fmt.Errorf("failed to read embedded motd directory: %w", err)
	}

	foundAny := false
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".txt") {
			content, err := fs.ReadFile(assetsFS, filepath.Join("motd", entry.Name()))
			if err != nil {
				continue
			}

			// Split by lines and add non-empty lines
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					m.messages = append(m.messages, trimmed)
					foundAny = true
				}
			}
		}
	}

	if !foundAny {
		return fmt.Errorf("no valid messages found in embedded motd")
	}

	return nil
}

func (m *MOTD) selectRandomMessage() {
	if len(m.messages) > 0 {
		m.currentMessage = m.messages[rand.Intn(len(m.messages))]
	} else {
		m.currentMessage = ""
	}
}

func (m *MOTD) GetMessage() string {
	if m == nil {
		return ""
	}
	return m.currentMessage
}

func (m *MOTD) NeedsRefresh() bool {
	if m == nil {
		return false
	}
	return time.Since(m.loadedAt) > 24*time.Hour
}

func (m *MOTD) Refresh() {
	if m == nil {
		return
	}
	m.selectRandomMessage()
	m.loadedAt = time.Now()
}
