package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/umang/mean-cli/internal/api"
	"github.com/umang/mean-cli/internal/audio"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/cli"
)

// ─── Related ─────────────────────────────────────────────────────────────────
func relatedCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "related <word>",
		Short: "Shows semantically related words",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			words, err := api.FetchRelated(word)
			if err != nil {
				return err
			}
			cli.PrintRelated(word, words, formatJson)
			return nil
		},
	}
}

// ─── Compare ─────────────────────────────────────────────────────────────────
func compareCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "compare <word1> <word2>",
		Short: "Compares definitions and details of two words side-by-side",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			w1Name := args[0]
			w2Name := args[1]
			formatJson, _ := cmd.Flags().GetBool("json")

			// Lookup word 1
			w1, err := db.GetWord(w1Name)
			if err != nil || w1 == nil {
				w1, err = api.Lookup(w1Name)
				if err != nil {
					return fmt.Errorf("could not lookup %q: %w", w1Name, err)
				}
				_ = db.SaveWord(w1)
			}

			// Lookup word 2
			w2, err := db.GetWord(w2Name)
			if err != nil || w2 == nil {
				w2, err = api.Lookup(w2Name)
				if err != nil {
					return fmt.Errorf("could not lookup %q: %w", w2Name, err)
				}
				_ = db.SaveWord(w2)
			}

			cli.PrintCompare(w1, w2, formatJson)
			return nil
		},
	}
}

// ─── Similar ─────────────────────────────────────────────────────────────────
func similarCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "similar <word>",
		Short: "Shows synonyms of a word",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			words, err := api.FetchSimilar(word)
			if err != nil {
				return err
			}
			cli.PrintSimilar(word, words, formatJson)
			return nil
		},
	}
}

// ─── Opposite ────────────────────────────────────────────────────────────────
func oppositeCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "opposite <word>",
		Short: "Shows antonyms of a word",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			words, err := api.FetchOpposite(word)
			if err != nil {
				return err
			}
			cli.PrintOpposite(word, words, formatJson)
			return nil
		},
	}
}

// ─── Pronounce ───────────────────────────────────────────────────────────────
func pronounceCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "pronounce <word>",
		Short: "Plays audio pronunciation and shows IPA phonetic data",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wordName := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			w, err := db.GetWord(wordName)
			if err != nil || w == nil {
				w, err = api.Lookup(wordName)
				if err != nil {
					return err
				}
				_ = db.SaveWord(w)
			}

			var audioURL, ipa string
			for _, p := range w.Phonetics {
				if p.Audio != "" {
					audioURL = p.Audio
				}
				if p.Text != "" {
					ipa = p.Text
				}
			}

			if formatJson {
				cli.PrintJSON(map[string]string{
					"word":      w.Word,
					"ipa":       ipa,
					"audio_url": audioURL,
				})
				return nil
			}

			fmt.Println()
			fmt.Printf("  📣 Pronunciation: %s  %s\n", strings.ToUpper(w.Word), ipa)
			if audioURL != "" {
				fmt.Printf("  🔗 Audio URL: %s\n", audioURL)
				fmt.Println("  ⚡ Playing audio...")
				audio.PlayFromURL(audioURL)
			} else {
				fmt.Println("  ✗ No pronunciation audio link found.")
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── Translate ───────────────────────────────────────────────────────────────
func translateCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:     "translate <word/phrase> <language>",
		Aliases: []string{"tr"},
		Short:   "Translates a word or phrase into the target language",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Language is the last argument
			targetLang := args[len(args)-1]
			text := strings.Join(args[:len(args)-1], " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			result, err := api.Translate(text, targetLang)
			if err != nil {
				return err
			}

			cli.PrintTranslate(text, targetLang, result, formatJson)
			return nil
		},
	}
}

// ─── Origin ──────────────────────────────────────────────────────────────────
func originCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "origin <word>",
		Short: "Shows etymology/origin of a word",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wordName := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			w, err := db.GetWord(wordName)
			if err != nil || w == nil {
				w, err = api.Lookup(wordName)
				if err != nil {
					return err
				}
				_ = db.SaveWord(w)
			}

			cli.PrintOrigin(w.Word, w.Etymology, formatJson)
			return nil
		},
	}
}

