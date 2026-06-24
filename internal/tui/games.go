package tui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/audio"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/games"
	"github.com/umang/mean-cli/internal/models"
)

var (
	styleCorrect   = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	styleWrong     = lipgloss.NewStyle().Foreground(colorDanger).Bold(true)
	styleHighlight = lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
)

// ─── TUI Hangman Model ────────────────────────────────────────────────────────

type HangmanModel struct {
	db      *cache.DB
	deck    []models.Word
	session *games.HangmanSession

	score  int
	solved int
	streak int

	Quitting bool

	width  int
	height int

	Embedded bool
}

func NewHangmanModel(db *cache.DB, deck []models.Word) *HangmanModel {
	m := &HangmanModel{
		db:   db,
		deck: deck,
	}
	m.nextQuestion()
	return m
}

func (m *HangmanModel) nextQuestion() {
	if len(m.deck) > 0 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		target := m.deck[r.Intn(len(m.deck))]
		m.session = games.NewHangmanSession(target)
	}
}

func (m HangmanModel) Init() tea.Cmd {
	return nil
}

func (m HangmanModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.Quitting = true
			return m, nil

		case "enter":
			if m.session.IsWon() || m.session.IsLost() {
				m.nextQuestion()
			}

		case "p":
			if (m.session.IsWon() || m.session.IsLost()) && len(m.deck) > 0 {
				w := m.session.Word
				for _, ph := range w.Phonetics {
					if ph.Audio != "" {
						audio.PlayFromURL(ph.Audio)
						break
					}
				}
			}

		default:
			// Capture single letter guesses
			s := msg.String()
			if len(s) == 1 && !m.session.IsWon() && !m.session.IsLost() {
				char := []rune(s)[0]
				if isLetterRune(char) {
					m.session.Guess(char)
					if m.session.IsWon() {
						m.score++
						m.solved++
						m.streak++
					} else if m.session.IsLost() {
						m.solved++
						m.streak = 0
					}
				}
			}
		}
	}
	return m, nil
}

func (m HangmanModel) View() string {
	if m.width == 0 {
		return "initialising hangman…"
	}

	var b strings.Builder

	// Header
	if !m.Embedded {
		b.WriteString(styleHeader.Render(styleTitle.Render("🎮  mean") + styleSubtext.Render("  —  Hangman Vocabulary TUI Game")) + "\n\n")
	}

	if len(m.deck) == 0 {
		msg := "\n  No words available for Hangman!\n  Star words or view definitions to compile local history.\n"
		if m.Embedded {
			b.WriteString(msg)
		} else {
			b.WriteString(stylePanel.Width(m.width - 8).Render(msg) + "\n")
			b.WriteString("\n  " + keyBind("q", "quit") + "\n")
		}
		return b.String()
	}

	var cardW int
	if m.Embedded {
		cardW = m.width
	} else {
		cardW = m.width - 8
		if cardW < 20 {
			cardW = 20
		}
	}

	var body strings.Builder

	// Gallows drawing on the left side
	gallowsStr := games.Gallows[6-m.session.Lives]

	// Word Blanks and guess states
	var gameSide strings.Builder
	gameSide.WriteString("\n")
	gameSide.WriteString(styleWordName.Copy().Underline(false).Foreground(colorAccent).Render("WORD TO GUESS:") + "\n")
	gameSide.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(colorText).Render(m.session.DisplayWord()) + "\n\n")
	gameSide.WriteString("  Lives: " + strings.Repeat("❤️ ", m.session.Lives) + strings.Repeat("🖤 ", 6-m.session.Lives) + "\n\n")
	gameSide.WriteString("  Guessed letters: " + styleSubtext.Render(stringGuessesList(m.session.Guesses)) + "\n")

	// Render side-by-side or stacked depending on width
	var viewBlock string
	if cardW > 60 {
		viewBlock = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(24).Render(gallowsStr),
			gameSide.String(),
		)
	} else {
		viewBlock = gallowsStr + "\n" + gameSide.String()
	}
	body.WriteString(viewBlock + "\n\n")

	// Feedback and details
	if m.session.IsWon() {
		body.WriteString("  " + lipgloss.NewStyle().Background(colorSuccess).Foreground(lipgloss.Color("#111827")).Padding(0, 2).Bold(true).Render("🎉 YOU GUESSED IT!") + "\n\n")
		body.WriteString("  The word was: " + styleHighlight.Render(strings.ToUpper(m.session.Word.Word)) + "\n")
		body.WriteString("  " + styleMuted.Render("[ Press Enter to play next ]") + keyBind("p", "pronounce") + "\n")
	} else if m.session.IsLost() {
		body.WriteString("  " + lipgloss.NewStyle().Background(colorDanger).Foreground(lipgloss.Color("#111827")).Padding(0, 2).Bold(true).Render("💀 GAME OVER!") + "\n\n")
		body.WriteString("  The word was: " + styleHighlight.Render(strings.ToUpper(m.session.Word.Word)) + "\n")
		body.WriteString("  " + styleMuted.Render("[ Press Enter to play next ]") + keyBind("p", "pronounce") + "\n")
	} else {
		body.WriteString("  " + styleMuted.Render("Type any letter key [a-z] on your keyboard to guess!"))
	}

	if m.Embedded {
		// Streak/Score tracker inside body
		statsStr := fmt.Sprintf("Solved: %d   Wins: %d   Streak: 🔥 %d", m.solved, m.score, m.streak)
		body.WriteString("\n\n" + styleExamBadge.Render(statsStr) + "\n")
		b.WriteString(body.String())
	} else {
		b.WriteString(stylePanel.Width(cardW).Render(body.String()) + "\n\n")

		// Streak/Score tracker
		statsStr := fmt.Sprintf("Solved: %d   Wins: %d   Streak: 🔥 %d", m.solved, m.score, m.streak)
		b.WriteString("  " + styleExamBadge.Render(statsStr) + "\n\n")

		// Status helpers
		keys := []string{
			keyBind("[a-z]", "guess letter"),
			keyBind("q", "quit game"),
		}
		b.WriteString("  " + styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  ")) + "\n")
	}

	return b.String()
}

func isLetterRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func stringGuessesList(runs []rune) string {
	if len(runs) == 0 {
		return "none"
	}
	var s []string
	for _, r := range runs {
		s = append(s, string(r))
	}
	return strings.Join(s, "  ")
}

// ─── TUI Match Model ──────────────────────────────────────────────────────────

type TuiMatchModel struct {
	db      *cache.DB
	deck    []models.Word
	session *games.MatchSession
	state   quizState // reuse quizState guessing/answered enum
	guess   int

	score  int
	solved int
	streak int

	Quitting bool

	width  int
	height int

	Embedded bool
}

func NewTuiMatchModel(db *cache.DB, deck []models.Word) *TuiMatchModel {
	m := &TuiMatchModel{
		db:   db,
		deck: deck,
	}
	m.nextQuestion()
	return m
}

func (m *TuiMatchModel) nextQuestion() {
	if len(m.deck) >= 2 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		target := m.deck[r.Intn(len(m.deck))]
		m.session = games.NewMatchSession(target, m.deck)
		m.state = stateGuessing
	}
}

func (m TuiMatchModel) Init() tea.Cmd {
	return nil
}

func (m TuiMatchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.Quitting = true
			return m, nil

		case "enter":
			if m.state == stateAnswered {
				m.nextQuestion()
			}

		case "p":
			if m.state == stateAnswered && len(m.deck) > 0 {
				w := m.session.Word
				for _, ph := range w.Phonetics {
					if ph.Audio != "" {
						audio.PlayFromURL(ph.Audio)
						break
					}
				}
			}

		default:
			// Capture A/B/C/D input keys
			if m.state == stateGuessing {
				inp := strings.ToLower(msg.String())
				guess := -1
				switch inp {
				case "a":
					guess = 0
				case "b":
					guess = 1
				case "c":
					guess = 2
				case "d":
					guess = 3
				}
				if guess != -1 {
					m.guess = guess
					m.state = stateAnswered
					m.solved++
					if m.session.CheckGuess(guess) {
						m.score++
						m.streak++
					} else {
						m.streak = 0
					}
				}
			}
		}
	}
	return m, nil
}

