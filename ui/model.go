package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"zoneout/audio"
	"zoneout/config"
	"zoneout/models"
	"zoneout/stats"
)

type TickMsg time.Time
type PhaseCompleteMsg struct{}
type RescanMP3sMsg struct{}

type Model struct {
	pomodoro       *models.Pomodoro
	audioPlayer    *audio.AudioPlayer
	appStats       *stats.Stats
	appConfig      *config.Config
	motdManager    interface{} // MOTDManager interface from main package
	selectedMP3    int
	availableMP3s  []string
	showAudioMenu  bool
	showHelp       bool
	lastTickTime   time.Time
	width          int
	height         int
	lastPhaseMode  models.Mode
}

func NewModel(pomodoro *models.Pomodoro, audioPlayer *audio.AudioPlayer, appStats *stats.Stats, appConfig *config.Config, motdManager interface{}) *Model {
	m := &Model{
		pomodoro:       pomodoro,
		audioPlayer:    audioPlayer,
		appStats:       appStats,
		appConfig:      appConfig,
		motdManager:    motdManager,
		selectedMP3:    0,
		lastTickTime:   time.Now(),
		lastPhaseMode:  models.ModeIdle,
	}
	m.availableMP3s = audioPlayer.GetAvailableMP3s()
	return m
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.tickCmd(),
	)
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case TickMsg:
		// Check if MOTD needs refresh (every 24 hours)
		if m.motdManager != nil {
			if motd, ok := m.motdManager.(interface{ NeedsRefresh() bool; Refresh() }); ok {
				if motd.NeedsRefresh() {
					motd.Refresh()
				}
			}
		}

		if m.pomodoro.IsRunning {
			now := time.Now()
			delta := now.Sub(m.lastTickTime)
			m.lastTickTime = now

			previousMode := m.pomodoro.CurrentMode

			if m.pomodoro.Tick(delta) {
				// Phase completed (all sessions done)
				m.audioPlayer.Stop()
			}

			// Check if we just transitioned FROM focus to break (focus session just completed)
			if previousMode == models.ModeFocus && m.pomodoro.CurrentMode == models.ModeBreak {
				// Focus session just completed, add to stats (25 minutes per session)
				m.appStats.AddSession(25)
			}

			// Update the last phase mode
			if m.pomodoro.CurrentMode != models.ModeIdle {
				m.lastPhaseMode = m.pomodoro.CurrentMode
			}

			// Manage audio based on mode
			m.updateAudioMode()
		}
		return m, m.tickCmd()
	case PhaseCompleteMsg:
		// Handle phase completion
	case RescanMP3sMsg:
		if err := m.audioPlayer.ScanWhitenoiseDirectory(); err != nil {
			// Log error
		}
		m.availableMP3s = m.audioPlayer.GetAvailableMP3s()
	}

	return m, nil
}

func (m *Model) updateAudioMode() {
	// Only play audio during FOCUS mode
	if m.pomodoro.CurrentMode == models.ModeFocus {
		// Resume audio if it was playing before and we're back to focus
		if !m.audioPlayer.IsPlaying() && m.pomodoro.IsRunning {
			// Try to resume or restart the last selected audio if available
			if len(m.availableMP3s) > 0 {
				currentMP3 := m.audioPlayer.GetCurrentMP3()
				if currentMP3 == "" {
					// No audio selected yet, play the first one
					m.audioPlayer.PlayMP3(m.availableMP3s[0])
				} else {
					// Resume the previously selected audio
					m.audioPlayer.PlayMP3(currentMP3)
				}
			}
		}
	} else {
		// Pause audio during BREAK or IDLE modes
		if m.audioPlayer.IsPlaying() {
			m.audioPlayer.Pause()
		}
	}
}

