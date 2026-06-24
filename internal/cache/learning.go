package cache

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// LearningWord represents a word tracked in the spaced repetition system.
type LearningWord struct {
	Word         string     `json:"word"`
	Box          int        `json:"box"`
	NextReviewAt time.Time  `json:"nextReviewAt"`
	IntervalDays int        `json:"intervalDays"`
	LastReviewed *time.Time `json:"lastReviewed"`
}

// AddLearningWord initializes tracking for a word in Box 1.
func (db *DB) AddLearningWord(word string) error {
	word = strings.ToLower(strings.TrimSpace(word))
	if word == "" {
		return nil
	}
	_, err := db.db.Exec(
		`INSERT OR IGNORE INTO learning_words (word, box, next_review_at, interval_days) VALUES (?, 1, datetime('now'), 1)`,
		word,
	)
	return err
}

// GetLearningWord retrieves tracking metadata for a word.
func (db *DB) GetLearningWord(word string) (*LearningWord, error) {
	word = strings.ToLower(strings.TrimSpace(word))
	row := db.db.QueryRow(
		`SELECT word, box, next_review_at, interval_days, last_reviewed FROM learning_words WHERE word = ?`,
		word,
	)
	var lw LearningWord
	var lastReviewed sql.NullTime
	var nextReviewStr string
	err := row.Scan(&lw.Word, &lw.Box, &nextReviewStr, &lw.IntervalDays, &lastReviewed)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	// Try parsing nextReviewStr (sqlite storage format)
	if t, terr := time.Parse("2006-01-02 15:04:05", nextReviewStr); terr == nil {
		lw.NextReviewAt = t
	} else if t, terr := time.Parse(time.RFC3339, nextReviewStr); terr == nil {
		lw.NextReviewAt = t
	} else {
		lw.NextReviewAt = time.Now()
	}

	if lastReviewed.Valid {
		lw.LastReviewed = &lastReviewed.Time
	}
	return &lw, nil
}

// UpdateLearningWord updates Leitner box and scheduling information.
func (db *DB) UpdateLearningWord(word string, box int, nextReview time.Time, intervalDays int) error {
	word = strings.ToLower(strings.TrimSpace(word))
	var lastReviewed interface{} = time.Now()
	_, err := db.db.Exec(
		`INSERT INTO learning_words (word, box, next_review_at, interval_days, last_reviewed)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(word) DO UPDATE SET box=excluded.box, next_review_at=excluded.next_review_at, interval_days=excluded.interval_days, last_reviewed=excluded.last_reviewed`,
		word, box, nextReview.Format("2006-01-02 15:04:05"), intervalDays, lastReviewed,
	)
	return err
}

// GetDueWords returns words that are due for SRM review.
func (db *DB) GetDueWords() ([]string, error) {
	rows, err := db.db.Query(
		`SELECT word FROM learning_words WHERE next_review_at <= datetime('now') ORDER BY next_review_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}

// GetAllLearningWords returns all words currently in the learning pipeline.
func (db *DB) GetAllLearningWords() ([]string, error) {
	rows, err := db.db.Query(
		`SELECT word FROM learning_words ORDER BY word ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		words = append(words, w)
	}
	return words, rows.Err()
}

// GetLearningStats gathers all metrics.
func (db *DB) GetLearningStats() (learnedCount, favoritesCount, masteredCount int, currentStreak, maxStreak int, topCategory string, err error) {
	// favoritesCount
	row := db.db.QueryRow(`SELECT COUNT(*) FROM favorites`)
	_ = row.Scan(&favoritesCount)

	// learnedCount: box > 1 or mastered
	row = db.db.QueryRow(`SELECT COUNT(*) FROM learning_words WHERE box > 1`)
	_ = row.Scan(&learnedCount)

	// masteredCount: box = 5
	row = db.db.QueryRow(`SELECT COUNT(*) FROM learning_words WHERE box = 5`)
	_ = row.Scan(&masteredCount)

	// streaks
	cStr, _ := db.GetSetting("streak_current")
	mStr, _ := db.GetSetting("streak_max")
	currentStreak, _ = strconv.Atoi(cStr)
	maxStreak, _ = strconv.Atoi(mStr)

	// topCategory
	rows, err := db.db.Query(`SELECT data FROM words`)
	if err == nil {
		defer rows.Close()
		catCounts := map[string]int{}
		for rows.Next() {
			var data string
			if err := rows.Scan(&data); err == nil {
				if strings.Contains(data, `"examLevel":"GRE/Advanced"`) {
					catCounts["GRE/Advanced"]++
				} else if strings.Contains(data, `"examLevel":"IELTS/TOEFL"`) {
					catCounts["IELTS/TOEFL"]++
				} else if strings.Contains(data, `"examLevel":"intermediate"`) {
					catCounts["Intermediate"]++
				} else if strings.Contains(data, `"examLevel":"common"`) {
					catCounts["Common"]++
				}
			}
		}
		maxCat := "General Vocabulary"
		maxCount := 0
		for c, count := range catCounts {
			if count > maxCount {
				maxCount = count
				maxCat = c
			}
		}
		topCategory = maxCat
	} else {
		topCategory = "General Vocabulary"
	}

	return
}

// UpdateStreak increments or resets the user's daily usage streak.
func (db *DB) UpdateStreak() (int, int, error) {
	today := time.Now().Format("2006-01-02")
	lastDate, err := db.GetSetting("streak_last_date")
	if err != nil {
		return 0, 0, err
	}

	cStr, _ := db.GetSetting("streak_current")
	mStr, _ := db.GetSetting("streak_max")
	curr, _ := strconv.Atoi(cStr)
	maxStreak, _ := strconv.Atoi(mStr)

	if lastDate == "" {
		curr = 1
		maxStreak = 1
		_ = db.SetSetting("streak_current", "1")
		_ = db.SetSetting("streak_max", "1")
		_ = db.SetSetting("streak_last_date", today)
	} else if lastDate == today {
		// already active today, no streak changes
	} else {
		// check if it was yesterday
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if lastDate == yesterday {
			curr++
			if curr > maxStreak {
				maxStreak = curr
				_ = db.SetSetting("streak_max", strconv.Itoa(maxStreak))
			}
			_ = db.SetSetting("streak_current", strconv.Itoa(curr))
			_ = db.SetSetting("streak_last_date", today)
		} else {
			// streak broken
			curr = 1
			_ = db.SetSetting("streak_current", "1")
			_ = db.SetSetting("streak_last_date", today)
		}
	}
	return curr, maxStreak, nil
}
