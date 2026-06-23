package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/umang/mean-cli/internal/models"
	_ "modernc.org/sqlite"
)

// DB is the main cache database handle.
type DB struct {
	db *sql.DB
}

// DataDir returns the platform-appropriate config directory.
func DataDir() string {
	switch runtime.GOOS {
	case "windows":
		appdata := os.Getenv("APPDATA")
		if appdata != "" {
			return filepath.Join(appdata, "mean")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "AppData", "Roaming", "mean")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "mean")
	default:
		if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
			return filepath.Join(xdg, "mean")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "mean")
	}
}

// Open initialises (or opens) the SQLite database.
func Open() (*DB, error) {
	dir := DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("cannot create data dir: %w", err)
	}

	dbPath := filepath.Join(dir, "mean.db")
	sqlDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open database: %w", err)
	}

	// Enable WAL for better concurrent performance
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}

	db := &DB{db: sqlDB}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.db.Close()
}

// migrate creates all tables if they don't exist.
func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS words (
		word      TEXT PRIMARY KEY,
		data      TEXT NOT NULL,
		cached_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS history (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		word       TEXT NOT NULL,
		looked_at  DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS favorites (
		word     TEXT PRIMARY KEY,
		added_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := db.db.Exec(schema)
	return err
}

// ─── Word Cache ──────────────────────────────────────────────────────────────

// GetWord retrieves a cached word (returns nil if not found or expired).
func (db *DB) GetWord(word string) (*models.Word, error) {
	word = strings.ToLower(strings.TrimSpace(word))
	row := db.db.QueryRow(
		`SELECT data, cached_at FROM words WHERE word = ?`, word,
	)

	var data string
	var cachedAt time.Time
	if err := row.Scan(&data, &cachedAt); err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// Expire cache after 7 days
	if time.Since(cachedAt) > 7*24*time.Hour {
		_, _ = db.db.Exec(`DELETE FROM words WHERE word = ?`, word)
		return nil, nil
	}

	var w models.Word
	if err := json.Unmarshal([]byte(data), &w); err != nil {
		return nil, err
	}
	w.Source = "cache"
	return &w, nil
}

// SaveWord stores a word in the cache.
func (db *DB) SaveWord(w *models.Word) error {
	data, err := json.Marshal(w)
	if err != nil {
		return err
	}
	_, err = db.db.Exec(
		`INSERT OR REPLACE INTO words (word, data, cached_at) VALUES (?, ?, ?)`,
		strings.ToLower(w.Word), string(data), time.Now(),
	)
	return err
}

// ClearWordCache removes all cached words.
func (db *DB) ClearWordCache() error {
	_, err := db.db.Exec(`DELETE FROM words`)
	return err
}

// ─── History ─────────────────────────────────────────────────────────────────

// AddHistory records a word lookup.
func (db *DB) AddHistory(word string) error {
	_, err := db.db.Exec(
		`INSERT INTO history (word, looked_at) VALUES (?, ?)`,
		strings.ToLower(strings.TrimSpace(word)), time.Now(),
	)
	return err
}

// GetHistory returns the N most recent history entries.
func (db *DB) GetHistory(limit int) ([]string, error) {
	rows, err := db.db.Query(
		`SELECT word FROM history ORDER BY looked_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []string
	seen := map[string]bool{}
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			return nil, err
		}
		if !seen[w] {
			words = append(words, w)
			seen[w] = true
		}
	}
	return words, rows.Err()
}

// ClearHistory removes all history entries.
func (db *DB) ClearHistory() error {
	_, err := db.db.Exec(`DELETE FROM history`)
	return err
}

// ─── Favorites ───────────────────────────────────────────────────────────────

// AddFavorite adds a word to favorites.
func (db *DB) AddFavorite(word string) error {
	_, err := db.db.Exec(
		`INSERT OR IGNORE INTO favorites (word, added_at) VALUES (?, ?)`,
		strings.ToLower(strings.TrimSpace(word)), time.Now(),
	)
	return err
}

// RemoveFavorite removes a word from favorites.
func (db *DB) RemoveFavorite(word string) error {
	_, err := db.db.Exec(
		`DELETE FROM favorites WHERE word = ?`,
		strings.ToLower(strings.TrimSpace(word)),
	)
	return err
}

// IsFavorite reports whether a word is starred.
func (db *DB) IsFavorite(word string) (bool, error) {
	row := db.db.QueryRow(
		`SELECT 1 FROM favorites WHERE word = ?`,
		strings.ToLower(strings.TrimSpace(word)),
	)
	var v int
	err := row.Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// GetFavorites returns all favorited words.
func (db *DB) GetFavorites() ([]string, error) {
	rows, err := db.db.Query(
		`SELECT word FROM favorites ORDER BY added_at DESC`,
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

// ─── Settings ────────────────────────────────────────────────────────────────

// GetSetting retrieves a setting value.
func (db *DB) GetSetting(key string) (string, error) {
	row := db.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key)
	var val string
	if err := row.Scan(&val); err == sql.ErrNoRows {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return val, nil
}

// SetSetting upserts a setting value.
func (db *DB) SetSetting(key, value string) error {
	_, err := db.db.Exec(
		`INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)`, key, value,
	)
	return err
}
