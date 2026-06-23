package games

import (
	"math/rand"
	"time"

	"github.com/umang/mean-cli/internal/models"
)

// MatchSession manages a multiple-choice definition matching puzzle.
type MatchSession struct {
	Word          models.Word
	Options       []string
	CorrectIndex int
}

// NewMatchSession builds a question for a target word against a deck.
func NewMatchSession(target models.Word, deck []models.Word) *MatchSession {
	// Pick random definition
	def := ""
	if len(target.Definitions) > 0 {
		def = target.Definitions[0].Meaning
	}

	optionsMap := map[string]bool{target.Word: true}
	var optionWords []string
	optionWords = append(optionWords, target.Word)

	// Pick 3 distractors
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for len(optionWords) < 4 && len(deck) >= 4 {
		dist := deck[r.Intn(len(deck))]
		if !optionsMap[dist.Word] && dist.Word != "" {
			optionsMap[dist.Word] = true
			optionWords = append(optionWords, dist.Word)
		}
	}

	// Distractor fallback if deck is too small
	fallbacks := []string{"anachronism", "cacophony", "gregarious", "ephemeral", "capricious", "dilatory"}
	for len(optionWords) < 4 {
		f := fallbacks[r.Intn(len(fallbacks))]
		if !optionsMap[f] {
			optionsMap[f] = true
			optionWords = append(optionWords, f)
		}
	}

	// Shuffle options
	r.Shuffle(len(optionWords), func(i, j int) {
		optionWords[i], optionWords[j] = optionWords[j], optionWords[i]
	})

	correctIdx := 0
	for i, opt := range optionWords {
		if opt == target.Word {
			correctIdx = i
			break
		}
	}

	// Create a dummy copy of the word with a single definition to show on card
	targetDummy := target
	if def != "" {
		targetDummy.Definitions = []models.Definition{
			{Meaning: def, PartOfSpeech: target.Definitions[0].PartOfSpeech},
		}
	}

	return &MatchSession{
		Word:         targetDummy,
		Options:      optionWords,
		CorrectIndex: correctIdx,
	}
}

// CheckGuess verifies if option index (0 to 3) is correct.
func (m *MatchSession) CheckGuess(idx int) bool {
	return idx == m.CorrectIndex
}