// ─── Usage ───────────────────────────────────────────────────────────────────
func usageCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "usage <word>",
		Short: "Shows casual, formal, and academic usage examples",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wordName := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			w, err := db.GetWord(wordName)
			if err != nil || w == nil {
				w, err = api.Lookup(wordName)
				if err != nil {
					return err
				}
				_ = db.SaveWord(w)
			}

			casual, formal, academic := api.GenerateUsageGuide(w)
			cli.PrintUsage(w.Word, casual, formal, academic, formatJson)
			return nil
		},
	}
}

// ─── Examples ────────────────────────────────────────────────────────────────
func examplesCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "examples <word>",
		Short: "Fetches sentence examples containing the word",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wordName := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			w, err := db.GetWord(wordName)
			if err != nil || w == nil {
				w, err = api.Lookup(wordName)
				if err != nil {
					return err
				}
				_ = db.SaveWord(w)
			}

			if formatJson {
				cli.PrintJSON(w.Examples)
				return nil
			}

			fmt.Println()
			fmt.Printf("  📝  Examples containing: %s\n", strings.ToUpper(w.Word))
			fmt.Println("  " + strings.Repeat("─", 40))
			if len(w.Examples) == 0 {
				fmt.Println("  No sentence examples found.")
			} else {
				for i, ex := range w.Examples {
					fmt.Printf("  %d. \"%s\"\n", i+1, ex)
				}
			}
			fmt.Println()
			return nil
		},
	}
}

// ─── Phrase ──────────────────────────────────────────────────────────────────
func phraseCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "phrase <word>",
		Short: "Shows common phrases containing the word",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			phrases, err := api.FetchPhrases(word)
			if err != nil {
				return err
			}
			cli.PrintPhrases(word, phrases, formatJson)
			return nil
		},
	}
}

// ─── Idiom ───────────────────────────────────────────────────────────────────
func idiomCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "idiom <word>",
		Short: "Shows common idioms containing the word",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			idioms, err := api.FetchIdioms(word)
			if err != nil {
				return err
			}
			cli.PrintIdioms(word, idioms, formatJson)
			return nil
		},
	}
}

// ─── Ladder ──────────────────────────────────────────────────────────────────
func ladderCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "ladder <word>",
		Short: "Groups synonyms by intensity strength",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			word := strings.Join(args, " ")
			formatJson, _ := cmd.Flags().GetBool("json")

			// get synonyms/similar
			syns, err := api.FetchSimilar(word)
			if err != nil {
				return err
			}

			// filter synonyms that are not the same word
			var filtered []string
			seen := map[string]bool{strings.ToLower(word): true}
			for _, s := range syns {
				lowerS := strings.ToLower(s)
				if !seen[lowerS] {
					filtered = append(filtered, s)
					seen[lowerS] = true
				}
			}

			cli.PrintLadder(word, filtered, formatJson)
			return nil
		},
	}
}

// ─── Domain ──────────────────────────────────────────────────────────────────
var domainVocabs = map[string][]string{
	"cybersecurity": {"threat", "exploit", "mitigation", "forensics", "payload", "vulnerability", "encryption", "firewall", "phishing", "malware"},
	"finance":       {"liquidity", "arbitrage", "equity", "leverage", "volatility", "portfolio", "dividend", "amortization", "depreciation", "commodity"},
	"medical":       {"diagnosis", "prognosis", "chronic", "acute", "biopsy", "epidemic", "immunity", "symptom", "pathogen", "outpatient"},
	"legal":         {"litigation", "jurisdiction", "plaintiff", "defendant", "affidavit", "subpoena", "contract", "testimony", "tort", "statute"},
	"business":      {"synergy", "paradigm", "scalability", "monetize", "leverage", "deliverable", "milestone", "stakeholder", "retention", "turnover"},
}

func domainCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "domain <name>",
		Short: "Lists domain-specific vocabulary terms (e.g. cybersecurity, finance)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domainName := strings.ToLower(args[0])
			formatJson, _ := cmd.Flags().GetBool("json")

			list, ok := domainVocabs[domainName]
			if !ok {
				var domains []string
				for d := range domainVocabs {
					domains = append(domains, d)
				}
				return fmt.Errorf("domain %q not found. Supported domains: %s", domainName, strings.Join(domains, ", "))
			}

			if formatJson {
				cli.PrintJSON(list)
				return nil
			}

			fmt.Println()
			fmt.Printf("  🛡️  Domain Vocabulary: %s\n", strings.ToUpper(domainName))
			fmt.Println("  " + strings.Repeat("─", 40))
			for _, term := range list {
				fmt.Printf("  • %s\n", term)
			}
			fmt.Println()
			return nil
		},
	}
}
