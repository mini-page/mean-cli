package cli

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/audio"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/games"
	"github.com/umang/mean-cli/internal/models"
)

// Styling palettes for CLI games
var (
	styleCorrect = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Bold(true)
	styleWrong   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F87171")).Bold(true)
	styleHighlight = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
)

// RunCliHangman handles the CLI text interactive game loop for Hangman.
func RunCliHangman(deck []models.Word) {
	if len(deck) == 0 {
		fmt.Println("\n  ✗ Error: Study deck is empty. Search some words first!")
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := deck[r.Intn(len(deck))]
	session := games.NewHangmanSession(target)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println(styleHighlight.Render("  🎮  MEAN HANGMAN  (CLI Mode)"))
	fmt.Println("  -----------------------------")

	for !session.IsWon() && !session.IsLost() {
		// Print gallows and state
		fmt.Println(games.Gallows[6-session.Lives])
		fmt.Printf("\n  Word: %s\n", session.DisplayWord())
		fmt.Printf("  Lives Left: %d  |  Guessed: %s\n", session.Lives, stringGuesses(session.Guesses))
		fmt.Print("\n  Guess a letter: ")

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if len(input) == 0 {
			continue
		}

		char := []rune(input)[0]
		correct := session.Guess(char)
		if correct {
			fmt.Println(styleCorrect.Render("  ✓ Yes! Good guess."))
		} else {
			fmt.Println(styleWrong.Render("  ✗ No, wrong letter."))
		}
		fmt.Println()
	}

	if session.IsWon() {
		fmt.Println(games.Gallows[6-session.Lives])
		fmt.Println(styleCorrect.Render(fmt.Sprintf("\n  🎉 YOU WON! The word was: %s", strings.ToUpper(target.Word))))
	} else {
		fmt.Println(games.Gallows[6])
		fmt.Println(styleWrong.Render(fmt.Sprintf("\n  💀 GAME OVER! The word was: %s", strings.ToUpper(target.Word))))
	}

	// Show word details at the end
	PrintWord(&target)
}

// RunCliMatcher handles the CLI text multiple-choice game.
func RunCliMatcher(deck []models.Word) {
	if len(deck) < 2 {
		fmt.Println("\n  ✗ Error: Matcher requires a study deck of at least 2 words.")
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	target := deck[r.Intn(len(deck))]
	session := games.NewMatchSession(target, deck)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println(styleHighlight.Render("  🎮  DEFINITION MATCHER  (CLI Mode)"))
	fmt.Println("  -----------------------------")

	// Print Definition
	def := "No definition found"
	if len(session.Word.Definitions) > 0 {
		def = session.Word.Definitions[0].Meaning
	}
	fmt.Printf("\n  Definition:\n  %s\n\n", styleHighlight.Render(def))

	// Print options
	optionKeys := []string{"A", "B", "C", "D"}
	for i, opt := range session.Options {
		fmt.Printf("   %s. %s\n", optionKeys[i], opt)
	}

	fmt.Print("\n  Choose correct option (A/B/C/D): ")

	guessIdx := -1
	for guessIdx == -1 {
		if !scanner.Scan() {
			return
		}
		input := strings.ToUpper(strings.TrimSpace(scanner.Text()))
		switch input {
		case "A":
			guessIdx = 0
		case "B":
			guessIdx = 1
		case "C":
			guessIdx = 2
		case "D":
			guessIdx = 3
		default:
			fmt.Print("  Invalid choice. Choose A, B, C, or D: ")
		}
	}

	if session.CheckGuess(guessIdx) {
		fmt.Println(styleCorrect.Render("\n  🎉 CORRECT! Nicely matched."))
	} else {
		correctOpt := optionKeys[session.CorrectIndex]
		fmt.Println(styleWrong.Render(fmt.Sprintf("\n  ✗ INCORRECT! Correct option was %s (%s).", correctOpt, target.Word)))
	}
	PrintWord(&target)
}

func stringGuesses(runs []rune) string {
	var s []string
	for _, r := range runs {
		s = append(s, string(r))
	}
	return strings.Join(s, ", ")
}

// RunCliQuiz runs the text interactive spelling quiz game.
func RunCliQuiz(deck []models.Word) {
	if len(deck) == 0 {
		fmt.Println("\n  ✗ Error: Study deck is empty. Search some words first!")
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	score := 0
	solved := 0
	streak := 0

	fmt.Println()
	fmt.Println(styleHighlight.Render("  🎮  VOCABULARY SPELLING QUIZ  (CLI Mode)"))
	fmt.Println("  -----------------------------")

	for i, w := range deck {
		solved++
		fmt.Printf("\n  Question %d of %d\n", i+1, len(deck))
		
		// Print censored definitions
		if len(w.Definitions) > 0 {
			fmt.Println("\n  Definitions:")
			for idx, d := range w.Definitions {
				cleanMeaning := censorWord(d.Meaning, w.Word, "[_____]")
				fmt.Printf("    %d. [%s] %s\n", idx+1, d.PartOfSpeech, cleanMeaning)
			}
		}
		
		// Examples
		if len(w.Examples) > 0 {
			fmt.Println("\n  Examples:")
			for idx, ex := range w.Examples {
				cleanEx := censorWord(ex, w.Word, "[_____]")
				fmt.Printf("    %d. \"%s\"\n", idx+1, cleanEx)
			}
		}

		fmt.Print("\n  Guess the word: ")
		if !scanner.Scan() {
			break
		}
		guess := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if guess == strings.ToLower(w.Word) {
			score++
			streak++
			fmt.Println(styleCorrect.Render(fmt.Sprintf("  ✓ CORRECT! (Streak: 🔥 %d)", streak)))
		} else {
			streak = 0
			fmt.Println(styleWrong.Render(fmt.Sprintf("  ✗ INCORRECT! The correct word was: %s", strings.ToUpper(w.Word))))
		}
		
		// Optional audio pronunciation play trigger at the end of guess
		for _, ph := range w.Phonetics {
			if ph.Audio != "" {
				fmt.Println(styleHighlight.Render("  [Press 'p' + Enter to play pronunciation, or any other key to continue]"))
				fmt.Print("  Option: ")
				if scanner.Scan() {
					if strings.ToLower(strings.TrimSpace(scanner.Text())) == "p" {
						audio.PlayFromURL(ph.Audio)
						// wait briefly for trigger
						time.Sleep(1 * time.Second)
					}
				}
				break
			}
		}
	}

	scorePct := 0.0
	if solved > 0 {
		scorePct = (float64(score) / float64(solved)) * 100.0
	}
	fmt.Printf("\n  🏆 Quiz finished! Score: %d/%d (%.1f%%)\n\n", score, solved, scorePct)
}

// RunCliFlashcards runs the interactive text flashcards active recall loop.
func RunCliFlashcards(db *cache.DB, deck []models.Word) {
	if len(deck) == 0 {
		fmt.Println("\n  ✗ Error: Study deck is empty. Search some words first!")
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println()
	fmt.Println(styleHighlight.Render("  🎴  FLASHCARD STUDY DECK  (CLI Mode)"))
	fmt.Println("  -----------------------------")

	for i := 0; i < len(deck); {
		w := deck[i]
		isFav, _ := db.IsFavorite(w.Word)
		star := "☆"
		if isFav {
			star = "★"
		}

		fmt.Println(strings.Repeat("─", 45))
		fmt.Printf("  Card %d of %d   |  Favorite: %s\n", i+1, len(deck), star)
		fmt.Println(strings.Repeat("─", 45))
		fmt.Printf("\n  WORD: %s\n", styleHighlight.Render(strings.ToUpper(w.Word)))
		if w.Pronunciation != "" {
			fmt.Printf("  Pronunciation: %s\n", w.Pronunciation)
		}
		if w.ExamLevel != "" {
			fmt.Printf("  Exam Level: %s\n", w.ExamLevel)
		}

		fmt.Println("\n  [Menu: Press Enter to flip card, 'n' next, 'b' back, 'q' quit]")
		fmt.Print("  Choose action: ")
		if !scanner.Scan() {
			break
		}
		action := strings.ToLower(strings.TrimSpace(scanner.Text()))

		switch action {
		case "q":
			return
		case "n":
			i = (i + 1) % len(deck)
			continue
		case "b":
			i = (i - 1 + len(deck)) % len(deck)
			continue
		case "s":
			if isFav {
				_ = db.RemoveFavorite(w.Word)
				fmt.Println("  ☆ Removed from favorites.")
			} else {
				_ = db.AddFavorite(w.Word)
				fmt.Println("  ★ Added to favorites.")
			}
			continue
		case "p":
			for _, ph := range w.Phonetics {
				if ph.Audio != "" {
					audio.PlayFromURL(ph.Audio)
					time.Sleep(500 * time.Millisecond)
					break
				}
			}
			continue
		default:
			// Flip and show full word sheet
			fmt.Println("\n  ================ REVEALED CARD DETAILS ================")
			PrintWord(&w)
			fmt.Println("  =======================================================")
			fmt.Print("  [Press Enter to continue to next card] ")
			_ = scanner.Scan()
			i = (i + 1) % len(deck)
		}
	}
}

func censorWord(text, target, replacement string) string {
	lowerText := strings.ToLower(text)
	lowerTarget := strings.ToLower(target)
	idx := strings.Index(lowerText, lowerTarget)
	if idx == -1 {
		return text
	}
	return text[:idx] + replacement + censorWord(text[idx+len(target):], target, replacement)
}
