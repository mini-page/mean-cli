package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/audio"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/models"
)

// FlashcardModel handles the state of the flashcard study deck.
type FlashcardModel struct {
	db       *cache.DB
	deck     []models.Word
	cursor   int
	flipped  bool
	viewport viewport.Model
	isFav    bool

	Quitting bool

	width  int
	height int

	Embedded bool
}

// NewFlashcardModel creates a new flashcard session.
func NewFlashcardModel(db *cache.DB, customDeck []models.Word) *FlashcardModel {
	return &FlashcardModel{
		db:       db,
		deck:     customDeck,
		viewport: viewport.New(0, 0),
	}
}

func (m FlashcardModel) Init() tea.Cmd {
	return nil
}

func (m *FlashcardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.Quitting = true
			return m, nil

		case " ", "enter":
			m.flipped = !m.flipped
			m.refreshViewport()

		case "right", "n", "l":
			if len(m.deck) > 0 {
				m.cursor = (m.cursor + 1) % len(m.deck)
				m.flipped = false
				m.updateFavState()
				m.refreshViewport()
			}

		case "left", "h":
			if len(m.deck) > 0 {
				m.cursor = (m.cursor - 1 + len(m.deck)) % len(m.deck)
				m.flipped = false
				m.updateFavState()
				m.refreshViewport()
			}

		case "s":
			if len(m.deck) > 0 {
				word := m.deck[m.cursor].Word
				if m.isFav {
					_ = m.db.RemoveFavorite(word)
					m.isFav = false
				} else {
					_ = m.db.AddFavorite(word)
					m.isFav = true
				}
				m.refreshViewport()
			}

		case "p":
			if len(m.deck) > 0 {
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

	if m.flipped {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *FlashcardModel) updateFavState() {
	if len(m.deck) > 0 {
		isFav, _ := m.db.IsFavorite(m.deck[m.cursor].Word)
		m.isFav = isFav
	}
}

func (m *FlashcardModel) resizeViewport() {
	// Padding/margin boundaries
	if m.Embedded {
		m.viewport.Width = m.width - 4
		m.viewport.Height = m.height - 6
	} else {
		m.viewport.Width = m.width - 12
		m.viewport.Height = m.height - 13
	}
	if m.viewport.Height < 3 {
		m.viewport.Height = 3
	}
	m.refreshViewport()
}

func (m *FlashcardModel) refreshViewport() {
	if len(m.deck) == 0 {
		return
	}
	w := m.deck[m.cursor]
	if m.flipped {
		m.viewport.SetContent(buildWordContent(&w, Model{}, m.viewport.Width))
	}
}

func (m FlashcardModel) View() string {
	if m.width == 0 {
		return "initialising study deck…"
	}

	var b strings.Builder

	// Header
	if !m.Embedded {
		b.WriteString(styleHeader.Render(styleTitle.Render("🎴  mean") + styleSubtext.Render("  —  Flashcard Active Recall study")) + "\n\n")
	}

	if len(m.deck) == 0 {
		msg := "\n  No words in your study deck!\n  Search for words and star them, or query definitions to populate history.\n"
		if m.Embedded {
			b.WriteString(msg)
		} else {
			b.WriteString(stylePanel.Width(m.width - 8).Render(msg) + "\n")
			b.WriteString("\n  " + keyBind("q", "quit") + "\n")
		}
		return b.String()
	}

	w := m.deck[m.cursor]
	var cardW int
	if m.Embedded {
		cardW = m.width
	} else {
		cardW = m.width - 8
		if cardW < 20 {
			cardW = 20
		}
	}

	// Card visual body
	var cardContent string
	star := ""
	if m.isFav {
		star = styleStar.Render(" ★")
	}

	if !m.flipped {
		// Front of Card
		var cardBody strings.Builder
		cardBody.WriteString("\n\n")
		cardBody.WriteString(lipgloss.NewStyle().Width(cardW - 4).Align(lipgloss.Center).Render(
			styleWordName.Copy().Underline(false).Foreground(colorAccent).Render(strings.ToUpper(w.Word)) + star,
		) + "\n")

		if w.Pronunciation != "" {
			cardBody.WriteString(lipgloss.NewStyle().Width(cardW - 4).Align(lipgloss.Center).Render(
				stylePronunciation.Render(w.Pronunciation),
			) + "\n")
		}
		if w.ExamLevel != "" {
			cardBody.WriteString("\n" + lipgloss.NewStyle().Width(cardW - 4).Align(lipgloss.Center).Render(
				styleExamBadge.Render(w.ExamLevel),
			) + "\n")
		}

		cardBody.WriteString("\n\n" + lipgloss.NewStyle().Width(cardW - 4).Align(lipgloss.Center).Render(
			styleMuted.Render("[ Press Space/Enter to flip card ]"),
		) + "\n")

		cardContent = cardBody.String()
	} else {
		// Back of Card (shows definitions using viewport)
		var cardBody strings.Builder
		cardBody.WriteString(styleWordName.Copy().Underline(false).Foreground(colorAccent).Render(strings.ToUpper(w.Word)) + star + "\n")
		cardBody.WriteString(m.viewport.View())
		cardContent = cardBody.String()
	}

	if m.Embedded {
		deckProgress := fmt.Sprintf("Card %d of %d", m.cursor+1, len(m.deck))
		b.WriteString(cardContent + "\n\n" + styleExamBadge.Render(deckProgress) + "\n")
	} else {
		// Render card inside rounded border
		b.WriteString(stylePanel.Width(cardW).Render(cardContent) + "\n\n")

		// Stats and footer controls
		deckProgress := fmt.Sprintf("Card %d of %d", m.cursor+1, len(m.deck))
		b.WriteString("  " + styleExamBadge.Render(deckProgress) + "\n\n")

		// Status navigation helpers
		keys := []string{
			keyBind("Space/Enter", "flip"),
			keyBind("→ / n", "next"),
			keyBind("← / h", "prev"),
			keyBind("s", "star"),
			keyBind("p", "pronounce"),
			keyBind("q", "exit study"),
		}
		b.WriteString("  " + styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  ")) + "\n")
	}

	return b.String()
}

// StartFlashcards launches the active recall Bubble Tea session.
func StartFlashcards(db *cache.DB, deck []models.Word) error {
	m := NewFlashcardModel(db, deck)
	m.updateFavState()
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
