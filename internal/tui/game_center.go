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

func (m GameCenterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Keep track of window size across states
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sz.Width
		m.height = sz.Height
		if m.hangman != nil {
			m.hangman.width = sz.Width
			m.hangman.height = sz.Height
		}
		if m.matcher != nil {
			m.matcher.width = sz.Width
			m.matcher.height = sz.Height
		}
		if m.quiz != nil {
			m.quiz.width = sz.Width
			m.quiz.height = sz.Height
		}
		if m.flashcard != nil {
			m.flashcard.width = sz.Width
			m.flashcard.height = sz.Height
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
					m.state = stateHangman
				case 1:
					m.matcher = NewTuiMatchModel(m.db, m.deck)
					m.matcher.width = m.width
					m.matcher.height = m.height
					m.state = stateMatcher
				case 2:
					m.quiz = NewQuizModel(m.db, m.deck)
					m.quiz.width = m.width
					m.quiz.height = m.height
					m.state = stateQuiz
				case 3:
					m.flashcard = NewFlashcardModel(m.db, m.deck)
					m.flashcard.width = m.width
					m.flashcard.height = m.height
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

	// Header
	b.WriteString(styleHeader.Render(styleTitle.Render("🎮  mean") + styleSubtext.Render("  —  Study & Game Center Menu")) + "\n\n")

	cardW := m.width - 8
	if cardW < 20 {
		cardW = 20
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

	b.WriteString(stylePanel.Width(cardW).Render(panelBody.String()) + "\n\n")

	// Status helpers
	keys := []string{
		keyBind("↑↓ / jk", "move cursor"),
		keyBind("Enter", "select game"),
		keyBind("q / Esc", "quit menu"),
	}
	b.WriteString("  " + styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  ")) + "\n")

	return b.String()
}

// StartGameCenter launches the unified Bubble Tea Game Center dashboard.
func StartGameCenter(db *cache.DB, deck []models.Word) error {
	m := NewGameCenterModel(db, deck)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
