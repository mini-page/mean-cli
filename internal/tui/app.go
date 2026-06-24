package tui

import (
	"fmt"
	"os"
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
	"github.com/umang/mean-cli/internal/export"
	"github.com/umang/mean-cli/internal/models"
)

// ─── Messages ─────────────────────────────────────────────────────────────────

type wordResultMsg struct{ word *models.Word }
type wordErrMsg struct{ err error }
type favToggledMsg struct{ isFav bool }
type clearCopyNotificationMsg struct{}
type clearExportNotificationMsg struct{}

type relatedResultMsg struct{ words []string }
type phrasesResultMsg struct{ phrases []string }
type idiomsResultMsg struct{ idioms []string }
type usageResultMsg struct{ casual, formal, academic string }

type compareResultMsg struct {
	w1  *models.Word
	w2  *models.Word
	err error
}

type translateResultMsg struct {
	result string
	err    error
}

type tabType int

const (
	tabSearch tabType = iota
	tabCompare
	tabTranslate
	tabDomain
	tabFavorites
	tabRecent
	tabLearn
	tabReview
	tabGames
	tabStats
)

var domainVocabs = map[string][]string{
	"cybersecurity": {"threat", "exploit", "mitigation", "forensics", "payload", "vulnerability", "encryption", "firewall", "phishing", "malware"},
	"finance":       {"liquidity", "arbitrage", "equity", "leverage", "volatility", "portfolio", "dividend", "amortization", "depreciation", "commodity"},
	"medical":       {"diagnosis", "prognosis", "chronic", "acute", "biopsy", "epidemic", "immunity", "symptom", "pathogen", "outpatient"},
	"legal":         {"litigation", "jurisdiction", "plaintiff", "defendant", "affidavit", "subpoena", "contract", "testimony", "tort", "statute"},
	"business":      {"synergy", "paradigm", "scalability", "monetize", "leverage", "deliverable", "milestone", "stakeholder", "retention", "turnover"},
}

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
	exported  string
	err       string
	searching bool // text input focused

	// Extra lookup details (related, usage, phrases, idioms)
	relatedWords []string
	phrases      []string
	idioms       []string
	usageCasual  string
	usageFormal  string
	usageAcademic string

	// Compare tab state
	compareInput1 textinput.Model
	compareInput2 textinput.Model
	compareActive int // 0 = input 1, 1 = input 2
	word1         *models.Word
	word2         *models.Word
	compareErr    string
	compareVp     viewport.Model

	// Translate tab state
	transInput1  textinput.Model // text
	transInput2  textinput.Model // lang
	transActive  int // 0 = input 1, 1 = input 2
	transResult  string
	transErr     string
	transLoading bool

	// Domain tab state
	domainCursor int
	domainActive bool // true if listing domain words
	wordCursor   int  // index in domain word list

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

	// Compare inputs & viewport
	c1 := textinput.New()
	c1.Placeholder = "Word 1 (e.g. affect)"
	c1.Prompt = "⚖️  "
	c1.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	c1.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	c1.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)
	c1.Focus()

	c2 := textinput.New()
	c2.Placeholder = "Word 2 (e.g. effect)"
	c2.Prompt = "⚖️  "
	c2.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	c2.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	c2.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	cvp := viewport.New(0, 0)

	// Translate inputs
	tr1 := textinput.New()
	tr1.Placeholder = "Text to translate"
	tr1.Prompt = "🌐 "
	tr1.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	tr1.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	tr1.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)
	tr1.Focus()

	tr2 := textinput.New()
	tr2.Placeholder = "Target language (e.g. es, Hindi)"
	tr2.Prompt = "🏳️  "
	tr2.PromptStyle = lipgloss.NewStyle().Foreground(colorAccent)
	tr2.TextStyle = lipgloss.NewStyle().Foreground(colorText)
	tr2.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorMuted)

	return Model{
		db:            db,
		deck:          deck,
		input:         ti,
		spinner:       sp,
		viewport:      vp,
		searching:     true,
		activeTab:     tabSearch,
		focusMenu:     true,
		menuCursor:    0,
		gameCenter:    gc,
		compareInput1: c1,
		compareInput2: c2,
		compareVp:     cvp,
		transInput1:   tr1,
		transInput2:   tr2,
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
					m.menuCursor = 9
				}
			case "down", "j":
				if m.menuCursor < 9 {
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

				// Focus compare inputs if tabCompare selected
				if m.activeTab == tabCompare {
					m.compareActive = 0
					m.compareInput1.Focus()
					m.compareInput2.Blur()
				} else {
					m.compareInput1.Blur()
					m.compareInput2.Blur()
				}

				// Focus translate inputs if tabTranslate selected
				if m.activeTab == tabTranslate {
					m.transActive = 0
					m.transInput1.Focus()
					m.transInput2.Blur()
				} else {
					m.transInput1.Blur()
					m.transInput2.Blur()
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
			m.compareInput1.Blur()
			m.compareInput2.Blur()
			m.transInput1.Blur()
			m.transInput2.Blur()
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
			case "e":
				if !m.searching && m.word != nil {
					mdContent := export.ToMarkdown(m.word)
					txtContent := export.ToText(m.word)
					err1 := os.WriteFile(m.word.Word+".md", []byte(mdContent), 0644)
					err2 := os.WriteFile(m.word.Word+".txt", []byte(txtContent), 0644)
					if err1 == nil && err2 == nil {
						m.exported = "exported md & txt!"
					} else if err1 == nil {
						m.exported = "exported md!"
					} else if err2 == nil {
						m.exported = "exported txt!"
					} else {
						m.exported = "export failed"
					}
					return m, m.resetExportNotificationCmd()
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

	case tabCompare:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				m.compareActive = 1 - m.compareActive
				if m.compareActive == 0 {
					m.compareInput1.Focus()
					m.compareInput2.Blur()
				} else {
					m.compareInput1.Blur()
					m.compareInput2.Focus()
				}
			case "enter":
				w1Name := strings.TrimSpace(m.compareInput1.Value())
				w2Name := strings.TrimSpace(m.compareInput2.Value())
				if w1Name != "" && w2Name != "" {
					m.compareErr = ""
					m.word1 = nil
					m.word2 = nil
					return m, m.compareTuiCmd(w1Name, w2Name)
				}
			}
		}
		var cmd1, cmd2, cmdVp tea.Cmd
		m.compareInput1, cmd1 = m.compareInput1.Update(msg)
		m.compareInput2, cmd2 = m.compareInput2.Update(msg)
		m.compareVp, cmdVp = m.compareVp.Update(msg)
		cmds = append(cmds, cmd1, cmd2, cmdVp)

	case tabTranslate:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "tab":
				m.transActive = 1 - m.transActive
				if m.transActive == 0 {
					m.transInput1.Focus()
					m.transInput2.Blur()
				} else {
					m.transInput1.Blur()
					m.transInput2.Focus()
				}
			case "enter":
				text := strings.TrimSpace(m.transInput1.Value())
				lang := strings.TrimSpace(m.transInput2.Value())
				if text != "" && lang != "" {
					m.transLoading = true
					m.transResult = ""
					m.transErr = ""
					return m, m.translateTuiCmd(text, lang)
				}
			}
		}
		var cmd1, cmd2 tea.Cmd
		m.transInput1, cmd1 = m.transInput1.Update(msg)
		m.transInput2, cmd2 = m.transInput2.Update(msg)
		cmds = append(cmds, cmd1, cmd2)

	case tabDomain:
		domains := []string{"cybersecurity", "finance", "medical", "legal", "business"}
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "up", "k":
				if m.domainActive {
					if m.wordCursor > 0 {
						m.wordCursor--
					}
				} else {
					if m.domainCursor > 0 {
						m.domainCursor--
					}
				}
			case "down", "j":
				if m.domainActive {
					domainName := domains[m.domainCursor]
					list := domainVocabs[domainName]
					if m.wordCursor < len(list)-1 {
						m.wordCursor++
					}
				} else {
					if m.domainCursor < len(domains)-1 {
						m.domainCursor++
					}
				}
			case "enter":
				if m.domainActive {
					domainName := domains[m.domainCursor]
					list := domainVocabs[domainName]
					word := list[m.wordCursor]
					m.activeTab = tabSearch
					m.menuCursor = 0
					m.loading = true
					m.err = ""
					m.word = nil
					m.copied = false
					m.searching = false
					m.input.Blur()
					return m, m.lookupCmd(word)
				} else {
					m.domainActive = true
					m.wordCursor = 0
				}
			case "esc", "left", "h":
				if m.domainActive {
					m.domainActive = false
				} else {
					m.focusMenu = true
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
			case "c":
				_ = m.db.ClearHistory()
				m.listCursor = 0
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

	case tabStats:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "c":
				_ = m.db.ClearWordCache()
				m.exported = "cache cleared!"
				return m, m.resetExportNotificationCmd()
			}
		}
	}

	// ─── Global Background Message Triggers ───
	switch msg := msg.(type) {
	case wordResultMsg:
		m.loading = false
		m.word = msg.word
		m.copied = false
		isFav, _ := m.db.IsFavorite(m.word.Word)
		m.isFav = isFav

		m.relatedWords = nil
		m.phrases = nil
		m.idioms = nil
		m.usageCasual = ""
		m.usageFormal = ""
		m.usageAcademic = ""

		m.refreshViewport()

		return m, tea.Batch(
			m.fetchRelatedCmd(m.word.Word),
			m.fetchPhrasesCmd(m.word.Word),
			m.fetchIdiomsCmd(m.word.Word),
			m.fetchUsageCmd(m.word),
		)

	case compareResultMsg:
		if msg.err != nil {
			m.compareErr = msg.err.Error()
		} else {
			m.word1 = msg.w1
			m.word2 = msg.w2
			m.compareVp.SetContent(buildCompareContent(msg.w1, msg.w2, m.compareVp.Width))
		}

	case translateResultMsg:
		m.transLoading = false
		if msg.err != nil {
			m.transErr = msg.err.Error()
		} else {
			m.transResult = msg.result
		}

	case relatedResultMsg:
		m.relatedWords = msg.words
		m.refreshViewport()

	case phrasesResultMsg:
		m.phrases = msg.phrases
		m.refreshViewport()

	case idiomsResultMsg:
		m.idioms = msg.idioms
		m.refreshViewport()

	case usageResultMsg:
		m.usageCasual = msg.casual
		m.usageFormal = msg.formal
		m.usageAcademic = msg.academic
		m.refreshViewport()

	case wordErrMsg:
		m.loading = false
		m.err = msg.err.Error()

	case favToggledMsg:
		m.isFav = msg.isFav

	case clearCopyNotificationMsg:
		m.copied = false

	case clearExportNotificationMsg:
		m.exported = ""

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

func (m Model) fetchRelatedCmd(word string) tea.Cmd {
	return func() tea.Msg {
		words, err := api.FetchRelated(word)
		if err != nil {
			return relatedResultMsg{words: nil}
		}
		return relatedResultMsg{words: words}
	}
}

func (m Model) fetchPhrasesCmd(word string) tea.Cmd {
	return func() tea.Msg {
		phrases, err := api.FetchPhrases(word)
		if err != nil {
			return phrasesResultMsg{phrases: nil}
		}
		return phrasesResultMsg{phrases: phrases}
	}
}

func (m Model) fetchIdiomsCmd(word string) tea.Cmd {
	return func() tea.Msg {
		idioms, err := api.FetchIdioms(word)
		if err != nil {
			return idiomsResultMsg{idioms: nil}
		}
		return idiomsResultMsg{idioms: idioms}
	}
}

func (m Model) fetchUsageCmd(w *models.Word) tea.Cmd {
	return func() tea.Msg {
		casual, formal, academic := api.GenerateUsageGuide(w)
		return usageResultMsg{casual: casual, formal: formal, academic: academic}
	}
}

func (m Model) translateTuiCmd(text, lang string) tea.Cmd {
	return func() tea.Msg {
		res, err := api.Translate(text, lang)
		return translateResultMsg{result: res, err: err}
	}
}

func (m Model) compareTuiCmd(w1Name, w2Name string) tea.Cmd {
	return func() tea.Msg {
		w1, err := m.db.GetWord(w1Name)
		if err != nil || w1 == nil {
			w1, err = api.Lookup(w1Name)
			if err != nil {
				return compareResultMsg{err: fmt.Errorf("could not lookup %q", w1Name)}
			}
			_ = m.db.SaveWord(w1)
		}

		w2, err := m.db.GetWord(w2Name)
		if err != nil || w2 == nil {
			w2, err = api.Lookup(w2Name)
			if err != nil {
				return compareResultMsg{err: fmt.Errorf("could not lookup %q", w2Name)}
			}
			_ = m.db.SaveWord(w2)
		}

		return compareResultMsg{w1: w1, w2: w2}
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

func (m Model) resetExportNotificationCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return clearExportNotificationMsg{}
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
	case tabCompare:
		contentStr = m.viewCompareTab()
	case tabTranslate:
		contentStr = m.viewTranslateTab()
	case tabDomain:
		contentStr = m.viewDomainTab()
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
		{tabCompare, "⚖️ Compare"},
		{tabTranslate, "🌐 Translate"},
		{tabDomain, "🏛️ Domains"},
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

	tiWidth := m.width - 40
	if tiWidth < 10 {
		tiWidth = 10
	}
	m.input.Width = tiWidth - 2
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

	if m.exported != "" {
		badge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).
			Background(colorSuccess).
			Padding(0, 2).
			Bold(true).
			Render(m.exported)
		sb.WriteString("  " + badge + "\n\n")
	}

	renderStatRow := func(label string, value interface{}) {
		sb.WriteString(fmt.Sprintf("  %-18s :  %v\n", label, value))
	}

	renderStatRow("Pipeline Words", learned)
	renderStatRow("Starred/Favorites", favs)
	renderStatRow("Mastered (Box 5)", mastered)
	renderStatRow("Current Streak", fmt.Sprintf("🔥 %d days", currStreak))
	renderStatRow("Max Active Streak", fmt.Sprintf("🏆 %d days", maxStreak))
	renderStatRow("Top Category Focus", topCat)
	renderStatRow("Data Directory", cache.DataDir())

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
				keyBind("e", "export"),
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
				keyBind("c", "clear history"),
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
				keyBind("c", "clear word cache"),
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
	if m.exported != "" {
		exportedBadge := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#111827")).
			Background(colorAccent).
			Padding(0, 1).
			Bold(true).
			Render(m.exported)
		header += "  " + exportedBadge
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

	m.compareVp.Width = innerPanelW
	m.compareVp.Height = contentH
	if m.word1 != nil && m.word2 != nil {
		m.compareVp.SetContent(buildCompareContent(m.word1, m.word2, m.compareVp.Width))
	}
}

func (m *Model) refreshViewport() {
	if m.word == nil {
		return
	}
	m.viewport.SetContent(buildWordContent(m.word, *m, m.viewport.Width))
}

func buildCompareContent(w1, w2 *models.Word, width int) string {
	var sb strings.Builder

	renderWordCompact := func(w *models.Word) {
		sb.WriteString(fmt.Sprintf("  %s (%s)\n", lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render(strings.ToUpper(w.Word)), w.Pronunciation))
		if len(w.Definitions) > 0 {
			sb.WriteString(fmt.Sprintf("    * Definition: %s\n", w.Definitions[0].Meaning))
			if w.Definitions[0].Example != "" {
				sb.WriteString(fmt.Sprintf("      Example: \"%s\"\n", w.Definitions[0].Example))
			}
		}
		if len(w.Synonyms) > 0 {
			limit := len(w.Synonyms)
			if limit > 3 {
				limit = 3
			}
			sb.WriteString(fmt.Sprintf("    * Synonyms: %s\n", strings.Join(w.Synonyms[:limit], ", ")))
		}
	}

	renderWordCompact(w1)
	sb.WriteString("\n" + lipgloss.NewStyle().Foreground(colorBorder).Render("  "+strings.Repeat("╌", width-4)) + "\n\n")
	renderWordCompact(w2)

	return sb.String()
}

func buildWordContent(w *models.Word, m Model, width int) string {
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

	// Synonym Intensity Ladder
	if len(w.Synonyms) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" SYNONYM INTENSITY LADDER ") + "\n")
		sb.WriteString("    [Base] " + styleWordName.Copy().Underline(false).Foreground(colorSuccess).Render(w.Word) + "\n")
		indent := "    "
		colors := []lipgloss.Color{
			lipgloss.Color("#A78BFA"), // Light violet
			lipgloss.Color("#8B5CF6"), // Violet
			lipgloss.Color("#7C3AED"), // Darker violet
			lipgloss.Color("#EC4899"), // Pink
			lipgloss.Color("#EF4444"), // Red (extreme intensity)
		}
		var filtered []string
		seen := map[string]bool{strings.ToLower(w.Word): true}
		for _, s := range w.Synonyms {
			lowerS := strings.ToLower(s)
			if !seen[lowerS] {
				filtered = append(filtered, s)
				seen[lowerS] = true
			}
		}
		for i, s := range filtered {
			if i >= 6 {
				break
			}
			var colorIdx int
			if i >= len(colors) {
				colorIdx = len(colors) - 1
			} else {
				colorIdx = i
			}
			indent += "  "
			style := lipgloss.NewStyle().Foreground(colors[colorIdx]).Bold(true)
			sb.WriteString(indent + "🪜 " + style.Render(s) + "\n")
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

	// Related Words
	if len(m.relatedWords) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" RELATED WORDS ") + "\n  ")
		sb.WriteString(buildListString(m.relatedWords, styleSynonym.Copy().Background(colorPrimary), width-4) + "\n")
	}

	// Contextual Usage Guide
	if m.usageCasual != "" || m.usageFormal != "" || m.usageAcademic != "" {
		sb.WriteString("\n" + styleSectionTitle.Render(" CONTEXTUAL USAGE GUIDE ") + "\n")
		if m.usageCasual != "" {
			sb.WriteString("    " + lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render("[Casual]") + " " + m.usageCasual + "\n")
		}
		if m.usageFormal != "" {
			sb.WriteString("    " + lipgloss.NewStyle().Foreground(colorWarning).Bold(true).Render("[Formal]") + " " + m.usageFormal + "\n")
		}
		if m.usageAcademic != "" {
			sb.WriteString("    " + lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("[Academic]") + " " + m.usageAcademic + "\n")
		}
	}

	// Examples
	if len(w.Examples) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" EXAMPLES ") + "\n")
		for i, ex := range w.Examples {
			sb.WriteString(fmt.Sprintf("    %d. %s\n", i+1, styleExample.Render("\""+ex+"\"")))
		}
	}

	// Common Phrases
	if len(m.phrases) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" COMMON PHRASES ") + "\n  ")
		sb.WriteString(buildListString(m.phrases, styleMuted.Copy().Foreground(colorText), width-4) + "\n")
	}

	// Idioms
	if len(m.idioms) > 0 {
		sb.WriteString("\n" + styleSectionTitle.Render(" IDIOMS & EXPRESSIONS ") + "\n  ")
		sb.WriteString(buildListString(m.idioms, styleMuted.Copy().Foreground(colorWarning), width-4) + "\n")
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

func (m Model) viewCompareTab() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("⚖️ Word Comparison Tools") + "\n\n")

	boxW := m.width - 40
	if boxW < 10 {
		boxW = 10
	}
	m.compareInput1.Width = boxW - 2
	m.compareInput2.Width = boxW - 2

	c1Style := styleSearchBox.Width(boxW)
	c2Style := styleSearchBox.Width(boxW)
	if m.compareActive == 0 && !m.focusMenu {
		c1Style = c1Style.BorderForeground(colorPrimary)
	} else {
		c1Style = c1Style.BorderForeground(colorBorder)
	}
	if m.compareActive == 1 && !m.focusMenu {
		c2Style = c2Style.BorderForeground(colorPrimary)
	} else {
		c2Style = c2Style.BorderForeground(colorBorder)
	}

	sb.WriteString("  " + c1Style.Render(m.compareInput1.View()) + "   " + c2Style.Render(m.compareInput2.View()) + "\n\n")

	if m.compareErr != "" {
		sb.WriteString(styleError.Render("  ✗ "+m.compareErr) + "\n")
	} else if m.word1 != nil && m.word2 != nil {
		m.compareVp.Height = m.height - 15
		if m.compareVp.Height < 3 {
			m.compareVp.Height = 3
		}
		sb.WriteString(m.compareVp.View())
	} else {
		sb.WriteString(styleMuted.Render("  Enter two words above to compare them side-by-side.") + "\n")
	}
	return sb.String()
}

