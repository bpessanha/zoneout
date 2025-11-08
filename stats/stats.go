package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Stats struct {
	TotalSessions      int   `json:"total_sessions"`
	TodaySessions      int   `json:"today_sessions"`
	LastSessionDate    string `json:"last_session_date"`
	TotalFocusMinutes  int   `json:"total_focus_minutes"`
	statsFile          string
	mu                 sync.Mutex
}

func NewStats() *Stats {
	s := &Stats{}
	s.Load()
	return s
}

func NewStatsWithPath(configDir string) *Stats {
	s := &Stats{
		statsFile: filepath.Join(configDir, ".zoneout_stats"),
	}
	s.Load()
	return s
}

func (s *Stats) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use default if not set
	statsPath := s.statsFile
	if statsPath == "" {
		statsPath = ".zoneout_stats"
	}

	// Try to read stats file
	data, err := os.ReadFile(statsPath)
	if err != nil {
		// File doesn't exist yet, initialize with zeros
		s.TotalSessions = 0
		s.TodaySessions = 0
		s.LastSessionDate = ""
		s.TotalFocusMinutes = 0
		return nil
	}

	// Parse JSON
	if err := json.Unmarshal(data, s); err != nil {
		return fmt.Errorf("failed to parse stats file: %w", err)
	}

	// Check if it's a new day
	today := time.Now().Format("2006-01-02")
	if s.LastSessionDate != today {
		s.TodaySessions = 0
		s.LastSessionDate = today
	}

	return nil
}

func (s *Stats) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use default if not set
	statsPath := s.statsFile
	if statsPath == "" {
		statsPath = ".zoneout_stats"
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := os.WriteFile(statsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

func (s *Stats) AddSession(focusMinutes int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use default if not set
	statsPath := s.statsFile
	if statsPath == "" {
		statsPath = ".zoneout_stats"
	}

	today := time.Now().Format("2006-01-02")

	// Reset today's count if it's a new day
	if s.LastSessionDate != today {
		s.TodaySessions = 0
	}

	s.TotalSessions++
	s.TodaySessions++
	s.TotalFocusMinutes += focusMinutes
	s.LastSessionDate = today

	// Save to file
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := os.WriteFile(statsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

func (s *Stats) GetTotalSessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TotalSessions
}

func (s *Stats) GetTodaySessions() int {
	// Reload from disk to ensure we have the latest value
	s.Load()

	// After Load(), the mutex is already released, so we can safely lock again
	s.mu.Lock()
	todaySessions := s.TodaySessions
	lastDate := s.LastSessionDate
	s.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if lastDate != today {
		// It's a new day, reset today's count
		s.mu.Lock()
		s.TodaySessions = 0
		s.LastSessionDate = today
		s.mu.Unlock()
		return 0
	}
	return todaySessions
}

func (s *Stats) GetBadge() string {
	sessions := s.GetTodaySessions()

	// Emoji badges based on sessions completed today (in order of progression)
	// Check from highest to lowest to get the correct badge
	if sessions >= 20 {
		return "💎"  // Legend
	} else if sessions >= 15 {
		return "🌟"  // Super Star
	} else if sessions >= 10 {
		return "👑"  // Royalty
	} else if sessions >= 8 {
		return "🚀"  // Rocketing
	} else if sessions >= 5 {
		return "💪"  // Strong Work
	} else if sessions >= 3 {
		return "⭐"  // Rising Star
	} else if sessions >= 1 {
		return "🔥"  // On Fire!
	}
	return "🌱"  // Just Started
}

func (s *Stats) GetBadgeDescription() string {
	sessions := s.GetTodaySessions()

	// Return description based on sessions count (in order of progression)
	if sessions >= 20 {
		return "Legend"
	} else if sessions >= 15 {
		return "Super Star"
	} else if sessions >= 10 {
		return "Royalty"
	} else if sessions >= 8 {
		return "Rocketing"
	} else if sessions >= 5 {
		return "Strong Work"
	} else if sessions >= 3 {
		return "Rising Star"
	} else if sessions >= 1 {
		return "On Fire!"
	}
	return "Just Starting"
}