func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.audioPlayer.Stop()
		return m, tea.Quit

	case " ": // space - Start/Pause
		if m.pomodoro.CurrentMode == models.ModeIdle {
			m.pomodoro.Start()
			m.lastTickTime = time.Now()
		} else if m.pomodoro.IsRunning && !m.pomodoro.IsPaused {
			m.pomodoro.Pause()
			m.audioPlayer.Pause()
			m.lastTickTime = time.Now() // Reset time to avoid jump on resume
		} else if m.pomodoro.IsPaused {
			m.pomodoro.Resume()
			m.audioPlayer.Resume()
			m.lastTickTime = time.Now() // Reset time to avoid jump on resume
		}

	case "R": // reset cycle
		m.pomodoro.Stop()
		m.audioPlayer.Stop()
		m.showAudioMenu = false

	case "r": // reset session
		m.pomodoro.RemainingTime = m.pomodoro.TotalTime
		m.pomodoro.PlayStartSound()
		m.lastTickTime = time.Now()

	case ">": // skip to next phase
		if m.pomodoro.IsRunning || m.pomodoro.IsPaused {
			m.pomodoro.NextPhase()
			// Update audio mode after skipping
			m.updateAudioMode()
		}

	case "a":
		if len(m.availableMP3s) > 0 {
			m.showAudioMenu = !m.showAudioMenu
		}

	case "up":
		if m.showAudioMenu && m.selectedMP3 > 0 {
			m.selectedMP3--
		}

	case "down":
		if m.showAudioMenu && m.selectedMP3 < len(m.availableMP3s)-1 {
			m.selectedMP3++
		}

	case "enter":
		if m.showAudioMenu && len(m.availableMP3s) > 0 {
			m.audioPlayer.PlayMP3(m.availableMP3s[m.selectedMP3])
			m.showAudioMenu = false
		}

	case "esc":
		if m.showAudioMenu {
			m.showAudioMenu = false
		} else if m.showHelp {
			m.showHelp = false
		}

	case "h", "?":
		m.showHelp = !m.showHelp
		m.showAudioMenu = false // Close audio menu if help opens

	case "m": // New random MOTD
		if m.motdManager != nil {
			if motd, ok := m.motdManager.(interface{ Refresh() }); ok {
				motd.Refresh()
			}
		}

	case "+", "=": // Volume up
		newVolume := m.audioPlayer.VolumeUp()
		if m.appConfig != nil {
			m.appConfig.SetVolume(newVolume)
		}
		// Restart audio with new volume if playing
		if m.audioPlayer.IsPlaying() {
			currentMP3 := m.audioPlayer.GetCurrentMP3()
			if currentMP3 != "" {
				m.audioPlayer.PlayMP3(currentMP3)
			}
		}

	case "-", "_": // Volume down
		newVolume := m.audioPlayer.VolumeDown()
		if m.appConfig != nil {
			m.appConfig.SetVolume(newVolume)
		}
		// Restart audio with new volume if playing
		if m.audioPlayer.IsPlaying() {
			currentMP3 := m.audioPlayer.GetCurrentMP3()
			if currentMP3 != "" {
				m.audioPlayer.PlayMP3(currentMP3)
			}
		}
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	content := m.renderDashboard()

	if m.showHelp {
		content += "\n\n" + m.renderHelp()
	} else if m.showAudioMenu {
		content += "\n\n" + m.renderAudioMenu()
	}

	return content
}

