package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/umang/mean-cli/internal/models"
)

const freeDictBaseURL = "https://api.dictionaryapi.dev/api/v2/entries/en/"

// freeDictEntry mirrors the Free Dictionary API JSON shape.
type freeDictEntry struct {
	Word      string `json:"word"`
	Phonetics []struct {
		Text  string `json:"text"`
		Audio string `json:"audio"`
	} `json:"phonetics"`
	Meanings []struct {
		PartOfSpeech string `json:"partOfSpeech"`
		Definitions  []struct {
			Definition string `json:"definition"`
			Example    string `json:"example"`
			Synonyms   []string `json:"synonyms"`
			Antonyms   []string `json:"antonyms"`
		} `json:"definitions"`
		Synonyms []string `json:"synonyms"`
		Antonyms []string `json:"antonyms"`
	} `json:"meanings"`
	Etymology string `json:"origin"`
}

// LookupFreeDictionary fetches word data from dictionaryapi.dev.
func LookupFreeDictionary(word string) (*models.Word, error) {
	encoded := url.PathEscape(strings.ToLower(strings.TrimSpace(word)))
	reqURL := freeDictBaseURL + encoded

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("word %q not found", word)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error: HTTP %d", resp.StatusCode)
	}

	var entries []freeDictEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no data returned for %q", word)
	}

	return convertFreeDictToWord(entries[0]), nil
}

func convertFreeDictToWord(e freeDictEntry) *models.Word {
	w := &models.Word{
		Word:      e.Word,
		Etymology: e.Etymology,
		Source:    "api",
		CachedAt:  time.Now(),
	}

	// Phonetics
	for _, p := range e.Phonetics {
		if p.Text != "" {
			if w.Pronunciation == "" {
				w.Pronunciation = p.Text
			}
			w.Phonetics = append(w.Phonetics, models.Phonetic{
				Text:  p.Text,
				Audio: p.Audio,
			})
		}
	}

	// Deduplicate sets
	synSeen := map[string]bool{}
	antSeen := map[string]bool{}
	exSeen := map[string]bool{}

	for _, m := range e.Meanings {
		for _, d := range m.Definitions {
			w.Definitions = append(w.Definitions, models.Definition{
				PartOfSpeech: m.PartOfSpeech,
				Meaning:      d.Definition,
				Example:      d.Example,
			})
			if d.Example != "" && !exSeen[d.Example] {
				w.Examples = append(w.Examples, d.Example)
				exSeen[d.Example] = true
			}
			for _, s := range d.Synonyms {
				if !synSeen[s] {
					w.Synonyms = append(w.Synonyms, s)
					synSeen[s] = true
				}
			}
			for _, a := range d.Antonyms {
				if !antSeen[a] {
					w.Antonyms = append(w.Antonyms, a)
					antSeen[a] = true
				}
			}
		}
		for _, s := range m.Synonyms {
			if !synSeen[s] {
				w.Synonyms = append(w.Synonyms, s)
				synSeen[s] = true
			}
		}
		for _, a := range m.Antonyms {
			if !antSeen[a] {
				w.Antonyms = append(w.Antonyms, a)
				antSeen[a] = true
			}
		}
	}

	return w
}
