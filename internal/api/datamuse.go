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

func fetchRelated(relation, word string) ([]string, error) {
	params := url.Values{}
	params.Set(relation, strings.ToLower(strings.TrimSpace(word)))
	params.Set("max", "10")

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

	words := make([]string, 0, len(results))
	for _, r := range results {
		words = append(words, r.Word)
	}
	return words, nil
}
