package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/umang/mean-cli/internal/api"
	"github.com/umang/mean-cli/internal/audio"
	"github.com/umang/mean-cli/internal/cache"
	"github.com/umang/mean-cli/internal/models"
)

// ─── Messages ─────────────────────────────────────────────────────────────────

type wordResultMsg struct{ word *models.Word }
type wordErrMsg struct{ err error }
type favToggledMsg struct{ isFav bool }
type clearCopyNotificationMsg struct{}

type tabType int

const (
	tabSearch tabType = iota
	tabFavorites
	tabRecent
	tabLearn
	tabReview
	tabGames
	tabStats
)

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	db        *cache.DB
	deck      []models.Word
	input     textinput.Model
	spinner   spinner.Model
	viewport  viewport.Model

	// Menu state
	activeTab  tabType
	focusMenu  bool // true if focus is on the sidebar menu
	menuCursor int

	// Search tab state
	word      *models.Word
	isFav     bool
	loading   bool
	copied    bool
	err       string
	searching bool // text input focused

	// List cursors
	listCursor int

	// Review queue state
	dueWords        []string
	reviewIndex     int
	reviewShow      bool // show definition
	reviewedCorrect int
	reviewCompleted bool

	// Embedded Game Center
	gameCenter *GameCenterModel

	width  int
	height int
}

