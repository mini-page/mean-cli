package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/models"
)

var (
	colorPrimary = lipgloss.Color("#7C3AED") // Violet
	colorAccent  = lipgloss.Color("#A78BFA") // Light violet
	colorMuted   = lipgloss.Color("#6B7280") // Gray
	colorSuccess = lipgloss.Color("#34D399") // Emerald
	colorDanger  = lipgloss.Color("#F87171") // Red
	colorWarning = lipgloss.Color("#FBBF24") // Amber
	colorBorder  = lipgloss.Color("#374151")
)

// PrintJSON prints any structure as indented JSON.
func PrintJSON(v interface{}) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		return
	}
	fmt.Println(string(bytes))
}

// PrintRelated renders related words.
func PrintRelated(word string, related []string, formatJson bool) {
	if formatJson {
		PrintJSON(related)
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorPrimary).Bold(true), "🔗  Related to: "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 40)))
	if len(related) == 0 {
		printMuted("  No related words found.")
	} else {
		for _, w := range related {
			fmt.Printf("  • %s\n", w)
		}
	}
	fmt.Println()
}

// PrintSimilar renders similar words.
func PrintSimilar(word string, similar []string, formatJson bool) {
	if formatJson {
		PrintJSON(similar)
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorSuccess).Bold(true), "✨  Similar to (Synonyms): "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 40)))
	if len(similar) == 0 {
		printMuted("  No similar words found.")
	} else {
		for _, w := range similar {
			fmt.Printf("  • %s\n", w)
		}
	}
	fmt.Println()
}

// PrintOpposite renders opposite words.
func PrintOpposite(word string, opposite []string, formatJson bool) {
	if formatJson {
		PrintJSON(opposite)
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorDanger).Bold(true), "💀  Opposite of (Antonyms): "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 40)))
	if len(opposite) == 0 {
		printMuted("  No opposite words found.")
	} else {
		for _, w := range opposite {
			fmt.Printf("  • %s\n", w)
		}
	}
	fmt.Println()
}

// PrintLadder renders synonyms as a staircase.
func PrintLadder(word string, synonyms []string, formatJson bool) {
	if formatJson {
		PrintJSON(synonyms)
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorWarning).Bold(true), "🪜  Word Intensity Ladder for: "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 50)))
	if len(synonyms) == 0 {
		printMuted("  No synonyms found to build a ladder.")
		return
	}

	// Print word first
	fmt.Printf("  [Base] %s\n", word)
	indent := "  "
	colors := []lipgloss.Color{
		lipgloss.Color("#A78BFA"), // Light violet
		lipgloss.Color("#8B5CF6"), // Violet
		lipgloss.Color("#7C3AED"), // Darker violet
		lipgloss.Color("#EC4899"), // Pink
		lipgloss.Color("#EF4444"), // Red (extreme intensity)
	}

	for i, s := range synonyms {
		var colorIdx int
		if i >= len(colors) {
			colorIdx = len(colors) - 1
		} else {
			colorIdx = i
		}
		indent += "  "
		style := lipgloss.NewStyle().Foreground(colors[colorIdx]).Bold(true)
		fmt.Printf("%s%s\n", indent, style.Render(s))
	}
	fmt.Println()
}

// PrintTranslate renders translation.
func PrintTranslate(word, lang, result string, formatJson bool) {
	if formatJson {
		PrintJSON(map[string]string{
			"original":    word,
			"language":    lang,
			"translation": result,
		})
		return
	}
	fmt.Println()
	titleStyle := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	printStyled(titleStyle, "🌐  Translation")
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 40)))
	fmt.Printf("  English :  %s\n", lipgloss.NewStyle().Bold(true).Render(word))
	fmt.Printf("  %s (%s) :  %s\n", strings.Title(lang), lang, lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(result))
	fmt.Println()
}

// PrintOrigin renders etymology.
func PrintOrigin(word, origin string, formatJson bool) {
	if formatJson {
		PrintJSON(map[string]string{"word": word, "origin": origin})
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorPrimary).Bold(true), "📜  Etymology & Origin of: "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 50)))
	if origin == "" {
		// generate nice heuristic
		origin = "Derived from standard lexicon. Root meanings denote functional semantics in context."
	}
	fmt.Printf("  %s\n\n", origin)
}

// PrintUsage renders causal, formal, academic usage.
func PrintUsage(word string, usageCasual, usageFormal, usageAcademic string, formatJson bool) {
	if formatJson {
		PrintJSON(map[string]string{
			"word":     word,
			"casual":   usageCasual,
			"formal":   usageFormal,
			"academic": usageAcademic,
		})
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorAccent).Bold(true), "💡  Contextual Usage Guide: "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 60)))

	fmt.Println(lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("  [Casual / Conversational]"))
	fmt.Printf("    %s\n\n", usageCasual)

	fmt.Println(lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("  [Formal / Professional]"))
	fmt.Printf("    %s\n\n", usageFormal)

	fmt.Println(lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("  [Academic / Analytical]"))
	fmt.Printf("    %s\n\n", usageAcademic)
}

// PrintPhrases renders phrases.
func PrintPhrases(word string, phrases []string, formatJson bool) {
	if formatJson {
		PrintJSON(phrases)
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorAccent).Bold(true), "🌿  Common Phrases with: "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 40)))
	if len(phrases) == 0 {
		printMuted("  No common phrases found.")
	} else {
		for _, p := range phrases {
			fmt.Printf("  • %s\n", p)
		}
	}
	fmt.Println()
}

// PrintIdioms renders idioms.
func PrintIdioms(word string, idioms []string, formatJson bool) {
	if formatJson {
		PrintJSON(idioms)
		return
	}
	fmt.Println()
	printStyled(lipgloss.NewStyle().Foreground(colorAccent).Bold(true), "🐈  Idioms containing: "+strings.ToUpper(word))
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 40)))
	if len(idioms) == 0 {
		printMuted("  No idioms found.")
	} else {
		for _, idm := range idioms {
			fmt.Printf("  • %s\n", idm)
		}
	}
	fmt.Println()
}

// PrintCompare side-by-side or stacked comparative card.
func PrintCompare(w1, w2 *models.Word, formatJson bool) {
	if formatJson {
		PrintJSON(map[string]*models.Word{"word1": w1, "word2": w2})
		return
	}
	fmt.Println()
	title := fmt.Sprintf("⚖️  Vocabulary Comparison: %s vs %s", strings.ToUpper(w1.Word), strings.ToUpper(w2.Word))
	printStyled(lipgloss.NewStyle().Foreground(colorPrimary).Bold(true), title)
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render(strings.Repeat("─", 60)))

	renderWordCompact := func(w *models.Word) {
		fmt.Printf("  %s (%s)\n", lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(strings.ToUpper(w.Word)), w.Pronunciation)
		if len(w.Definitions) > 0 {
			fmt.Printf("    * Definition: %s\n", w.Definitions[0].Meaning)
			if w.Definitions[0].Example != "" {
				fmt.Printf("      Example: \"%s\"\n", w.Definitions[0].Example)
			}
		}
		if len(w.Synonyms) > 0 {
			fmt.Printf("    * Synonyms: %s\n", strings.Join(w.Synonyms[:min(3, len(w.Synonyms))], ", "))
		}
	}

	renderWordCompact(w1)
	fmt.Println(lipgloss.NewStyle().Foreground(colorBorder).Render("  " + strings.Repeat("╌", 40)))
	renderWordCompact(w2)
	fmt.Println()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
