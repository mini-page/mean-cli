package models

import "time"

// Definition holds a single meaning entry.
type Definition struct {
	PartOfSpeech string `json:"partOfSpeech"`
	Meaning      string `json:"meaning"`
	Example      string `json:"example"`
}

// Phonetic holds pronunciation data.
type Phonetic struct {
	Text  string `json:"text"`
	Audio string `json:"audio"`
}

// Word is the core domain struct used throughout the app.
type Word struct {
	Word          string       `json:"word"`
	Pronunciation string       `json:"pronunciation"`
	Phonetics     []Phonetic   `json:"phonetics"`
	Definitions   []Definition `json:"definitions"`
	Synonyms      []string     `json:"synonyms"`
	Antonyms      []string     `json:"antonyms"`
	Examples      []string     `json:"examples"`
	Etymology     string       `json:"etymology"`
	ExamLevel     string       `json:"examLevel"` // e.g. "GRE", "common"
	CachedAt      time.Time    `json:"cachedAt"`
	Source        string       `json:"source"` // "api" | "cache"
}