// New creates a fresh TUI model.
func New(db *cache.DB, deck []models.Word) Model {
	ti := textinput.New()
	ti.Placeholder = "Type word & Enter"
	ti.Focus()
	ti.CharLimit = 100
	ti.Prompt = "🔍 "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styleSpinner

	vp := viewport.New(0, 0)

	gc := NewGameCenterModel(db, deck)
	gc.Embedded = true

	return Model{
		db:         db,
		deck:       deck,
		input:      ti,
		spinner:    sp,
		viewport:   vp,
		searching:  true,
		activeTab:  tabSearch,
		focusMenu:  true,
		menuCursor: 0,
		gameCenter: gc,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle window size changes
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sz.Width
		m.height = sz.Height
		m.resizeViewport()
		// Also update gameCenter size based on inner content area
		contentWidth := sz.Width - 32
		if contentWidth < 15 {
			contentWidth = 15
		}
		panelHeight := sz.Height - 7
		if panelHeight < 5 {
			panelHeight = 5
		}
		m.gameCenter.width = contentWidth - 4
		m.gameCenter.height = panelHeight - 2
	}

	// ─── If Game TUI is active, delegate entirely ───
	if m.activeTab == tabGames && m.gameCenter.state == stateMenu {
		if k, ok := msg.(tea.KeyMsg); ok {
			if k.String() == "q" || k.String() == "esc" {
				m.focusMenu = true
				m.activeTab = tabSearch
				m.menuCursor = 0
				return m, nil
			}
		}
	}
	if m.activeTab == tabGames {
		// Delegate update to game center
		var newGC tea.Model
		var cmd tea.Cmd
		newGC, cmd = m.gameCenter.Update(msg)
		m.gameCenter = newGC.(*GameCenterModel)
		return m, cmd
	}

	// ─── Menu Navigation ───
	if m.focusMenu {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "up", "k":
				if m.menuCursor > 0 {
					m.menuCursor--
				} else {
					m.menuCursor = 6
				}
			case "down", "j":
				if m.menuCursor < 6 {
					m.menuCursor++
				} else {
					m.menuCursor = 0
				}
			case "enter", "right", "l":
				m.activeTab = tabType(m.menuCursor)
				m.focusMenu = false
				m.listCursor = 0

				// Trigger loading list elements or setup
				if m.activeTab == tabReview {
					m.startReviewSession()
				}
				if m.activeTab == tabSearch {
					m.searching = true
					m.input.Focus()
				} else {
					m.searching = false
					m.input.Blur()
				}
			}
		}
		return m, nil
	}

	// ─── Right Panel Focus ───
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "left", "h":
			m.focusMenu = true
			m.searching = false
			m.input.Blur()
			return m, nil
		}
	}

	// Tab-specific key inputs
	switch m.activeTab {
	case tabSearch:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				m.searching = !m.searching
				if m.searching {
					m.input.Focus()
				} else {
					m.input.Blur()
				}
			case "/":
				if !m.searching {
					m.searching = true
					m.input.Focus()
					return m, nil
				}
			case "enter":
				if m.searching && strings.TrimSpace(m.input.Value()) != "" {
					word := strings.TrimSpace(m.input.Value())
					m.loading = true
					m.err = ""
					m.word = nil
					m.copied = false
					m.searching = false
					m.input.Blur()
					return m, m.lookupCmd(word)
				}
			case "s":
				if !m.searching && m.word != nil {
					return m, m.toggleFavCmd(m.word.Word)
				}
			case "c":
				if !m.searching && m.word != nil {
					definitionText := buildPlainCopyText(m.word)
					if err := clipboard.WriteAll(definitionText); err == nil {
						m.copied = true
						return m, m.resetCopyNotificationCmd()
					}
				}
			case "p":
				if !m.searching && m.word != nil {
					for _, ph := range m.word.Phonetics {
						if ph.Audio != "" {
							audio.PlayFromURL(ph.Audio)
							break
						}
					}
				}
			}
		}

	case tabFavorites:
		favs, _ := m.db.GetFavorites()
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.listCursor > 0 {
					m.listCursor--
				}
			case "down", "j":
				if m.listCursor < len(favs)-1 {
					m.listCursor++
				}
			case "enter":
				if len(favs) > 0 && m.listCursor < len(favs) {
					word := favs[m.listCursor]
					m.activeTab = tabSearch
					m.menuCursor = 0
					m.loading = true
					m.err = ""
					m.word = nil
					m.copied = false
					m.searching = false
					m.input.Blur()
					return m, m.lookupCmd(word)
				}
			case "s":
				if len(favs) > 0 && m.listCursor < len(favs) {
					word := favs[m.listCursor]
					_ = m.db.RemoveFavorite(word)
					// adjust cursor
					if m.listCursor >= len(favs)-1 && m.listCursor > 0 {
						m.listCursor--
					}
				}
			}
		}

	case tabRecent:
		hist, _ := m.db.GetHistory(20)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.listCursor > 0 {
					m.listCursor--
				}
			case "down", "j":
				if m.listCursor < len(hist)-1 {
					m.listCursor++
				}
			case "enter":
				if len(hist) > 0 && m.listCursor < len(hist) {
					word := hist[m.listCursor]
					m.activeTab = tabSearch
					m.menuCursor = 0
					m.loading = true
					m.err = ""
					m.word = nil
					m.copied = false
					m.searching = false
					m.input.Blur()
					return m, m.lookupCmd(word)
				}
			}
		}

	case tabLearn:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "l", "L":
				m.enqueueDailyWords()
			}
		}

	case tabReview:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				if m.reviewCompleted {
					m.startReviewSession()
				} else if !m.reviewShow {
					m.reviewShow = true
				}
			case "y", "Y":
				if !m.reviewCompleted && m.reviewShow {
					m.handleReviewAnswer(true)
				}
			case "n", "N":
				if !m.reviewCompleted && m.reviewShow {
					m.handleReviewAnswer(false)
				}
			}
		}
	}

	// ─── Global Background Message Triggers ───
	switch msg := msg.(type) {
	case wordResultMsg:
		m.loading = false
		m.word = msg.word
		m.copied = false
		m.refreshViewport()
		isFav, _ := m.db.IsFavorite(m.word.Word)
		m.isFav = isFav

	case wordErrMsg:
		m.loading = false
		m.err = msg.err.Error()

	case favToggledMsg:
		m.isFav = msg.isFav

	case clearCopyNotificationMsg:
		m.copied = false

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.activeTab == tabSearch {
		if m.searching {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)
		} else {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) lookupCmd(word string) tea.Cmd {
	return func() tea.Msg {
		if cached, err := m.db.GetWord(word); err == nil && cached != nil {
			_ = m.db.AddHistory(word)
			return wordResultMsg{word: cached}
		}

		w, err := api.Lookup(word)
		if err != nil {
			return wordErrMsg{err: err}
		}
		_ = m.db.SaveWord(w)
		_ = m.db.AddHistory(word)
		return wordResultMsg{word: w}
	}
}

func (m Model) toggleFavCmd(word string) tea.Cmd {
	return func() tea.Msg {
		if m.isFav {
			_ = m.db.RemoveFavorite(word)
			return favToggledMsg{isFav: false}
		}
		_ = m.db.AddFavorite(word)
		return favToggledMsg{isFav: true}
	}
}

func (m Model) resetCopyNotificationCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearCopyNotificationMsg{}
	})
}

