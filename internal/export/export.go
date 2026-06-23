package export

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/umang/mean-cli/internal/models"
)

// ToMarkdown renders a Word as a Markdown document.
func ToMarkdown(w *models.Word) string {
	var sb strings.Builder

	sb.WriteString("# " + w.Word + "\n\n")

	if w.Pronunciation != "" {
		sb.WriteString("**Pronunciation:** `" + w.Pronunciation + "`\n\n")
	}

	if w.ExamLevel != "" {
		sb.WriteString("**Exam Level:** " + w.ExamLevel + "\n\n")
	}

	if len(w.Definitions) > 0 {
		sb.WriteString("## Definitions\n\n")
		lastPOS := ""
		for i, d := range w.Definitions {
			if d.PartOfSpeech != lastPOS {
				lastPOS = d.PartOfSpeech
				sb.WriteString("\n**" + d.PartOfSpeech + "**\n\n")
			}
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, d.Meaning))
			if d.Example != "" {
				sb.WriteString("   > *\"" + d.Example + "\"*\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(w.Synonyms) > 0 {
		sb.WriteString("## Synonyms\n\n")
		sb.WriteString(strings.Join(w.Synonyms, ", ") + "\n\n")
	}

	if len(w.Antonyms) > 0 {
		sb.WriteString("## Antonyms\n\n")
		sb.WriteString(strings.Join(w.Antonyms, ", ") + "\n\n")
	}

	if len(w.Examples) > 0 {
		sb.WriteString("## Examples\n\n")
		for i, ex := range w.Examples {
			sb.WriteString(fmt.Sprintf("%d. *\"%s\"*\n", i+1, ex))
		}
		sb.WriteString("\n")
	}

	if w.Etymology != "" {
		sb.WriteString("## Etymology\n\n")
		sb.WriteString(w.Etymology + "\n\n")
	}

	sb.WriteString("---\n*Exported by mean-cli on " + time.Now().Format("2006-01-02 15:04") + "*\n")
	return sb.String()
}

// ToText renders a Word as plain text.
func ToText(w *models.Word) string {
	var sb strings.Builder
	sep := strings.Repeat("─", 60)

	sb.WriteString(sep + "\n")
	sb.WriteString(strings.ToUpper(w.Word) + "\n")
	if w.Pronunciation != "" {
		sb.WriteString(w.Pronunciation + "\n")
	}
	sb.WriteString(sep + "\n\n")

	if len(w.Definitions) > 0 {
		sb.WriteString("DEFINITIONS\n\n")
		lastPOS := ""
		for i, d := range w.Definitions {
			if d.PartOfSpeech != lastPOS {
				lastPOS = d.PartOfSpeech
				sb.WriteString("\n[" + strings.ToUpper(d.PartOfSpeech) + "]\n")
			}
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i+1, d.Meaning))
			if d.Example != "" {
				sb.WriteString("     \"" + d.Example + "\"\n")
			}
		}
		sb.WriteString("\n")
	}

	if len(w.Synonyms) > 0 {
		sb.WriteString("SYNONYMS\n  " + strings.Join(w.Synonyms, ", ") + "\n\n")
	}

	if len(w.Antonyms) > 0 {
		sb.WriteString("ANTONYMS\n  " + strings.Join(w.Antonyms, ", ") + "\n\n")
	}

	if len(w.Examples) > 0 {
		sb.WriteString("EXAMPLES\n")
		for i, ex := range w.Examples {
			sb.WriteString(fmt.Sprintf("  %d. \"%s\"\n", i+1, ex))
		}
		sb.WriteString("\n")
	}

	if w.Etymology != "" {
		sb.WriteString("ETYMOLOGY\n  " + w.Etymology + "\n\n")
	}

	sb.WriteString(sep + "\n")
	sb.WriteString("Exported by mean-cli — " + time.Now().Format("2006-01-02 15:04") + "\n")
	return sb.String()
}

// SaveToFile writes content to a file, printing the path on success.
func SaveToFile(filename, content string) error {
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("  ✓ Saved to:", filename)
	fmt.Println()
	return nil
}