func (m Model) viewTranslateTab() string {
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("🌐 Vocabulary Translation Center") + "\n\n")

	boxW := m.width - 40
	if boxW < 10 {
		boxW = 10
	}
	m.transInput1.Width = boxW - 2
	m.transInput2.Width = boxW - 2

	t1Style := styleSearchBox.Width(boxW)
	t2Style := styleSearchBox.Width(boxW)
	if m.transActive == 0 && !m.focusMenu {
		t1Style = t1Style.BorderForeground(colorPrimary)
	} else {
		t1Style = t1Style.BorderForeground(colorBorder)
	}
	if m.transActive == 1 && !m.focusMenu {
		t2Style = t2Style.BorderForeground(colorPrimary)
	} else {
		t2Style = t2Style.BorderForeground(colorBorder)
	}

	sb.WriteString("  " + t1Style.Render(m.transInput1.View()) + "   " + t2Style.Render(m.transInput2.View()) + "\n\n")

	if m.transLoading {
		sb.WriteString("  " + m.spinner.View() + " Translating…\n")
	} else if m.transErr != "" {
		sb.WriteString(styleError.Render("  ✗ "+m.transErr) + "\n")
	} else if m.transResult != "" {
		titleStyle := lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
		sb.WriteString(titleStyle.Render("  Result:") + "\n")
		sb.WriteString("    English   :  " + lipgloss.NewStyle().Bold(true).Render(m.transInput1.Value()) + "\n")
		sb.WriteString("    Translation:  " + lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(m.transResult) + "\n")
	} else {
		sb.WriteString(styleMuted.Render("  Enter text and target language/code (e.g. es, fr, Hindi) to translate.") + "\n")
	}
	return sb.String()
}

