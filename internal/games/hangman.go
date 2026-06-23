package games

import (
	"strings"

	"github.com/umang/mean-cli/internal/models"
)

// Gallows stages for Hangman drawing.
var Gallows = []string{
	`
  +---+
  |   |
      |
      |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
      |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
  |   |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|   |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|\  |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|\  |
 /    |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|\  |
 / \  |
      |
=========`,
}

// HangmanSession holds the running state of a Hangman game.
type HangmanSession struct {
	Word    models.Word
	Guesses []rune
	Lives   int
}

// NewHangmanSession creates a session with a target word.
func NewHangmanSession(word models.Word) *HangmanSession {
	return &HangmanSession{
		Word:  word,
		Lives: 6,
	}
}

// Guess checks the input character and adjusts lives. Returns true if correct.
func (h *HangmanSession) Guess(char rune) bool {
	char = toLowerRune(char)
	if h.HasGuessed(char) {
		return false
	}
	h.Guesses = append(h.Guesses, char)

	// Check if correct
	correct := false
	for _, c := range h.Word.Word {
		if toLowerRune(c) == char {
			correct = true
			break
		}
	}
	if !correct {
		h.Lives--
	}
	return correct
}

// HasGuessed checks if letter was already entered.
func (h *HangmanSession) HasGuessed(char rune) bool {
	for _, g := range h.Guesses {
		if g == char {
			return true
		}
	}
	return false
}

// IsWon reports if all word letters are successfully guessed.
func (h *HangmanSession) IsWon() bool {
	for _, c := range h.Word.Word {
		if isLetter(c) && !h.HasGuessed(toLowerRune(c)) {
			return false
		}
	}
	return true
}

// IsLost reports if lives are exhausted.
func (h *HangmanSession) IsLost() bool {
	return h.Lives <= 0
}

// DisplayWord returns the word with blanks (e.g. "e p h _ _ _ _ a l").
func (h *HangmanSession) DisplayWord() string {
	var result []string
	for _, c := range h.Word.Word {
		if !isLetter(c) {
			result = append(result, string(c))
		} else if h.HasGuessed(toLowerRune(c)) {
			result = append(result, string(c))
		} else {
			result = append(result, "_")
		}
	}
	return strings.Join(result, " ")
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func toLowerRune(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}
