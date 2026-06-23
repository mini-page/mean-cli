package api

import (
	"fmt"
	"strings"

	"github.com/umang/mean-cli/internal/models"
)

// Lookup is the main entry point. It tries the Free Dictionary API first,
// then enriches missing synonyms/antonyms via Datamuse.
func Lookup(word string) (*models.Word, error) {
	w, err := LookupFreeDictionary(word)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %w", err)
	}

	// Enrich synonyms if missing
	if len(w.Synonyms) == 0 {
		syns, serr := FetchSynonyms(word)
		if serr == nil {
			w.Synonyms = syns
		}
	}

	// Enrich antonyms if missing
	if len(w.Antonyms) == 0 {
		ants, aerr := FetchAntonyms(word)
		if aerr == nil {
			w.Antonyms = ants
		}
	}

	// Limit lists for display
	if len(w.Synonyms) > 10 {
		w.Synonyms = w.Synonyms[:10]
	}
	if len(w.Antonyms) > 10 {
		w.Antonyms = w.Antonyms[:10]
	}
	if len(w.Examples) > 5 {
		w.Examples = w.Examples[:5]
	}

	// Assign exam level heuristic
	w.ExamLevel = guessExamLevel(word)

	return w, nil
}

// guessExamLevel applies a simple heuristic based on word length + complexity.
func guessExamLevel(word string) string {
	l := len(strings.Fields(word))
	wl := len(word)
	if l > 1 {
		return "phrase"
	}
	switch {
	case wl >= 10:
		return "GRE/Advanced"
	case wl >= 7:
		return "IELTS/TOEFL"
	case wl >= 5:
		return "intermediate"
	default:
		return "common"
	}
}