// Spaced Repetition Helpers
func (m *Model) startReviewSession() {
	_, _, _ = m.db.UpdateStreak()
	due, _ := m.db.GetDueWords()
	m.dueWords = due
	m.reviewIndex = 0
	m.reviewShow = false
	m.reviewedCorrect = 0
	m.reviewCompleted = false
}

func (m *Model) handleReviewAnswer(correct bool) {
	if m.reviewIndex >= len(m.dueWords) {
		return
	}
	word := m.dueWords[m.reviewIndex]
	lw, _ := m.db.GetLearningWord(word)

	intervals := map[int]int{
		1: 1, 2: 3, 3: 7, 4: 14, 5: 30,
	}

	if correct {
		m.reviewedCorrect++
		newBox := 1
		if lw != nil {
			newBox = lw.Box + 1
			if newBox > 5 {
				newBox = 5
			}
		}
		days := intervals[newBox]
		_ = m.db.UpdateLearningWord(word, newBox, time.Now().AddDate(0, 0, days), days)
	} else {
		_ = m.db.UpdateLearningWord(word, 1, time.Now().AddDate(0, 0, 1), 1)
	}

	m.reviewIndex++
	m.reviewShow = false
	if m.reviewIndex >= len(m.dueWords) {
		m.reviewCompleted = true
	}
}

func (m *Model) enqueueDailyWords() {
	var candidates []string
	favs, _ := m.db.GetFavorites()
	for _, f := range favs {
		lw, _ := m.db.GetLearningWord(f)
		if lw == nil {
			candidates = append(candidates, f)
		}
	}

	if len(candidates) < 3 {
		hist, _ := m.db.GetHistory(30)
		for _, h := range hist {
			lw, _ := m.db.GetLearningWord(h)
			if lw == nil {
				candidates = append(candidates, h)
			}
		}
	}

	if len(candidates) < 3 {
		for _, w := range m.deck {
			lw, _ := m.db.GetLearningWord(w.Word)
			if lw == nil {
				candidates = append(candidates, w.Word)
			}
		}
	}

	added := 0
	for _, c := range candidates {
		if added >= 3 {
			break
		}
		_ = m.db.AddLearningWord(c)
		added++
	}
}

// ─── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "initialising…"
	}

	var sb strings.Builder

	// Header
	sb.WriteString(m.viewHeader() + "\n")

	// Left sidebar
	sidebarStr := m.viewSidebar()

	// Right content
	var contentStr string
	switch m.activeTab {
	case tabSearch:
		contentStr = m.viewSearchTab()
	case tabFavorites:
		contentStr = m.viewFavoritesTab()
	case tabRecent:
		contentStr = m.viewRecentTab()
	case tabLearn:
		contentStr = m.viewLearnTab()
	case tabReview:
		contentStr = m.viewReviewTab()
	case tabGames:
		contentStr = m.gameCenter.View()
	case tabStats:
		contentStr = m.viewStatsTab()
	}

	panelHeight := m.height - 7
	if panelHeight < 5 {
		panelHeight = 5
	}

	// Styles for layout panels
	sbBorderColor := colorBorder
	if m.focusMenu {
		sbBorderColor = colorPrimary
	}

	sidebarBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(sbBorderColor).
		Width(24).
		Height(panelHeight).
		Padding(1, 1).
		Render(sidebarStr)

	contentBorderColor := colorBorder
	if !m.focusMenu {
		contentBorderColor = colorPrimary
	}

	contentWidth := m.width - 32
	if contentWidth < 15 {
		contentWidth = 15
	}

	contentBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(contentBorderColor).
		Width(contentWidth).
		Height(panelHeight).
		Padding(1, 2).
		Render(contentStr)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, sidebarBox, "  ", contentBox)
	sb.WriteString(columns + "\n")

	sb.WriteString(m.viewStatusBar())

	return sb.String()
}