func (m TuiMatchModel) View() string {
	if m.width == 0 {
		return "initialising match game…"
	}

	var b strings.Builder

	// Header
	if !m.Embedded {
		b.WriteString(styleHeader.Render(styleTitle.Render("🎮  mean") + styleSubtext.Render("  —  Definition Matcher Game")) + "\n\n")
	}

	if len(m.deck) < 2 {
		msg := "\n  Matcher requires a study deck of at least 2 words!\n  Star words or view definitions to compile local history.\n"
		if m.Embedded {
			b.WriteString(msg)
		} else {
			b.WriteString(stylePanel.Width(m.width - 8).Render(msg) + "\n")
			b.WriteString("\n  " + keyBind("q", "quit") + "\n")
		}
		return b.String()
	}

	var cardW int
	if m.Embedded {
		cardW = m.width
	} else {
		cardW = m.width - 8
		if cardW < 20 {
			cardW = 20
		}
	}

	var body strings.Builder

	// Render Definition
	def := "No definition found"
	if len(m.session.Word.Definitions) > 0 {
		def = m.session.Word.Definitions[0].Meaning
	}
	body.WriteString(styleWordName.Copy().Underline(false).Foreground(colorAccent).Render("MATCH THIS DEFINITION:") + "\n\n")
	body.WriteString("  " + styleSubtext.Bold(true).Render(def) + "\n\n")

	body.WriteString(styleSectionTitle.Render(" Options ") + "\n")

	optionKeys := []string{"A", "B", "C", "D"}
	for i, opt := range m.session.Options {
		// Highlight options based on state
		prefix := "   " + optionKeys[i] + ". "
		optionLine := prefix + opt
		if m.state == stateAnswered {
			if i == m.session.CorrectIndex {
				optionLine = "  " + styleCorrect.Render("✓ "+optionKeys[i]+". "+opt)
			} else if i == m.guess {
				optionLine = "  " + styleWrong.Render("✗ "+optionKeys[i]+". "+opt)
			} else {
				optionLine = "   " + styleMuted.Render(optionKeys[i]+". "+opt)
			}
		} else {
			optionLine = prefix + styleWordName.Copy().Underline(false).Render(opt)
		}
		body.WriteString(optionLine + "\n")
	}

	body.WriteString("\n")

	// Answer feedback
	if m.state == stateAnswered {
		var feedback string
		if m.guess == m.session.CorrectIndex {
			feedback = lipgloss.NewStyle().
				Background(colorSuccess).Foreground(lipgloss.Color("#111827")).
				Padding(0, 2).Bold(true).Render("✓ CORRECT!")
		} else {
			feedback = lipgloss.NewStyle().
				Background(colorDanger).Foreground(lipgloss.Color("#111827")).
				Padding(0, 2).Bold(true).Render(fmt.Sprintf("✗ INCORRECT!  The correct word was: %s", strings.ToUpper(m.session.Word.Word)))
		}
		body.WriteString("  " + feedback + "\n\n")
		body.WriteString("  " + styleMuted.Render("[ Press Enter to continue ]") + keyBind("p", "pronounce word") + "\n")
	} else {
		body.WriteString("  " + styleMuted.Render("Press A, B, C, or D to select your answer."))
	}

	if m.Embedded {
		// Streak/Score tracker inside body
		scorePct := 0.0
		if m.solved > 0 {
			scorePct = (float64(m.score) / float64(m.solved)) * 100.0
		}
		statsStr := fmt.Sprintf("Score: %d/%d (%.1f%%)   Streak: 🔥 %d", m.score, m.solved, scorePct, m.streak)
		body.WriteString("\n\n" + styleExamBadge.Render(statsStr) + "\n")
		b.WriteString(body.String())
	} else {
		b.WriteString(stylePanel.Width(cardW).Render(body.String()) + "\n\n")

		// Streak/Score tracker
		scorePct := 0.0
		if m.solved > 0 {
			scorePct = (float64(m.score) / float64(m.solved)) * 100.0
		}
		statsStr := fmt.Sprintf("Score: %d/%d (%.1f%%)   Streak: 🔥 %d", m.score, m.solved, scorePct, m.streak)
		b.WriteString("  " + styleExamBadge.Render(statsStr) + "\n\n")

		// Status helpers
		keys := []string{
			keyBind("A/B/C/D", "make choice"),
			keyBind("q", "quit game"),
		}
		b.WriteString("  " + styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  ")) + "\n")
	}

	return b.String()
}

// ─── TUI Launch Runners ───────────────────────────────────────────────────────

func StartHangman(db *cache.DB, deck []models.Word) error {
	m := NewHangmanModel(db, deck)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func StartMatcher(db *cache.DB, deck []models.Word) error {
	m := NewTuiMatchModel(db, deck)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
