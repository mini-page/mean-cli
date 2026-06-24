package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/umang/mean-cli/internal/api"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/cli"
	"github.com/umang/mean-cli/internal/export"
	"github.com/umang/mean-cli/internal/models"
	"github.com/umang/mean-cli/internal/tui"
)

const version = "1.0.0"

func main() {
	db, err := cache.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	root := buildRoot(db)
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func isInputPiped() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func buildRoot(db *cache.DB) *cobra.Command {
	root := &cobra.Command{
		Use:   "mean [word]",
		Short: "📖 mean — a fast cross-platform dictionary CLI/TUI",
		Long: `mean is a fast, beautiful dictionary for your terminal.

  mean <word>        quick lookup
  mean               open interactive TUI
  mean history       show search history
  mean favorites     list starred words
  mean star <word>   toggle favorite
  mean --daily       word of the day
  mean --random      random word
  mean export md     export last result as Markdown
  mean export txt    export last result as plain text
  mean cache clear   clear local word cache`,
		Version: version,
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			daily, _ := cmd.Flags().GetBool("daily")
			random, _ := cmd.Flags().GetBool("random")
			formatJson, _ := cmd.Flags().GetBool("json")

			var inputWord string
			if isInputPiped() {
				var sb strings.Builder
				scanner := bufio.NewScanner(os.Stdin)
				for scanner.Scan() {
					sb.WriteString(scanner.Text())
				}
				inputWord = strings.TrimSpace(sb.String())
			}

			switch {
			case daily:
				return runDaily(db)
			case random:
				return runRandom(db)
			case inputWord != "":
				if formatJson {
					w, err := db.GetWord(inputWord)
					if err != nil || w == nil {
						w, err = api.Lookup(inputWord)
						if err != nil {
							cli.PrintJSON(map[string]string{"error": err.Error()})
							return nil
						}
						_ = db.SaveWord(w)
					}
					_ = db.AddHistory(inputWord)
					cli.PrintJSON(w)
					return nil
				}
				cli.LookupAndPrint(db, inputWord)
				return nil
			case len(args) > 0:
				word := strings.Join(args, " ")
				if formatJson {
					w, err := db.GetWord(word)
					if err != nil || w == nil {
						w, err = api.Lookup(word)
						if err != nil {
							cli.PrintJSON(map[string]string{"error": err.Error()})
							return nil
						}
						_ = db.SaveWord(w)
					}
					_ = db.AddHistory(word)
					cli.PrintJSON(w)
					return nil
				}
				cli.LookupAndPrint(db, word)
				return nil
			default:
				return runTUI(db)
			}
		},
	}

	root.Flags().Bool("daily", false, "Show word of the day")
	root.Flags().Bool("random", false, "Look up a random word")
	root.PersistentFlags().Bool("json", false, "Output results in JSON format")

	// Subcommands
	root.AddCommand(historyCmd(db))
	root.AddCommand(favoritesCmd(db))
	root.AddCommand(starCmd(db))
	root.AddCommand(exportCmd(db))
	root.AddCommand(cacheCmd(db))
	root.AddCommand(quizCmd(db))
	root.AddCommand(flashcardsCmd(db))
	root.AddCommand(gameCmd(db))

	// Core relationship commands
	root.AddCommand(relatedCmd(db))
	root.AddCommand(compareCmd(db))
	root.AddCommand(similarCmd(db))
	root.AddCommand(oppositeCmd(db))
	root.AddCommand(pronounceCmd(db))
	root.AddCommand(translateCmd(db))
	root.AddCommand(originCmd(db))
	root.AddCommand(usageCmd(db))
	root.AddCommand(examplesCmd(db))
	root.AddCommand(phraseCmd(db))
	root.AddCommand(idiomCmd(db))
	root.AddCommand(ladderCmd(db))
	root.AddCommand(domainCmd(db))

	// Learning OS commands
	root.AddCommand(learnCmd(db))
	root.AddCommand(reviewCmd(db))
	root.AddCommand(streakCmd(db))
	root.AddCommand(statsCmd(db))
	root.AddCommand(serveCmd(db))

	// Apply custom help recursively
	applyCustomHelp(root)

	return root
}

// ─── TUI ─────────────────────────────────────────────────────────────────────

func runTUI(db *cache.DB) error {
	deck := loadStudyDeck(db)
	m := tui.New(db, deck)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}

// ─── Daily / Random ───────────────────────────────────────────────────────────

var dailyWords = []string{
	"ephemeral", "serendipity", "melancholy", "paradigm", "eloquent",
	"resilience", "ambiguous", "benevolent", "cogent", "tenacious",
	"perspicacious", "sanguine", "loquacious", "pedantic", "ebullient",
	"recalcitrant", "obsequious", "pernicious", "vicarious", "solipsism",
}

