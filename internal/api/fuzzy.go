package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Levenshtein calculates the edit distance between two strings.
func Levenshtein(s1, s2 string) int {
	s1 = strings.ToLower(strings.TrimSpace(s1))
	s2 = strings.ToLower(strings.TrimSpace(s2))
	if s1 == s2 {
		return 0
	}
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	x := make([]int, len(s2)+1)
	for i := range x {
		x[i] = i
	}

	for i := 0; i < len(s1); i++ {
		prev := i + 1
		for j := 0; j < len(s2); j++ {
			val := x[j]
			if s1[i] != s2[j] {
				val++
			}
			sub := val
			ins := x[j+1] + 1
			del := prev + 1

			min := sub
			if ins < min {
				min = ins
			}
			if del < min {
				min = del
			}
			x[j], prev = prev, min
		}
		x[len(s2)] = prev
	}
	return x[len(s2)]
}

type suggestionEntry struct {
	Word string `json:"word"`
}

// FetchSuggestions queries Datamuse autocomplete API for spelling suggestions.
func FetchSuggestions(word string) ([]string, error) {
	word = strings.ToLower(strings.TrimSpace(word))
	reqURL := "https://api.datamuse.com/sug?s=" + url.QueryEscape(word) + "&max=5"

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var results []suggestionEntry
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, err
	}

	var suggestions []string
	seen := map[string]bool{}
	for _, r := range results {
		w := strings.ToLower(strings.TrimSpace(r.Word))
		if w != "" && w != word && !seen[w] {
			suggestions = append(suggestions, r.Word)
			seen[w] = true
		}
	}
	return suggestions, nil
}
