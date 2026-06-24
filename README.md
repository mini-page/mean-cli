# 📖 mean — Terminal Vocabulary OS & Learning System

A fast, beautiful, offline-first dictionary, Leitner spaced repetition review system, and vocabulary building suite for developers in the terminal. Built with Go, Bubble Tea, and SQLite.

![Mean TUI Dashboard](assets/tui_mockup.jpg)

---

## ✨ Features

- **TUI Navigation Dashboard**: Double-panel sidebar navigation linking Lookup, Starred lists, Search History, Daily Learning pipeline, Spaced Repetition reviews, Analytics, and Games.
- **Leitner Spaced Repetition (SRM)**: Active recall session queues that automatically schedule practice based on 5 progressive confidence boxes.
- **Word Intensity Ladders**: Synonyms ranked and displayed as stair-stepping ladders mapping semantic strength.
- **Developer Features**: Global JSON flag `--json` and full stdin stream pipe support (e.g. `echo recursion | mean --json`).
- **Local API Server**: Start a lightweight local definition and analytics REST API on `localhost:8080` with `mean serve`.
- **Predefined Domain Focuses**: Domain-specific vocabularies for fields like `cybersecurity`, `finance`, `medical`, `legal`, and `business`.
- **Offline Cache**: First search fetches from API and caches locally in a lightweight SQLite database for instant offline access.
- **Pronunciation Audio**: TUI voice player and background audio utility for multi-platform native playback.

---

## 🚀 Installation

### Using Go
If you have Go installed on your system:
```bash
go install github.com/umang/mean-cli/cmd/mean@latest
```

### Manual Compilation
```bash
git clone https://github.com/umang/mean-cli.git
cd mean-cli
go build -o mean.exe ./cmd/mean
```

---

## ⌨️ Usage

### Quick Lookup & Developer Features
```bash
# Look up word definition directly
mean serendipity

# Stdin Pipe support
echo recursion | mean

# JSON formatted output
mean ephemeral --json
echo paradox | mean --json
```

### Missing Core Commands
```bash
# Related: Shows semantically connected words
mean related recursion

# Compare: Shows differences, parts of speech, and definitions of two words side-by-side
mean compare affect effect

# Similar: Shows synonyms of a word
mean similar happy

# Opposite: Shows antonyms of a word
mean opposite courage

# Pronounce: Shows IPA and plays audio pronunciation
mean pronounce ephemeral

# Translate: Translates words or phrases to a target language (supports ISO codes & names)
mean translate hello hindi
mean tr "good morning" es
```

### Knowledge Commands
```bash
# Origin: Shows etymology / origin breakdown
mean origin algorithm

# Usage: Classifies casual, formal, and academic sentences
mean usage irony

# Examples: Lists sentence examples containing the word
mean examples paradox

# Phrase: Lists common compound phrases containing the word
mean phrase break

# Idiom: Lists idioms and expressions containing the word
mean idiom cat
```

### Learning OS & Gamification
```bash
# Learn: Enqueues today's candidate words to your Leitner queue
mean learn

# Review: Starts interactive spaced repetition active recall session
mean review

# Streak: Outputs current and longest daily active learning streaks
mean streak

# Stats: Displays comprehensive database and learning analytics
mean stats

# Ladder: Synonym intensity staircases
mean ladder happy

# Domain: Lists specialized domain vocab lists
mean domain cybersecurity
```

### Developer Local API Server
Start a background local REST API on port `8080`:
```bash
mean serve
```
- Endpoint 1: `GET http://localhost:8080/api/define?word=ephemeral`
- Endpoint 2: `GET http://localhost:8080/api/stats`

---

## 🖥️ Interactive Dashboard (TUI)
Simply run the command with no arguments:
```bash
mean
```

### TUI Keyboard Navigation
- **Focus Menu (Left column active)**:
  - `↑ / ↓` or `k / j`: Navigate menu options
  - `Enter` or `→` or `l`: Focus right panel content
- **Focus Panel (Right column active)**:
  - `Esc` or `←` or `h`: Defocus panel and return to sidebar menu
  - **In Search tab**:
    - `/`: Focus search input box
    - `Tab`: Toggle focus between search box and definition viewport (for scrolling)
    - `s`: Star / Unstar active word
    - `c`: Copy definition text to clipboard
    - `p`: Play pronunciation audio
  - **In Starred & History tabs**:
    - `↑ / ↓` or `k / j`: Scroll list of words
    - `Enter`: Open selected word definition
    - `s`: Unstar selected word
  - **In Daily Learn tab**:
    - `L`: Enqueue 3 new daily words to Leitner system
  - **In Review tab**:
    - `Enter`: Reveal word definition
    - `y` / `n`: Mark card as correct / incorrect
  - **In Games tab**:
    - `↑ / ↓`: Move cursor
    - `Enter`: Select game (Hangman, Matcher, Quiz, Flashcards)

---

## 🛠️ Stack & Architecture

- **Go 1.26+**
- **Bubble Tea + Lip Gloss**: Terminal UI framework and design styling
- **Cobra**: Command line interface framework
- **Pure Go SQLite (`modernc.org/sqlite`)**: Zero-dependency SQLite driver (allows seamless cross-compilation without CGO/gcc)

---

## 📄 License
MIT License. Open source and free to use.