func (m *Model) renderDashboard() string {
	var sb strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B6B")).
		Padding(1, 2)
	sb.WriteString(titleStyle.Render("🍅 ZONEOUT"))
	sb.WriteString("\n\n")

	// Mode display
	modeStr := m.pomodoro.GetModeString()
	modeColor := "#FFD93D"
	if modeStr == "FOCUS" {
		modeColor = "#FF6B6B"
	} else if modeStr == "BREAK" {
		modeColor = "#6BCF7F"
	}

	modeStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(modeColor)).
		PaddingLeft(2)
	sb.WriteString(modeStyle.Render(fmt.Sprintf("Mode: %s", modeStr)))
	sb.WriteString("\n\n")

	// Large timer display with ASCII art style
	timeStr := m.pomodoro.FormatTime()
	largeTimerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00D9FF")).
		PaddingLeft(4).
		PaddingTop(1).
		PaddingBottom(1).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#00D9FF"))

	// Create a large ASCII-style display
	largeTimer := m.createLargeTimer(timeStr)
	sb.WriteString(largeTimerStyle.Render(largeTimer))
	sb.WriteString("\n\n")

	// Progress bar
	progressBar := m.createProgressBar()
	sb.WriteString(progressBar)
	sb.WriteString("\n\n")

	// Session info with stats
	sessionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0E7E5")).
		PaddingLeft(2)
	sb.WriteString(sessionStyle.Render(fmt.Sprintf("Session %d of %d | Completed Today: %d",
		m.pomodoro.CurrentSession, m.pomodoro.Session.TotalSessions, m.appStats.GetTodaySessions())))
	sb.WriteString("\n\n")

	// Badge
	badge := m.appStats.GetBadge()
	badgeDesc := m.appStats.GetBadgeDescription()

	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFD93D")).
		Bold(true).
		PaddingLeft(2)
	sb.WriteString(badgeStyle.Render(fmt.Sprintf("Badge: %s %s",
		badge, badgeDesc)))
	sb.WriteString("\n\n")

	// Status
	statusStr := "Status: Idle"
	statusColor := "#A0E7E5"
	if m.pomodoro.IsRunning {
		statusStr = "Status: Running"
		statusColor = "#6BCF7F"
	} else if m.pomodoro.IsPaused {
		statusStr = "Status: Paused"
		statusColor = "#FFD93D"
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		PaddingLeft(2)
	sb.WriteString(statusStyle.Render(statusStr))
	sb.WriteString("\n\n")

	// Volume level
	volumePercent := int(m.audioPlayer.GetVolume() * 100)
	volumeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A0E7E5")).
		PaddingLeft(2)
	sb.WriteString(volumeStyle.Render(fmt.Sprintf("Volume: %d%%", volumePercent)))
	sb.WriteString("\n\n")

	// MOTD Message
	if m.motdManager != nil {
		if motd, ok := m.motdManager.(interface{ GetMessage() string }); ok && motd != nil {
			message := motd.GetMessage()
			if message != "" {
				motdStyle := lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFD93D")).
					PaddingLeft(2).
					Italic(true)
				sb.WriteString(motdStyle.Render(fmt.Sprintf("💬 %s", message)))
				sb.WriteString("\n\n")
			}
		}
	}

	// Help hint
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		PaddingLeft(2)
	sb.WriteString(hintStyle.Render("Press 'h' or '?' for help"))

	return sb.String()
}