func (m Model) viewHeader() string {
	logo := styleTitle.Render("📖  mean") + styleSubtext.Render("  —  Terminal Vocabulary OS & Learning System")
	return styleHeader.Render(logo)
}

func (m Model) viewSidebar() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("📁  OS MENU") + "\n\n")

	menuItems := []struct {
		tab   tabType
		label string
	}{
		{tabSearch, "🔍 Lookup"},
		{tabFavorites, "★ Starred"},
		{tabRecent, "🕒 History"},
		{tabLearn, "📚 Daily Learn"},
		{tabReview, "🔁 Review (SRM)"},
		{tabGames, "🎮 Games"},
		{tabStats, "📊 Analytics"},
	}

	for i, item := range menuItems {
		var line string
		if i == m.menuCursor {
			if m.focusMenu {
				line = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(" ▶ " + item.label)
			} else {
				line = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render("   " + item.label)
			}
		} else {
			line = styleMuted.Render("   " + item.label)
		}
		sb.WriteString(line + "\n")
	}

	return sb.String()
}

func (m Model) viewSearchTab() string {
	var sb strings.Builder
	sb.WriteString(m.viewWordHeader() + "\n")

	if m.loading {
		sb.WriteString("  " + m.spinner.View() + " Looking up…\n")
	} else if m.err != "" {
		sb.WriteString(styleError.Render("  ✗ "+m.err) + "\n")
	} else if m.word != nil {
		m.viewport.Height = m.height - 15
		if m.viewport.Height < 3 {
			m.viewport.Height = 3
		}
		sb.WriteString(m.viewport.View())
	} else {
		sb.WriteString(styleMuted.Render("  Enter a word to search below.") + "\n")
	}

	sb.WriteString("\n\n" + m.viewSearchBoxInput())
	return sb.String()
}

func (m Model) viewSearchBoxInput() string {
	var style lipgloss.Style
	if !m.focusMenu && m.searching {
		style = styleSearchBox.BorderForeground(colorPrimary)
	} else {
		style = styleSearchBox.BorderForeground(colorBorder)
	}

	tiWidth := m.width - 38
	if tiWidth < 10 {
		tiWidth = 10
	}
	m.input.Width = tiWidth
	return style.Width(tiWidth).Render(m.input.View())
}

func (m Model) viewFavoritesTab() string {
	favs, _ := m.db.GetFavorites()
	if len(favs) == 0 {
		return "\n  " + styleMuted.Render("No starred words yet.") + "\n\n  Star words during Search with [s]!"
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("★ Starred Words") + "\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf("  Total: %d", len(favs))) + "\n\n")

	for i, f := range favs {
		if i == m.listCursor && !m.focusMenu {
			sb.WriteString(fmt.Sprintf(" ▶ %s\n", lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(f)))
		} else {
			sb.WriteString(fmt.Sprintf("   %s\n", f))
		}
	}
	return sb.String()
}

func (m Model) viewRecentTab() string {
	hist, _ := m.db.GetHistory(20)
	if len(hist) == 0 {
		return "\n  " + styleMuted.Render("No search history yet.")
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("🕒 Search History") + "\n\n")

	for i, h := range hist {
		if i == m.listCursor && !m.focusMenu {
			sb.WriteString(fmt.Sprintf(" ▶ %s\n", lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(h)))
		} else {
			sb.WriteString(fmt.Sprintf("   %s\n", h))
		}
	}
	return sb.String()
}

func (m Model) viewLearnTab() string {
	list, _ := m.db.GetAllLearningWords()

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("📚 Daily Learn Queue") + "\n\n")

	if len(list) == 0 {
		sb.WriteString("  No words enqueued for spaced repetition learning.\n\n")
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("[ Press L to enqueue 3 words ]") + "\n\n")
		sb.WriteString("  (Candidate words are chosen from favorites & history)")
	} else {
		sb.WriteString("  Current pipeline queue:\n")
		for i, w := range list {
			if i >= 12 {
				sb.WriteString("  • ...\n")
				break
			}
			lw, _ := m.db.GetLearningWord(w)
			box := 1
			if lw != nil {
				box = lw.Box
			}
			sb.WriteString(fmt.Sprintf("  • %-18s (Box %d)\n", w, box))
		}
		sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("[ Press L to enqueue 3 more words ]") + "\n")
	}

	return sb.String()
}

