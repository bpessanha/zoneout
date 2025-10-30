package main

import (
	"fmt"
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

func (m *MOTD) loadMessages(motdDir string) error {
	m.messages = []string{}

	// Check if directory exists
	if _, err := os.Stat(motdDir); os.IsNotExist(err) {
		return fmt.Errorf("motd directory does not exist: %s", motdDir)
	}

	entries, err := os.ReadDir(motdDir)
	if err != nil {
		return fmt.Errorf("failed to read motd directory: %w", err)
	}

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
				}
			}
		}
	}

	if len(m.messages) == 0 {
		return fmt.Errorf("no valid messages found in motd directory")
	}

	return nil
}

func (m *MOTD) selectRandomMessage() {
	if len(m.messages) > 0 {
		m.currentMessage = m.messages[rand.Intn(len(m.messages))]
	}
}

func (m *MOTD) GetMessage() string {
	return m.currentMessage
}

func (m *MOTD) NeedsRefresh() bool {
	return time.Since(m.loadedAt) > 24*time.Hour
}

func (m *MOTD) Refresh() {
	m.selectRandomMessage()
	m.loadedAt = time.Now()
}
