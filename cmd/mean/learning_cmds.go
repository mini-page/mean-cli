package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/umang/mean-cli/internal/api"
	"github.com/umang/mean-cli/internal/cache"
)

// ─── Learn ───────────────────────────────────────────────────────────────────
func learnCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "learn",
		Short: "Display and enqueue today's vocabulary learning list",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Find candidates: favorites or history not in learning queue
			_, _, _ = db.UpdateStreak()
			
			var candidates []string
			
			// Try favorites first
			favs, _ := db.GetFavorites()
			for _, f := range favs {
				lw, _ := db.GetLearningWord(f)
				if lw == nil {
					candidates = append(candidates, f)
				}
			}

			// Try history if candidates are fewer than 3
			if len(candidates) < 3 {
				hist, _ := db.GetHistory(40)
				for _, h := range hist {
					lw, _ := db.GetLearningWord(h)
					if lw == nil {
						// Ensure it's not already in candidates
						already := false
						for _, c := range candidates {
							if strings.ToLower(c) == strings.ToLower(h) {
								already = true
								break
							}
						}
						if !already {
							candidates = append(candidates, h)
						}
					}
				}
			}

			// Try curated deck as fallback
			if len(candidates) < 3 {
				for _, cw := range curatedDeck {
					lw, _ := db.GetLearningWord(cw.Word)
					if lw == nil {
						already := false
						for _, c := range candidates {
							if strings.ToLower(c) == strings.ToLower(cw.Word) {
								already = true
								break
							}
						}
						if !already {
							candidates = append(candidates, cw.Word)
						}
					}
				}
			}

			// Take top 3
			if len(candidates) > 3 {
				candidates = candidates[:3]
			}

			if len(candidates) == 0 {
				fmt.Println()
				fmt.Println("  ✓ All available words are already in your learning queue!")
				fmt.Println("  Run 'mean review' to practice them.")
				fmt.Println()
				return nil
			}

			fmt.Println()
			fmt.Println("  ✨ Today's Words to Learn:")
			fmt.Println("  " + strings.Repeat("─", 45))

			for i, word := range candidates {
				w, err := db.GetWord(word)
				if err != nil || w == nil {
					w, err = api.Lookup(word)
					if err != nil {
						continue
					}
					_ = db.SaveWord(w)
				}

				_ = db.AddLearningWord(w.Word)

				meaning := "No definition found"
				if len(w.Definitions) > 0 {
					meaning = w.Definitions[0].Meaning
				}
				fmt.Printf("  %d. %s: %s\n", i+1, strings.ToUpper(w.Word), meaning)
			}

			fmt.Println("  " + strings.Repeat("─", 45))
			fmt.Println("  ✓ Enqueued 3 words into your Leitner system queue!")
			fmt.Println("  Use 'mean review' to practice spaced repetition review.")
			fmt.Println()

			return nil
		},
	}
}

// ─── Review ──────────────────────────────────────────────────────────────────
func reviewCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "review",
		Short: "Starts an interactive Spaced Repetition Method (Leitner) review session",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _ = db.UpdateStreak()
			due, err := db.GetDueWords()
			if err != nil {
				return err
			}

			if len(due) == 0 {
				fmt.Println()
				fmt.Println("  🎉 You are all caught up! No words due for review right now.")
				fmt.Println("  Run 'mean learn' to add new words or play a game.")
				fmt.Println()
				return nil
			}

			fmt.Println()
			fmt.Printf("  🔁 Starting Spaced Repetition Review (%d words due)\n", len(due))
			fmt.Println("  " + strings.Repeat("─", 50))

			reader := bufio.NewReader(os.Stdin)
			correctCount := 0

			intervals := map[int]int{
				1: 1,  // Box 1 -> 1 day
				2: 3,  // Box 2 -> 3 days
				3: 7,  // Box 3 -> 7 days
				4: 14, // Box 4 -> 14 days
				5: 30, // Box 5 -> 30 days
			}

			for i, wordName := range due {
				lw, err := db.GetLearningWord(wordName)
				if err != nil || lw == nil {
					continue
				}

				w, err := db.GetWord(wordName)
				if err != nil || w == nil {
					w, err = api.Lookup(wordName)
					if err != nil {
						continue
					}
					_ = db.SaveWord(w)
				}

				fmt.Printf("\n  [%d/%d] Active Recall: What is the meaning of %q?\n", i+1, len(due), strings.ToUpper(w.Word))
				fmt.Print("  👉 Press Enter to reveal definition...")
				_, _ = reader.ReadString('\n')

				meaning := "No definition found"
				if len(w.Definitions) > 0 {
					meaning = w.Definitions[0].Meaning
				}
				fmt.Printf("  💡 Meaning: %s\n", meaning)
				if len(w.Definitions) > 0 && w.Definitions[0].Example != "" {
					fmt.Printf("     Example: \"%s\"\n", w.Definitions[0].Example)
				}

				for {
					fmt.Print("  Did you remember it correctly? [y/n]: ")
					ans, _ := reader.ReadString('\n')
					ans = strings.ToLower(strings.TrimSpace(ans))
					if ans == "y" || ans == "yes" {
						correctCount++
						newBox := lw.Box + 1
						if newBox > 5 {
							newBox = 5
						}
						days := intervals[newBox]
						nextReview := time.Now().AddDate(0, 0, days)
						_ = db.UpdateLearningWord(w.Word, newBox, nextReview, days)
						fmt.Printf("  ✓ Correct! Word promoted to Box %d (next review in %d days).\n", newBox, days)
						break
					} else if ans == "n" || ans == "no" {
						_ = db.UpdateLearningWord(w.Word, 1, time.Now().AddDate(0, 0, 1), 1)
						fmt.Println("  ✗ Incorrect. Word demoted back to Box 1 (next review in 1 day).")
						break
					}
				}
			}

			fmt.Println("\n  " + strings.Repeat("─", 50))
			fmt.Printf("  ✓ Spaced repetition review complete! (%d/%d correct)\n", correctCount, len(due))
			fmt.Println("  Keep learning daily to grow your streak!")
			fmt.Println()

			return nil
		},
	}
}