func (m Model) viewReviewTab() string {
	if len(m.dueWords) == 0 {
		return "\n  " + styleMuted.Render("🎉 All caught up!") + "\n\n  No review cards due for review.\n  Try 'Daily Learn' to add more words, or play games!"
	}

	if m.reviewCompleted {
		return fmt.Sprintf("\n  " + styleMuted.Render("✓ Review session complete!") + "\n\n  Reviewed :  %d words\n  Correct  :  %d words\n\n  Press [Enter] to restart.", len(m.dueWords), m.reviewedCorrect)
	}

	var sb strings.Builder
	word := m.dueWords[m.reviewIndex]

	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("🔁 Leitner Spaced Repetition") + "\n")
	sb.WriteString(styleMuted.Render(fmt.Sprintf("  Card %d of %d", m.reviewIndex+1, len(m.dueWords))) + "\n\n")

	sb.WriteString("  What is the definition of:\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(strings.ToUpper(word)) + "\n\n")

	if !m.reviewShow {
		sb.WriteString("  [ Press Enter to show definition ]\n")
	} else {
		w, _ := m.db.GetWord(word)
		meaning := "No cached definition."
		if w != nil && len(w.Definitions) > 0 {
			meaning = w.Definitions[0].Meaning
		}
		sb.WriteString("  " + lipgloss.NewStyle().Bold(true).Render("Definition:") + "\n")
		sb.WriteString("  " + meaning + "\n\n")
		sb.WriteString("  Did you remember it correctly?\n")
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("[y] Yes") + "    " + lipgloss.NewStyle().Foreground(colorDanger).Bold(true).Render("[n] No") + "\n")
	}

	return sb.String()
}

func (m Model) viewStatsTab() string {
	learned, favs, mastered, currStreak, maxStreak, topCat, _ := m.db.GetLearningStats()

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("📊 Mean Vocabulary OS Analytics") + "\n\n")

	renderStatRow := func(label string, value interface{}) {
		sb.WriteString(fmt.Sprintf("  %-18s :  %v\n", label, value))
	}

	renderStatRow("Pipeline Words", learned)
	renderStatRow("Starred/Favorites", favs)
	renderStatRow("Mastered (Box 5)", mastered)
	renderStatRow("Current Streak", fmt.Sprintf("🔥 %d days", currStreak))
	renderStatRow("Max Active Streak", fmt.Sprintf("🏆 %d days", maxStreak))
	renderStatRow("Top Category Focus", topCat)

	sb.WriteString("\n  Keep checking in daily to maintain your streak!")
	return sb.String()
}

