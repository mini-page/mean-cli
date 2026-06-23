package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/audio"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/models"
)

type quizState int

const (
	stateGuessing quizState = iota
	stateAnswered
)

type QuizModel struct {
	db       *cache.DB
	deck     []models.Word
	cursor   int
	input    textinput.Model
	state    quizState
	guess    string
	isCorrect bool

	score  int
	solved int
	streak int

	Quitting bool

	width  int
	height int
}

func NewQuizModel(db *cache.DB, deck []models.Word) *QuizModel {
	ti := textinput.New()
	ti.Placeholder = "Type your guess here and press Enter"
	ti.Focus()
	ti.CharLimit = 100
	ti.Prompt = "✍️  "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	return &QuizModel{
		db:    db,
		deck:  deck,
		input: ti,
		state: stateGuessing,
	}
}

func (m QuizModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m QuizModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			if m.state == stateAnswered {
				m.Quitting = true
				return m, nil
			}
		case "esc":
			m.Quitting = true
			return m, nil

		case "enter":
			if len(m.deck) == 0 {
				return m, tea.Quit
			}

			if m.state == stateGuessing {
				// User submitted a guess
				m.guess = strings.TrimSpace(m.input.Value())
				if m.guess == "" {
					return m, nil
				}

				correctWord := strings.ToLower(m.deck[m.cursor].Word)
				guessWord := strings.ToLower(m.guess)

				m.state = stateAnswered
				m.solved++

				if guessWord == correctWord {
					m.isCorrect = true
					m.score++
					m.streak++
				} else {
					m.isCorrect = false
					m.streak = 0
				}
			} else {
				// Go to next card
				m.cursor = (m.cursor + 1) % len(m.deck)
				m.state = stateGuessing
				m.input.SetValue("")
				m.input.Focus()
			}

		case "p":
			// If correct/answered, allow pronouncing the correct word
			if m.state == stateAnswered && len(m.deck) > 0 {
				w := m.deck[m.cursor]
				for _, ph := range w.Phonetics {
					if ph.Audio != "" {
						audio.PlayFromURL(ph.Audio)
						break
					}
				}
			}
		}
	}

	if m.state == stateGuessing {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m QuizModel) View() string {
	if m.width == 0 {
		return "initialising vocabulary quiz…"
	}

	var b strings.Builder

	// Header
	b.WriteString(styleHeader.Render(styleTitle.Render("🎓  mean") + styleSubtext.Render("  —  Vocabulary Master Quiz")) + "\n\n")

	if len(m.deck) == 0 {
		b.WriteString(stylePanel.Width(m.width - 8).Render(
			"\n  No words available for a quiz!\n  Star words or view definitions to compile local history.\n",
		) + "\n")
		b.WriteString("\n  " + keyBind("q", "quit") + "\n")
		return b.String()
	}

	w := m.deck[m.cursor]
	cardW := m.width - 8
	if cardW < 20 {
		cardW = 20
	}

	// Quiz Body
	var body strings.Builder

	// Hiding the target word in sentences/definitions
	blanks := strings.Repeat("_ ", len(w.Word))
	body.WriteString(styleWordName.Copy().Underline(false).Foreground(colorAccent).Render("GUESS THE WORD:") + "  " + stylePronunciation.Render(blanks) + "\n\n")

	// Print definitions (hiding the target word)
	if len(w.Definitions) > 0 {
		body.WriteString(styleSectionTitle.Render(" Definitions ") + "\n")
		for i, d := range w.Definitions {
			cleanMeaning := censorWord(d.Meaning, w.Word, "[_____]")
			body.WriteString(fmt.Sprintf("    %d. %s  %s\n", i+1, stylePartOfSpeech.Render(" "+strings.ToUpper(d.PartOfSpeech)+" "), cleanMeaning))
		}
	}

	// Examples
	var cleanExamples []string
	for _, ex := range w.Examples {
		cleanExamples = append(cleanExamples, censorWord(ex, w.Word, "[_____]"))
	}
	if len(cleanExamples) > 0 {
		body.WriteString("\n" + styleSectionTitle.Render(" Example Usage ") + "\n")
		for i, ex := range cleanExamples {
			body.WriteString(fmt.Sprintf("    %d. %s\n", i+1, styleExample.Render("\""+ex+"\"")))
		}
	}

	body.WriteString("\n")

	// Render guessing/feedback states
	if m.state == stateGuessing {
		body.WriteString(styleSearchBox.Width(cardW - 4).Render(m.input.View()) + "\n")
	} else {
		// Answered feedback state
		var feedback string
		if m.isCorrect {
			feedback = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#111827")).
				Background(colorSuccess).
				Padding(0, 2).
				Bold(true).
				Render("✓ CORRECT!")
		} else {
			feedback = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#111827")).
				Background(colorDanger).
				Padding(0, 2).
				Bold(true).
				Render(fmt.Sprintf("✗ INCORRECT!  The word was: %s", strings.ToUpper(w.Word)))
		}
		body.WriteString("  " + feedback + "\n\n")
		
		pronounceHelp := ""
		for _, ph := range w.Phonetics {
			if ph.Audio != "" {
				pronounceHelp = "  " + keyBind("p", "pronounce word")
				break
			}
		}
		body.WriteString("  " + styleMuted.Render("[ Press Enter to continue ]") + pronounceHelp + "\n")
	}

	b.WriteString(stylePanel.Width(cardW).Render(body.String()) + "\n\n")

	// Scoring layout
	scorePct := 0.0
	if m.solved > 0 {
		scorePct = (float64(m.score) / float64(m.solved)) * 100.0
	}
	statsStr := fmt.Sprintf("Score: %d/%d (%.1f%%)   Streak: 🔥 %d", m.score, m.solved, scorePct, m.streak)
	b.WriteString("  " + styleExamBadge.Render(statsStr) + "\n\n")

	// Status helpers
	keys := []string{
		keyBind("Enter", "submit / next"),
		keyBind("q", "quit quiz"),
	}
	b.WriteString("  " + styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  ")) + "\n")

	return b.String()
}

func censorWord(text, target, replacement string) string {
	lowerText := strings.ToLower(text)
	lowerTarget := strings.ToLower(target)
	idx := strings.Index(lowerText, lowerTarget)
	if idx == -1 {
		return text
	}
	// Case-insensitive replace for target word
	return text[:idx] + replacement + censorWord(text[idx+len(target):], target, replacement)
}

func StartQuiz(db *cache.DB, deck []models.Word) error {
	m := NewQuizModel(db, deck)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