// ─── Streak ──────────────────────────────────────────────────────────────────
func streakCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "streak",
		Short: "Display your vocabulary building streak statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			curr, max, err := db.UpdateStreak()
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("  🔥 Vocabulary Building Streaks")
			fmt.Println("  " + strings.Repeat("─", 35))
			fmt.Printf("  Current Streak:  %d days\n", curr)
			fmt.Printf("  Longest Streak:  %d days\n", max)
			fmt.Println("  " + strings.Repeat("─", 35))
			fmt.Println()
			return nil
		},
	}
}

// ─── Stats ───────────────────────────────────────────────────────────────────
func statsCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Display learning and database statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			learned, favs, mastered, currStreak, maxStreak, topCat, err := db.GetLearningStats()
			if err != nil {
				return err
			}

			fmt.Println()
			fmt.Println("  📊 Mean Vocabulary OS Analytics")
			fmt.Println("  " + strings.Repeat("─", 40))
			fmt.Printf("  Words in Pipeline :  %d\n", learned)
			fmt.Printf("  Starred/Favorites :  %d\n", favs)
			fmt.Printf("  Mastered (Box 5)  :  %d\n", mastered)
			fmt.Printf("  Current Streak    :  %d days\n", currStreak)
			fmt.Printf("  Max Active Streak :  %d days\n", maxStreak)
			fmt.Printf("  Top Domain Focus  :  %s\n", topCat)
			fmt.Println("  " + strings.Repeat("─", 40))
			fmt.Println()
			return nil
		},
	}
}

// ─── Serve (HTTP Local API Mode) ──────────────────────────────────────────────
func serveCmd(db *cache.DB) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Starts local definition JSON REST API on localhost:8080",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println()
			fmt.Println("  📡 Spawning Mean Local API Server...")
			fmt.Println("  🚀 Listening at http://localhost:8080")
			fmt.Println()
			fmt.Println("  Endpoints:")
			fmt.Println("    GET /api/define?word=<word>  Returns full definition JSON")
			fmt.Println("    GET /api/stats               Returns statistics JSON")
			fmt.Println()
			fmt.Println("  Press Ctrl+C to terminate.")

			http.HandleFunc("/api/define", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")

				wordName := r.URL.Query().Get("word")
				if wordName == "" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"error":"missing 'word' query parameter"}`))
					return
				}

				wordData, err := db.GetWord(wordName)
				if err != nil || wordData == nil {
					wordData, err = api.Lookup(wordName)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
						return
					}
					_ = db.SaveWord(wordData)
				}

				_ = json.NewEncoder(w).Encode(wordData)
			})

			http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Access-Control-Allow-Origin", "*")

				learned, favs, mastered, currStreak, maxStreak, topCat, err := db.GetLearningStats()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(fmt.Sprintf(`{"error":%q}`, err.Error())))
					return
				}

				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"words_learned":  learned,
					"favorites":      favs,
					"mastered":       mastered,
					"current_streak": currStreak,
					"max_streak":     maxStreak,
					"top_category":   topCat,
				})
			})

			return http.ListenAndServe(":8080", nil)
		},
	}
}
