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

// GenerateUsageGuide creates casual, formal, academic example sentences for a word.
func GenerateUsageGuide(w *models.Word) (casual, formal, academic string) {
	var pos string
	if len(w.Definitions) > 0 {
		pos = strings.ToLower(w.Definitions[0].PartOfSpeech)
	}

	word := w.Word

	switch pos {
	case "adjective":
		casual = fmt.Sprintf("Honestly, that situation was completely %s.", word)
		formal = fmt.Sprintf("The department aims to maintain a %s environment during the transition period.", word)
		academic = fmt.Sprintf("Data indicates that the experimental group displayed a significantly %s response under controlled conditions.", word)
	case "noun":
		casual = fmt.Sprintf("I don't think I've ever encountered such a %s in my life.", word)
		formal = fmt.Sprintf("Our priority is to establish a robust %s to address these market changes.", word)
		academic = fmt.Sprintf("This paper examines the developmental trajectory of the %s under external environmental stressors.", word)
	case "verb":
		casual = fmt.Sprintf("It's hard to %s when everything is changing so fast.", word)
		formal = fmt.Sprintf("We must carefully %s the resources assigned to this quarterly project.", word)
		academic = fmt.Sprintf("Researchers hypothesize that these factors directly %s the system's overall equilibrium.", word)
	default:
		casual = fmt.Sprintf("I'm trying to figure out how to use '%s' naturally when hanging out with friends.", word)
		formal = fmt.Sprintf("Please review the guidelines to ensure correct application of '%s' in reports.", word)
		academic = fmt.Sprintf("The thematic relevance of '%s' remains a subject of intensive, ongoing scholarly debate.", word)
	}

	// Override with real API examples if available
	if len(w.Examples) > 0 {
		casual = w.Examples[0]
	}
	if len(w.Examples) > 1 {
		formal = w.Examples[1]
	}
	return
}

