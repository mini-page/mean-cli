package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/models"
)

type GameCenterState int

const (
	stateMenu GameCenterState = iota
	stateHangman
	stateMatcher
	stateQuiz
	stateFlashcards
)

type GameCenterModel struct {
	db        *cache.DB
	deck      []models.Word
	state     GameCenterState
	cursor    int
	hangman   *HangmanModel
	matcher   *TuiMatchModel
	quiz      *QuizModel
	flashcard *FlashcardModel

	width  int
	height int

	Embedded bool
}

func NewGameCenterModel(db *cache.DB, deck []models.Word) *GameCenterModel {
	return &GameCenterModel{
		db:   db,
		deck: deck,
	}
}

func (m GameCenterModel) Init() tea.Cmd {
	return nil
}

func (m *GameCenterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Keep track of window size across states
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		if m.Embedded {
			contentWidth := sz.Width - 32
			if contentWidth < 15 {
				contentWidth = 15
			}
			panelHeight := sz.Height - 7
			if panelHeight < 5 {
				panelHeight = 5
			}
			m.width = contentWidth - 4
			m.height = panelHeight - 2
		} else {
			m.width = sz.Width
			m.height = sz.Height
		}
		if m.hangman != nil {
			m.hangman.width = m.width
			m.hangman.height = m.height
		}
		if m.matcher != nil {
			m.matcher.width = m.width
			m.matcher.height = m.height
		}
		if m.quiz != nil {
			m.quiz.width = m.width
			m.quiz.height = m.height
		}
		if m.flashcard != nil {
			m.flashcard.width = m.width
			m.flashcard.height = m.height
			m.flashcard.resizeViewport()
		}
	}

	if m.state == stateMenu {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "q", "esc":
				return m, tea.Quit

			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}

			case "down", "j":
				if m.cursor < 3 {
					m.cursor++
				}

			case "enter":
				// Initialize sub-model and transition state
				switch m.cursor {
				case 0:
					m.hangman = NewHangmanModel(m.db, m.deck)
					m.hangman.width = m.width
					m.hangman.height = m.height
					m.hangman.Embedded = m.Embedded
					m.state = stateHangman
				case 1:
					m.matcher = NewTuiMatchModel(m.db, m.deck)
					m.matcher.width = m.width
					m.matcher.height = m.height
					m.matcher.Embedded = m.Embedded
					m.state = stateMatcher
				case 2:
					m.quiz = NewQuizModel(m.db, m.deck)
					m.quiz.width = m.width
					m.quiz.height = m.height
					m.quiz.Embedded = m.Embedded
					m.state = stateQuiz
				case 3:
					m.flashcard = NewFlashcardModel(m.db, m.deck)
					m.flashcard.width = m.width
					m.flashcard.height = m.height
					m.flashcard.Embedded = m.Embedded
					m.flashcard.resizeViewport()
					m.state = stateFlashcards
				}
			}
		}
		return m, nil
	}

	// Delegate update based on current sub-game state
	switch m.state {
	case stateHangman:
		var raw tea.Model
		var cmd tea.Cmd
		raw, cmd = m.hangman.Update(msg)
		m.hangman = raw.(*HangmanModel)
		cmds = append(cmds, cmd)

		if m.hangman.Quitting {
			m.state = stateMenu
			m.hangman = nil
		}

	case stateMatcher:
		var raw tea.Model
		var cmd tea.Cmd
		raw, cmd = m.matcher.Update(msg)
		m.matcher = raw.(*TuiMatchModel)
		cmds = append(cmds, cmd)

		if m.matcher.Quitting {
			m.state = stateMenu
			m.matcher = nil
		}

	case stateQuiz:
		var raw tea.Model
		var cmd tea.Cmd
		raw, cmd = m.quiz.Update(msg)
		m.quiz = raw.(*QuizModel)
		cmds = append(cmds, cmd)

		if m.quiz.Quitting {
			m.state = stateMenu
			m.quiz = nil
		}

	case stateFlashcards:
		var raw tea.Model
		var cmd tea.Cmd
		raw, cmd = m.flashcard.Update(msg)
		m.flashcard = raw.(*FlashcardModel)
		cmds = append(cmds, cmd)

		if m.flashcard.Quitting {
			m.state = stateMenu
			m.flashcard = nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m GameCenterModel) View() string {
	if m.width == 0 {
		return "initialising Game Center…"
	}

	// Render child view directly if active
	switch m.state {
	case stateHangman:
		return m.hangman.View()
	case stateMatcher:
		return m.matcher.View()
	case stateQuiz:
		return m.quiz.View()
	case stateFlashcards:
		return m.flashcard.View()
	}

	var b strings.Builder

	// Header (only standalone)
	if !m.Embedded {
		b.WriteString(styleHeader.Render(styleTitle.Render("🎮  mean") + styleSubtext.Render("  —  Study & Game Center Menu")) + "\n\n")
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

	// Dashboard Panel
	var panelBody strings.Builder
	panelBody.WriteString(styleWordName.Copy().Underline(false).Foreground(colorAccent).Render("CHOOSE YOUR GAME MODE:") + "\n\n")

	menuItems := []struct {
		title string
		desc  string
	}{
		{"1. Hangman", "Classic letter guessing vocabulary game"},
		{"2. Definition Matcher", "Multiple choice definition matching puzzle"},
		{"3. Spelling Quiz", "Guess and type the word from its definitions"},
		{"4. Active Flashcards", "Flippable active recall cards deck"},
	}

	for i, item := range menuItems {
		var line string
		if i == m.cursor {
			line = fmt.Sprintf(" ▶  %s  %s", 
				lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(item.title),
				styleSubtext.Render("·  "+item.desc),
			)
		} else {
			line = fmt.Sprintf("    %s  %s", 
				styleWordName.Copy().Underline(false).Render(item.title),
				styleMuted.Render("·  "+item.desc),
			)
		}
		panelBody.WriteString(line + "\n")
	}

	if m.Embedded {
		b.WriteString(panelBody.String())
	} else {
		b.WriteString(stylePanel.Width(cardW).Render(panelBody.String()) + "\n\n")

		// Status helpers
		keys := []string{
			keyBind("↑↓ / jk", "move cursor"),
			keyBind("Enter", "select game"),
			keyBind("q / Esc", "quit menu"),
		}
		b.WriteString("  " + styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  ")) + "\n")
	}

	return b.String()
}

// StartGameCenter launches the unified Bubble Tea Game Center dashboard.
func StartGameCenter(db *cache.DB, deck []models.Word) error {
	m := NewGameCenterModel(db, deck)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
