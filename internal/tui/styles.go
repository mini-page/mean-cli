package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// ─── Palette ──────────────────────────────────────────────────────────────
	colorPrimary   = lipgloss.Color("#7C3AED") // violet-600
	colorAccent    = lipgloss.Color("#A78BFA") // violet-400
	colorSuccess   = lipgloss.Color("#34D399") // emerald-400
	colorWarning   = lipgloss.Color("#FBBF24") // amber-400
	colorDanger    = lipgloss.Color("#F87171") // red-400
	colorMuted     = lipgloss.Color("#6B7280") // gray-500
	colorText      = lipgloss.Color("#F3F4F6") // gray-100
	colorSubtext   = lipgloss.Color("#9CA3AF") // gray-400
	colorBorder    = lipgloss.Color("#374151") // gray-700
	colorHighlight = lipgloss.Color("#1F1B4E") // dark violet bg

	// ─── Base Styles ──────────────────────────────────────────────────────────
	styleBase = lipgloss.NewStyle().
			Foreground(colorText)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSubtext = lipgloss.NewStyle().
			Foreground(colorSubtext)

	// ─── Panel / Box ──────────────────────────────────────────────────────────
	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1).
			MarginLeft(2)

	stylePanelFocused = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorPrimary).
				Padding(0, 1)

	// ─── Title / Header ───────────────────────────────────────────────────────
	styleTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleHeader = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	// ─── Word display ─────────────────────────────────────────────────────────
	styleWordName = lipgloss.NewStyle().
			Foreground(colorText).
			Bold(true).
			Underline(true)

	stylePronunciation = lipgloss.NewStyle().
				Foreground(colorSubtext).
				Italic(true)

	stylePartOfSpeech = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#111827")).
				Background(colorWarning).
				Padding(0, 1).
				Bold(true)

	// ─── Sections ────────────────────────────────────────────────────────────
	styleSectionTitle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#111827")).
				Background(colorAccent).
				Padding(0, 2).
				Bold(true).
				MarginTop(1).
				MarginBottom(1)

	styleSynonym = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).
			Background(colorSuccess).
			Bold(true)

	styleAntonym = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).
			Background(colorDanger).
			Bold(true)

	styleExample = lipgloss.NewStyle().
			Foreground(colorSubtext).
			Italic(true)

	styleEtymology = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleExamBadge = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Background(colorAccent).
			Padding(0, 1).
			Bold(true)

	// ─── Status bar ───────────────────────────────────────────────────────────
	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted).
			Background(lipgloss.Color("#111827")).
			Padding(0, 1)

	styleStatusKey = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// ─── Search ───────────────────────────────────────────────────────────────
	styleSearchBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(0, 1)

	// ─── Cached badge ─────────────────────────────────────────────────────────
	styleCacheBadge = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	// ─── Spinner / loading ────────────────────────────────────────────────────
	styleSpinner = lipgloss.NewStyle().
			Foreground(colorPrimary)

	// ─── Error ────────────────────────────────────────────────────────────────
	styleError = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	// ─── Favorite star ────────────────────────────────────────────────────────
	styleStar = lipgloss.NewStyle().
			Foreground(colorWarning)
)
