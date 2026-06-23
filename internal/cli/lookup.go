package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/api"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/models"
)

// PrintWord renders a word result to stdout in CLI (non-TUI) mode.
func PrintWord(w *models.Word) {
	width := 72

	// ─── Header ────────────────────────────────────────────────────────────────
	sep := strings.Repeat("─", width)
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true), "📖  "+strings.ToUpper(w.Word))
	if w.Pronunciation != "" {
		printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Italic(true), "    "+w.Pronunciation)
	}
	if w.ExamLevel != "" {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1F1B4E")).
			Background(lipgloss.Color("#A78BFA")).
			Padding(0, 1).Bold(true).Render(w.ExamLevel)
		fmt.Println("    " + badge)
	}
	if w.Source == "cache" {
		printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true), "    (served from cache)")
	}
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(sep))

	// ─── Definitions ───────────────────────────────────────────────────────────
	sectionHeader("Definitions")
	if len(w.Definitions) == 0 {
		printMuted("  No definitions found.")
	}
	lastPOS := ""
	for i, d := range w.Definitions {
		if d.PartOfSpeech != lastPOS {
			lastPOS = d.PartOfSpeech
			printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true), "\n  "+d.PartOfSpeech)
		}
		fmt.Printf("  %d. %s\n", i+1, d.Meaning)
		if d.Example != "" {
			printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Italic(true), "     \""+d.Example+"\"")
		}
	}

	// ─── Synonyms ──────────────────────────────────────────────────────────────
	if len(w.Synonyms) > 0 {
		sectionHeader("Synonyms")
		printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")), "  "+strings.Join(w.Synonyms, "  ·  "))
	}

	// ─── Antonyms ──────────────────────────────────────────────────────────────
	if len(w.Antonyms) > 0 {
		sectionHeader("Antonyms")
		printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")), "  "+strings.Join(w.Antonyms, "  ·  "))
	}

	// ─── Examples ──────────────────────────────────────────────────────────────
	if len(w.Examples) > 0 {
		sectionHeader("Examples")
		for i, ex := range w.Examples {
			printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Italic(true),
				fmt.Sprintf("  %d. \"%s\"", i+1, ex))
		}
	}

	// ─── Etymology ─────────────────────────────────────────────────────────────
	if w.Etymology != "" {
		sectionHeader("Etymology")
		printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")), "  "+w.Etymology)
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(sep))
	fmt.Println()
}

// LookupAndPrint performs a full CLI lookup with cache support.
func LookupAndPrint(db *cache.DB, word string) {
	// Try cache
	cached, err := db.GetWord(word)
	if err == nil && cached != nil {
		_ = db.AddHistory(word)
		PrintWord(cached)
		return
	}

	fmt.Fprintf(os.Stderr, "\r  Looking up %q…", word)
	w, err := api.Lookup(word)
	fmt.Fprintf(os.Stderr, "\r                          \r")

	if err != nil {
		printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Bold(true),
			"\n  ✗ "+err.Error())
		fmt.Println()
		os.Exit(1)
	}

	_ = db.SaveWord(w)
	_ = db.AddHistory(word)
	PrintWord(w)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func sectionHeader(title string) {
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true), "  "+title)
}

func printStyled(style lipgloss.Style, text string) {
	fmt.Println(style.Render(text))
}

func printMuted(text string) {
	printStyled(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")), text)
}