func (m Model) viewDomainTab() string {
	domains := []string{"cybersecurity", "finance", "medical", "legal", "business"}
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Foreground(colorAccent).Bold(true).Render("🏛️ Curriculum Domain Focus Lists") + "\n\n")

	if m.domainActive {
		domainName := domains[m.domainCursor]
		list := domainVocabs[domainName]
		sb.WriteString(styleMuted.Render(fmt.Sprintf("  Browse curated terms for: %s", strings.ToUpper(domainName))) + "\n\n")
		for i, term := range list {
			if i == m.wordCursor && !m.focusMenu {
				sb.WriteString(fmt.Sprintf("   ▶ %s\n", lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(term)))
			} else {
				sb.WriteString(fmt.Sprintf("     %s\n", term))
			}
		}
		sb.WriteString("\n  " + styleMuted.Render("[ Press Enter to view definition, Esc to go back ]"))
	} else {
		sb.WriteString(styleMuted.Render("  Select a professional field to explore specialized terms:") + "\n\n")
		for i, d := range domains {
			if i == m.domainCursor && !m.focusMenu {
				sb.WriteString(fmt.Sprintf("   ▶ %s\n", lipgloss.NewStyle().Foreground(colorSuccess).Bold(true).Render(strings.ToUpper(d))))
			} else {
				sb.WriteString(fmt.Sprintf("     %s\n", strings.Title(d)))
			}
		}
	}
	return sb.String()
}
