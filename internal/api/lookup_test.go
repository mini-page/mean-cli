package api

import (
	"testing"
)

func TestGuessExamLevel(t *testing.T) {
	tests := []struct {
		word     string
		expected string
	}{
		{"quixotic", "IELTS/TOEFL"},
		{"cat", "common"},
		{"obsequious", "GRE/Advanced"},
		{"artificial intelligence", "phrase"},
		{"house", "intermediate"},
	}

	for _, tc := range tests {
		actual := guessExamLevel(tc.word)
		if actual != tc.expected {
			t.Errorf("guessExamLevel(%q) = %q; expected %q", tc.word, actual, tc.expected)
		}
	}
}
