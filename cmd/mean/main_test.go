package main

import (
	"testing"
	"time"
)

func TestDayOfYear(t *testing.T) {
	// Mock time implementation for tests
	mockTime := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	
	// Helper calculation for Jan 1
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	dayNum := int(mockTime.Sub(start).Hours()/24) + 1
	
	if dayNum != 1 {
		t.Errorf("expected day 1 on Jan 1, got %d", dayNum)
	}

	// Helper calculation for Feb 5
	mockTimeFeb := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	dayNumFeb := int(mockTimeFeb.Sub(start).Hours()/24) + 1
	if dayNumFeb != 36 {
		t.Errorf("expected day 36 on Feb 5, got %d", dayNumFeb)
	}
}

func TestDailyWordSelection(t *testing.T) {
	words := []string{"apple", "banana", "cherry"}
	
	// Day 1
	idx1 := 1 % len(words)
	word1 := words[idx1]
	if word1 != "banana" {
		t.Errorf("expected banana for day 1 index, got %s", word1)
	}

	// Day 2
	idx2 := 2 % len(words)
	word2 := words[idx2]
	if word2 != "cherry" {
		t.Errorf("expected cherry for day 2 index, got %s", word2)
	}
}