func runDaily(db *cache.DB) error {
	// Use day-of-year to pick a consistent word each day
	idx := dayOfYear() % len(dailyWords)
	word := dailyWords[idx]
	fmt.Printf("\n  ✨ Word of the Day: %s\n\n", word)
	cli.LookupAndPrint(db, word)
	return nil
}

func runRandom(db *cache.DB) error {
	words := []string{
		"quixotic", "nefarious", "ethereal", "paradox", "stoic",
		"lament", "ephemeral", "verbose", "terse", "lucid",
		"cryptic", "fervent", "languid", "taciturn", "vindicate",
		"exacerbate", "ameliorate", "obfuscate", "corroborate", "elucidate",
	}
	// pseudo-random using seconds
	idx := int(nowSeconds()) % len(words)
	word := words[idx]
	fmt.Printf("\n  🎲 Random Word: %s\n\n", word)
	cli.LookupAndPrint(db, word)
	return nil
}

// ─── History ─────────────────────────────────────────────────────────────────

func historyCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show recent search history",
		RunE: func(cmd *cobra.Command, args []string) error {
			clear, _ := cmd.Flags().GetBool("clear")
			if clear {
				if err := db.ClearHistory(); err != nil {
					return err
				}
				fmt.Println()
				fmt.Println("  ✓ History cleared")
				fmt.Println()
				return nil
			}
			cli.PrintHistory(db)
			return nil
		},
	}
	cmd.Flags().Bool("clear", false, "Clear all history")
	return cmd
}

// ─── Favorites ───────────────────────────────────────────────────────────────

func favoritesCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "favorites",
		Aliases: []string{"fav", "favs"},
		Short:   "List starred/favorite words",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli.PrintFavorites(db)
			return nil
		},
	}
}

func starCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "star <word>",
		Short: "Toggle a word as favorite",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			cli.StarWord(db, word)
			return nil
		},
	}
}

// ─── Export ───────────────────────────────────────────────────────────────────

func exportCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [md|txt]",
		Short: "Export last looked-up word as Markdown or plain text",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format := strings.ToLower(args[0])

			// Fetch last history word
			hist, err := db.GetHistory(1)
			if err != nil || len(hist) == 0 {
				return fmt.Errorf("no recent word found — look up a word first")
			}
			word := hist[0]

			// Get from cache or API
			w, cerr := db.GetWord(word)
			if cerr != nil || w == nil {
				w, err = api.Lookup(word)
				if err != nil {
					return err
				}
			}

			switch format {
			case "md", "markdown":
				content := export.ToMarkdown(w)
				return export.SaveToFile(word+".md", content)
			case "txt", "text":
				content := export.ToText(w)
				return export.SaveToFile(word+".txt", content)
			default:
				return fmt.Errorf("unknown format %q — use 'md' or 'txt'", format)
			}
		},
	}
	return cmd
}

// ─── Cache ────────────────────────────────────────────────────────────────────

func cacheCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage local word cache",
	}

	clearSub := &cobra.Command{
		Use:   "clear",
		Short: "Clear all cached words",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := db.ClearWordCache(); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("  ✓ Word cache cleared")
			fmt.Println()
			return nil
		},
	}
	cmd.AddCommand(clearSub)

	pathSub := &cobra.Command{
		Use:   "path",
		Short: "Print data directory path",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println()
			fmt.Println("  Data directory:", cache.DataDir())
			fmt.Println()
		},
	}
	cmd.AddCommand(pathSub)

	return cmd
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func dayOfYear() int {
	now := timeNow()
	start := timeDate(now.Year(), 1, 1)
	return int(now.Sub(start).Hours()/24) + 1
}

func nowSeconds() int64 {
	return timeNow().Unix()
}

// ─── Quiz & Flashcard Commands ───────────────────────────────────────────────

func quizCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "quiz",
		Aliases: []string{"q"},
		Short:   "Start an interactive vocabulary quiz game",
		RunE: func(cmd *cobra.Command, args []string) error {
			useCli, _ := cmd.Flags().GetBool("cli")
			deck := loadStudyDeck(db)

			if useCli {
				cli.RunCliQuiz(deck)
				return nil
			}
			return tui.StartQuiz(db, deck)
		},
	}
	cmd.Flags().Bool("cli", false, "Run in text-interactive CLI mode")
	return cmd
}

func flashcardsCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "flashcards",
		Aliases: []string{"fc"},
		Short:   "Start an active recall flashcard session",
		RunE: func(cmd *cobra.Command, args []string) error {
			useCli, _ := cmd.Flags().GetBool("cli")
			deck := loadStudyDeck(db)

			if useCli {
				cli.RunCliFlashcards(db, deck)
				return nil
			}
			return tui.StartFlashcards(db, deck)
		},
	}
	cmd.Flags().Bool("cli", false, "Run in text-interactive CLI mode")
	return cmd
}

func loadStudyDeck(db *cache.DB) []models.Word {
	var deck []models.Word
	seen := map[string]bool{}

	// 1. Load favorites first
	favs, _ := db.GetFavorites()
	for _, w := range favs {
		if wordData, err := db.GetWord(w); err == nil && wordData != nil {
			deck = append(deck, *wordData)
			seen[strings.ToLower(w)] = true
		}
	}

	// 2. Load history next (limit to 30)
	hist, _ := db.GetHistory(30)
	for _, w := range hist {
		lowerW := strings.ToLower(w)
		if !seen[lowerW] {
			if wordData, err := db.GetWord(w); err == nil && wordData != nil {
				deck = append(deck, *wordData)
				seen[lowerW] = true
			}
		}
	}

	// 3. Fallback to curated words if deck is small
	if len(deck) < 5 {
		for _, cw := range curatedDeck {
			lowerW := strings.ToLower(cw.Word)
			if !seen[lowerW] {
				deck = append(deck, cw)
				seen[lowerW] = true
			}
		}
	}

	return deck
}

// ─── Game Commands ───────────────────────────────────────────────────────────

func gameCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "game [hangman|match|quiz|flashcards]",
		Short: "Play interactive word vocabulary games",
		Long: `Play vocabulary building games in TUI or CLI mode.

  mean game                Launch Unified Game Center TUI Dashboard
  mean game hangman        Launch Hangman TUI
  mean game hangman --cli  Launch Hangman CLI
  mean game match          Launch Definition Matcher TUI
  mean game match --cli    Launch Definition Matcher CLI
  mean game quiz           Launch Spelling Quiz TUI
  mean game quiz --cli     Launch Spelling Quiz CLI
  mean game flashcards     Launch Flashcards TUI
  mean game flashcards --cli Launch Flashcards CLI`,
		RunE: func(cmd *cobra.Command, args []string) error {
			deck := loadStudyDeck(db)
			return tui.StartGameCenter(db, deck)
		},
	}

	cmd.AddCommand(hangmanSubCmd(db))
	cmd.AddCommand(matchSubCmd(db))
	cmd.AddCommand(quizSubCmd(db))
	cmd.AddCommand(flashcardsSubCmd(db))

	return cmd
}

func quizSubCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "quiz",
		Aliases: []string{"q"},
		Short:   "Play Spelling Spelling Quiz game",
		RunE: func(cmd *cobra.Command, args []string) error {
			useCli, _ := cmd.Flags().GetBool("cli")
			deck := loadStudyDeck(db)

			if useCli {
				cli.RunCliQuiz(deck)
				return nil
			}
			return tui.StartQuiz(db, deck)
		},
	}
	cmd.Flags().Bool("cli", false, "Run in text-interactive CLI mode")
	return cmd
}

func flashcardsSubCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "flashcards",
		Aliases: []string{"fc"},
		Short:   "Play Active Recall Flashcards study session",
		RunE: func(cmd *cobra.Command, args []string) error {
			useCli, _ := cmd.Flags().GetBool("cli")
			deck := loadStudyDeck(db)

			if useCli {
				cli.RunCliFlashcards(db, deck)
				return nil
			}
			return tui.StartFlashcards(db, deck)
		},
	}
	cmd.Flags().Bool("cli", false, "Run in text-interactive CLI mode")
	return cmd
}

func hangmanSubCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "hangman",
		Aliases: []string{"h"},
		Short:   "Play Hangman vocabulary game",
		RunE: func(cmd *cobra.Command, args []string) error {
			useCli, _ := cmd.Flags().GetBool("cli")
			deck := loadStudyDeck(db)

			if useCli {
				cli.RunCliHangman(deck)
				return nil
			}
			return tui.StartHangman(db, deck)
		},
	}
	cmd.Flags().Bool("cli", false, "Run in text-interactive CLI mode")
	return cmd
}

func matchSubCmd(db *cache.DB) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "match",
		Aliases: []string{"m"},
		Short:   "Play Definition Matcher game",
		RunE: func(cmd *cobra.Command, args []string) error {
			useCli, _ := cmd.Flags().GetBool("cli")
			deck := loadStudyDeck(db)

			if useCli {
				cli.RunCliMatcher(deck)
				return nil
			}
			return tui.StartMatcher(db, deck)
		},
	}
	cmd.Flags().Bool("cli", false, "Run in text-interactive CLI mode")
	return cmd
}