func (m Model) viewStatusBar() string {
	var keys []string

	if m.focusMenu {
		keys = []string{
			keyBind("↑↓/jk", "navigate menu"),
			keyBind("Enter/→", "select option"),
			keyBind("q/Esc", "exit"),
		}
	} else {
		switch m.activeTab {
		case tabSearch:
			var focusHelp string
			if m.searching {
				focusHelp = keyBind("Tab", "scroll definition")
			} else {
				focusHelp = keyBind("Tab", "edit search")
			}
			keys = []string{
				keyBind("/", "focus search"),
				focusHelp,
				keyBind("s", "star"),
				keyBind("c", "copy"),
				keyBind("p", "pronounce"),
				keyBind("Esc/←", "back to menu"),
			}
		case tabFavorites:
			keys = []string{
				keyBind("↑↓/jk", "select word"),
				keyBind("Enter", "view definition"),
				keyBind("s", "unstar"),
				keyBind("Esc/←", "back to menu"),
			}
		case tabRecent:
			keys = []string{
				keyBind("↑↓/jk", "select word"),
				keyBind("Enter", "view definition"),
				keyBind("Esc/←", "back to menu"),
			}
		case tabLearn:
			keys = []string{
				keyBind("L", "enqueue new words"),
				keyBind("Esc/←", "back to menu"),
			}
		case tabReview:
			keys = []string{
				keyBind("Enter", "reveal definition"),
				keyBind("y/n", "mark correct/incorrect"),
				keyBind("Esc/←", "back to menu"),
			}
		case tabGames:
			switch m.gameCenter.state {
			case stateMenu:
				keys = []string{
					keyBind("↑↓/jk", "move cursor"),
					keyBind("Enter", "select game"),
					keyBind("Esc/q", "back to menu"),
				}
			case stateHangman:
				keys = []string{
					keyBind("[a-z]", "guess letter"),
					keyBind("Esc/q", "quit game"),
				}
				if m.gameCenter.hangman != nil && m.gameCenter.hangman.session != nil && (m.gameCenter.hangman.session.IsWon() || m.gameCenter.hangman.session.IsLost()) {
					keys = append(keys, keyBind("p", "pronounce"), keyBind("Enter", "next word"))
				}
			case stateMatcher:
				keys = []string{
					keyBind("A/B/C/D", "make choice"),
					keyBind("Esc/q", "quit game"),
				}
				if m.gameCenter.matcher != nil && m.gameCenter.matcher.state == stateAnswered {
					keys = append(keys, keyBind("p", "pronounce word"), keyBind("Enter", "next definition"))
				}
			case stateQuiz:
				keys = []string{
					keyBind("Enter", "submit / next"),
					keyBind("Esc", "quit quiz"),
				}
				if m.gameCenter.quiz != nil && m.gameCenter.quiz.state == stateAnswered {
					keys = append(keys, keyBind("p", "pronounce word"), keyBind("q", "quit"))
				}
			case stateFlashcards:
				keys = []string{
					keyBind("Space/Enter", "flip"),
					keyBind("→/n", "next"),
					keyBind("←/h", "prev"),
					keyBind("s", "star"),
					keyBind("p", "pronounce"),
					keyBind("Esc/q", "exit study"),
				}
			}
		case tabStats:
			keys = []string{
				keyBind("Esc/←", "back to menu"),
			}
		}
	}

	bar := styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  "))
	return "  " + bar
}

func keyBind(key, action string) string {
	return styleStatusKey.Render(key) + " " + styleMuted.Render(action)
}

func (m Model) viewWordHeader() string {
	w := m.word
	if w == nil {
		return ""
	}
	star := ""
	if m.isFav {
		star = styleStar.Render(" ★")
	}
	header := styleWordName.Render(w.Word) + star
	if w.Pronunciation != "" {
		header += "  " + stylePronunciation.Render(w.Pronunciation)
	}
	if w.ExamLevel != "" {
		header += "  " + styleExamBadge.Render(w.ExamLevel)
	}
	if w.Source == "cache" {
		header += "  " + styleCacheBadge.Render("(cached)")
	}
	if m.copied {
		copiedBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).
			Background(colorSuccess).
			Padding(0, 1).
			Bold(true).
			Render("copied!")
		header += "  " + copiedBadge
	}
	return "  " + header + "\n"
}

func (m *Model) resizeViewport() {
	innerPanelW := m.width - 36
	if innerPanelW < 10 {
		innerPanelW = 10
	}
	contentH := m.height - 15
	if contentH < 3 {
		contentH = 3
	}
	m.viewport.Width = innerPanelW
	m.viewport.Height = contentH
	m.refreshViewport()
}

