package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/cache"
)

// PrintHistory prints recent lookup history.
func PrintHistory(db *cache.DB) {
	words, err := db.GetHistory(50)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error fetching history:", err)
		os.Exit(1)
	}
	if len(words) == 0 {
		printMuted("\n  No history yet. Look up a word to get started.\n")
		return
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	entry := lipgloss.NewStyle().Foreground(lipgloss.Color("#F3F4F6"))
	num := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	fmt.Println()
	fmt.Println(title.Render("  📜  Search History"))
	fmt.Println()
	for i, w := range words {
		fmt.Printf("%s  %s\n",
			num.Render(fmt.Sprintf("  %2d.", i+1)),
			entry.Render(w),
		)
	}
	fmt.Println()
}

// PrintFavorites prints all starred words.
func PrintFavorites(db *cache.DB) {
	words, err := db.GetFavorites()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error fetching favorites:", err)
		os.Exit(1)
	}
	if len(words) == 0 {
		printMuted("\n  No favorites yet. Use 'mean star <word>' to add one.\n")
		return
	}

	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	entry := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
	num := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	fmt.Println()
	fmt.Println(title.Render("  ★  Favorites"))
	fmt.Println()
	for i, w := range words {
		fmt.Printf("%s  %s\n",
			num.Render(fmt.Sprintf("  %2d.", i+1)),
			entry.Render(w),
		)
	}
	fmt.Println()
}

// StarWord adds or removes a word from favorites.
func StarWord(db *cache.DB, word string) {
	word = strings.TrimSpace(strings.ToLower(word))
	isFav, err := db.IsFavorite(word)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	star := lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	if isFav {
		_ = db.RemoveFavorite(word)
		fmt.Println(muted.Render("\n  ☆  Removed ") + word + muted.Render(" from favorites\n"))
	} else {
		_ = db.AddFavorite(word)
		fmt.Println(star.Render("\n  ★  Added ") + word + star.Render(" to favorites\n"))
	}
}
