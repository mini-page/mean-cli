package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const datamuseBaseURL = "https://api.datamuse.com/words"

type datamuseWord struct {
	Word  string `json:"word"`
	Score int    `json:"score"`
}

// FetchSynonyms fetches synonyms from Datamuse.
func FetchSynonyms(word string) ([]string, error) {
	return fetchRelated("rel_syn", word)
}

// FetchAntonyms fetches antonyms from Datamuse.
func FetchAntonyms(word string) ([]string, error) {
	return fetchRelated("rel_ant", word)
}

// FetchRelated fetches general related words from Datamuse using 'ml' (means like).
func FetchRelated(word string) ([]string, error) {
	return fetchRelated("ml", word)
}

// FetchSimilar fetches similar words (synonyms) from Datamuse.
func FetchSimilar(word string) ([]string, error) {
	return fetchRelated("rel_syn", word)
}

// FetchOpposite fetches opposite words (antonyms) from Datamuse.
func FetchOpposite(word string) ([]string, error) {
	return fetchRelated("rel_ant", word)
}

// FetchPhrases finds phrases/compounds containing the word.
func FetchPhrases(word string) ([]string, error) {
	word = strings.ToLower(strings.TrimSpace(word))
	
	// Query Datamuse ml to get related multi-word entries
	params := url.Values{}
	params.Set("ml", word)
	params.Set("max", "40")

	results, err := queryDatamuse(params)
	if err != nil {
		return nil, err
	}

	var phrases []string
	seen := map[string]bool{}
	for _, r := range results {
		w := strings.ToLower(r.Word)
		// Check if it's a multi-word phrase containing our word
		if strings.Contains(w, " ") && strings.Contains(w, word) && !seen[w] {
			phrases = append(phrases, r.Word)
			seen[w] = true
		}
	}

	// Fallback to spelling-pattern query if we didn't find many phrases
	if len(phrases) < 3 {
		params = url.Values{}
		params.Set("sp", word+" *")
		params.Set("max", "20")
		res2, _ := queryDatamuse(params)
		for _, r := range res2 {
			w := strings.ToLower(r.Word)
			if strings.Contains(w, " ") && !seen[w] {
				phrases = append(phrases, r.Word)
				seen[w] = true
			}
		}
	}

	return phrases, nil
}

// FetchIdioms finds idioms or popular expressions containing the word.
func FetchIdioms(word string) ([]string, error) {
	word = strings.ToLower(strings.TrimSpace(word))
	
	// Predefined high-quality idioms for common words
	var fallbackIdioms = map[string][]string{
		"cat":     {"let the cat out of the bag", "curiosity killed the cat", "rain cats and dogs", "cat nap", "play cat and mouse"},
		"dog":     {"barking up the wrong tree", "every dog has its day", "dog eat dog", "let sleeping dogs lie", "work like a dog"},
		"rain":    {"rain cats and dogs", "right as rain", "save for a rainy day", "come rain or shine", "take a rain check"},
		"break":   {"break the ice", "break a leg", "break ground", "break even", "break down the door"},
		"courage": {"screw your courage to the sticking place", "take courage", "have the courage of one's convictions"},
		"irony":   {"dramatic irony", "situational irony", "verbal irony", "life's irony"},
		"paradox": {"liar paradox", "bootstrap paradox", "fermi paradox", "grandparent paradox"},
		"recursion": {"recursion relation", "tail recursion", "infinite recursion", "recursive call"},
		"loop":    {"loop hole", "in the loop", "throw for a loop", "loop back"},
	}

	if idioms, ok := fallbackIdioms[word]; ok {
		return idioms, nil
	}

	// Query Datamuse for phrases of length >= 3 containing the word
	params := url.Values{}
	params.Set("ml", word)
	params.Set("max", "60")
	results, err := queryDatamuse(params)
	if err != nil {
		return nil, err
	}

	var idioms []string
	seen := map[string]bool{}
	for _, r := range results {
		w := strings.ToLower(r.Word)
		// We define idiom heuristic as a multi-word phrase with 3 or more words
		parts := strings.Fields(w)
		if len(parts) >= 3 && strings.Contains(w, word) && !seen[w] {
			idioms = append(idioms, r.Word)
			seen[w] = true
		}
	}

	// If none found, query sp pattern * word *
	if len(idioms) == 0 {
		params = url.Values{}
		params.Set("sp", "* "+word+" *")
		params.Set("max", "30")
		res2, _ := queryDatamuse(params)
		for _, r := range res2 {
			w := strings.ToLower(r.Word)
			parts := strings.Fields(w)
			if len(parts) >= 3 && !seen[w] {
				idioms = append(idioms, r.Word)
				seen[w] = true
			}
		}
	}

	// Limit to 6 items max
	if len(idioms) > 6 {
		idioms = idioms[:6]
	}

	return idioms, nil
}

func fetchRelated(relation, word string) ([]string, error) {
	params := url.Values{}
	params.Set(relation, strings.ToLower(strings.TrimSpace(word)))
	params.Set("max", "10")

	results, err := queryDatamuse(params)
	if err != nil {
		return nil, err
	}

	words := make([]string, 0, len(results))
	for _, r := range results {
		words = append(words, r.Word)
	}
	return words, nil
}

func queryDatamuse(params url.Values) ([]datamuseWord, error) {
	reqURL := datamuseBaseURL + "?" + params.Encode()
	client := &http.Client{Timeout: 8 * time.Second}

	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("datamuse network error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("datamuse read error: %w", err)
	}

	var results []datamuseWord
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("datamuse parse error: %w", err)
	}

	return results, nil
}