func (m *Model) refreshViewport() {
	if m.word == nil {
		return
	}
	m.viewport.SetContent(buildWordContent(m.word, m.viewport.Width))
}

func buildWordContent(w *models.Word, width int) string {
	var sb strings.Builder

	// Definitions
	if len(w.Definitions) > 0 {
		sb.WriteString(styleSectionTitle.Render(" DEFINITIONS ") + "\n")
		lastPOS := ""
		for i, d := range w.Definitions {
			if d.PartOfSpeech != lastPOS {
				lastPOS = d.PartOfSpeech
				sb.WriteString("\n  " + stylePartOfSpeech.Render(" "+strings.ToUpper(d.PartOfSpeech)+" ") + "\n")
			}
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, d.Meaning))
			if d.Example != "" {
				sb.WriteString("       " + styleExample.Render("\""+d.Example+"\"") + "\n")
			}
		}
	}

	// Synonyms
	if len(w.Synonyms) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" SYNONYMS ") + "\n  ")
		sb.WriteString(buildListString(w.Synonyms, styleSynonym, width-4) + "\n")
	}

	// Antonyms
	if len(w.Antonyms) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" ANTONYMS ") + "\n  ")
		sb.WriteString(buildListString(w.Antonyms, styleAntonym, width-4) + "\n")
	}

	// Examples
	if len(w.Examples) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" EXAMPLES ") + "\n")
		for i, ex := range w.Examples {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, styleExample.Render("\""+ex+"\"")))
		}
	}

	// Etymology
	if w.Etymology != "" {
		sb.WriteString("\n" + styleSectionTitle.Render(" ETYMOLOGY ") + "\n")
		sb.WriteString("    " + styleEtymology.Render(w.Etymology) + "\n")
	}

	return sb.String()
}

func buildListString(items []string, style lipgloss.Style, maxWidth int) string {
	if len(items) == 0 {
		return styleMuted.Render("—")
	}
	var styled []string
	for _, s := range items {
		styled = append(styled, style.Render(" "+s+" "))
	}

	spacer := "  "

	var rows []string
	var currentRow []string
	currentLen := 0
	for _, item := range styled {
		itemPlain := lipgloss.Width(item)
		if len(currentRow) > 0 && currentLen+len(spacer)+itemPlain > maxWidth {
			rows = append(rows, strings.Join(currentRow, spacer))
			currentRow = nil
			currentLen = 0
		}
		currentRow = append(currentRow, item)
		if currentLen == 0 {
			currentLen = itemPlain
		} else {
			currentLen += len(spacer) + itemPlain
		}
	}
	if len(currentRow) > 0 {
		rows = append(rows, strings.Join(currentRow, spacer))
	}
	return strings.Join(rows, "\n  ")
}

func buildPlainCopyText(w *models.Word) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s (%s)\n\n", strings.ToUpper(w.Word), w.Pronunciation))
	
	if len(w.Definitions) > 0 {
		sb.WriteString("DEFINITIONS:\n")
		for i, d := range w.Definitions {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, d.PartOfSpeech, d.Meaning))
			if d.Example != "" {
				sb.WriteString(fmt.Sprintf("     Example: \"%s\"\n", d.Example))
			}
		}
		sb.WriteString("\n")
	}
	
	if len(w.Synonyms) > 0 {
		sb.WriteString("SYNONYMS:\n  " + strings.Join(w.Synonyms, ", ") + "\n\n")
	}
	if len(w.Antonyms) > 0 {
		sb.WriteString("ANTONYMS:\n  " + strings.Join(w.Antonyms, ", ") + "\n\n")
	}
	if len(w.Examples) > 0 {
		sb.WriteString("EXAMPLES:\n")
		for i, ex := range w.Examples {
			sb.WriteString(fmt.Sprintf("  %d. \"%s\"\n", i+1, ex))
		}
		sb.WriteString("\n")
	}
	if w.Etymology != "" {
		sb.WriteString("ETYMOLOGY:\n  " + w.Etymology + "\n\n")
	}
	return sb.String()
}