func (m *Model) createLargeTimer(timeStr string) string {
	// Create ASCII art numbers for the timer
	digits := map[string][]string{
		"0": {
			" ███████ ",
			"██     ██",
			"██     ██",
			"██     ██",
			" ███████ ",
		},
		"1": {
			"      █",
			"     ██",
			"    ███",
			"     ██",
			"    ████",
		},
		"2": {
			" ██████ ",
			"██     █",
			"     ██ ",
			"   ██   ",
			"████████",
		},
		"3": {
			" ██████ ",
			"██     █",
			"   ████ ",
			"██     █",
			" ██████ ",
		},
		"4": {
			"██    ██",
			"██    ██",
			"████████",
			"     ██ ",
			"     ██ ",
		},
		"5": {
			"████████",
			"██      ",
			"███████ ",
			"      ██",
			" ██████ ",
		},
		"6": {
			" ███████ ",
			"██       ",
			"████████ ",
			"██     ██",
			" ███████ ",
		},
		"7": {
			"████████",
			"      ██",
			"     ██ ",
			"    ██  ",
			"   ██   ",
		},
		"8": {
			" ███████ ",
			"██     ██",
			" ███████ ",
			"██     ██",
			" ███████ ",
		},
		"9": {
			" ███████ ",
			"██     ██",
			" ████████",
			"       ██",
			" ███████ ",
		},
		":": {
			"   ",
			" █ ",
			"   ",
			" █ ",
			"   ",
		},
	}

	lines := make([]string, 5)
	for _, char := range timeStr {
		charStr := string(char)
		if charArt, ok := digits[charStr]; ok {
			for i := 0; i < 5; i++ {
				lines[i] += charArt[i] + "  "
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m *Model) createProgressBar() string {
	// Bar dimensions - match timer width (approximately 50 chars)
	barWidth := 50
	var filledWidth int

	// Calculate progress based on mode
	if m.pomodoro.CurrentMode == models.ModeIdle || m.pomodoro.TotalTime == 0 {
		// Show empty bar when idle
		filledWidth = 0
	} else {
		progress := float64(m.pomodoro.TotalTime-m.pomodoro.RemainingTime) / float64(m.pomodoro.TotalTime)
		if progress < 0 {
			progress = 0
		}
		if progress > 1 {
			progress = 1
		}
		filledWidth = int(progress * float64(barWidth))
	}

	emptyWidth := barWidth - filledWidth

	// Determine color based on mode
	progressColor := "#666666" // Idle mode - dark gray
	if m.pomodoro.CurrentMode == models.ModeFocus {
		progressColor = "#FF6B6B" // Focus mode - red/orange
	} else if m.pomodoro.CurrentMode == models.ModeBreak {
		progressColor = "#6BCF7F" // Break mode - green
	}

	// Build the bar with colors
	filledBar := strings.Repeat("█", filledWidth)
	emptyBar := strings.Repeat("░", emptyWidth)

	// Create styled components
	filledStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(progressColor))

	emptyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#333333"))

	barText := fmt.Sprintf("[%s%s]",
		filledStyle.Render(filledBar),
		emptyStyle.Render(emptyBar))

	return barText
}

func (m *Model) renderHelp() string {
	var sb strings.Builder

	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Foreground(lipgloss.Color("#00D9FF"))

	sb.WriteString("─── HELP ───\n\n")
	sb.WriteString("SPACE     Start/Pause\n")
	sb.WriteString("R         Reset Cycle (back to idle)\n")
	sb.WriteString("r         Reset Session (restart timer)\n")
	sb.WriteString(">         Skip to next phase\n")
	sb.WriteString("a         Toggle audio menu\n")
	sb.WriteString("+/-       Volume Up/Down\n")
	sb.WriteString("h / ?     Toggle help\n")
	sb.WriteString("m         New random MOTD\n")
	sb.WriteString("ESC       Close menu\n")
	sb.WriteString("q         Quit\n")

	return menuStyle.Render(sb.String())
}

func (m *Model) renderAudioMenu() string {
	var sb strings.Builder

	menuStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Foreground(lipgloss.Color("#00D9FF"))

	sb.WriteString("─── AUDIO MENU ───\n\n")

	if len(m.availableMP3s) == 0 {
		sb.WriteString("No MP3 files found in ./whitenoise/\n")
	} else {
		for i, mp3 := range m.availableMP3s {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.selectedMP3 {
				prefix = "→ "
				style = style.Bold(true).Foreground(lipgloss.Color("#FFD93D"))
			}
			// Extract filename from path
			filename := mp3
			if slashIdx := strings.LastIndex(mp3, "/"); slashIdx >= 0 {
				filename = mp3[slashIdx+1:]
			}
			sb.WriteString(style.Render(prefix + filename + "\n"))
		}
	}

	sb.WriteString("\nenter - Select | ↑/↓ - Navigate | esc - Close\n")

	return menuStyle.Render(sb.String())
}
