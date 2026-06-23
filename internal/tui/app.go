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

// ─── Model ────────────────────────────────────────────────────────────────────

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	db        *cache.DB
	input     textinput.Model
	spinner   spinner.Model
	viewport  viewport.Model

	word      *models.Word
	isFav     bool
	loading   bool
	copied    bool
	err       string
	searching bool // input focused

	width  int
	height int
}

// New creates a fresh TUI model.
func New(db *cache.DB) Model {
	ti := textinput.New()
	ti.Placeholder = "Search for a word (e.g. ephemeral) and press Enter"
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

	return Model{
		db:        db,
		input:     ti,
		spinner:   sp,
		viewport:  vp,
		searching: true,
	}
}

// ─── Init ─────────────────────────────────────────────────────────────────────

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

// ─── Update ───────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeViewport()

	case tea.MouseMsg:
		if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		} else if msg.Type == tea.MouseLeft {
			if msg.Y <= 4 {
				m.searching = true
				m.input.Focus()
			} else {
				m.searching = false
				m.input.Blur()
			}
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if !m.searching {
				return m, tea.Quit
			}
		case "esc":
			if m.searching {
				m.searching = false
				m.input.Blur()
			} else {
				return m, tea.Quit
			}
		case "/", "ctrl+k":
			m.searching = true
			m.input.Focus()
		case "tab":
			m.searching = !m.searching
			if m.searching {
				m.input.Focus()
			} else {
				m.input.Blur()
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

	if m.searching {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ─── Commands ─────────────────────────────────────────────────────────────────

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

// ─── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "initialising…"
	}

	var b strings.Builder

	// Header
	b.WriteString(m.viewHeader())
	b.WriteString("\n")

	// Search Box
	b.WriteString(m.viewSearch())
	b.WriteString("\n\n")

	// Content Area
	if m.loading {
		loaderStr := "  " + m.spinner.View() + " Looking up…\n"
		b.WriteString(loaderStr)
	} else if m.err != "" {
		errStr := styleError.Render("  ✗ " + m.err)
		b.WriteString(errStr + "\n")
	} else if m.word != nil {
		b.WriteString(m.viewWordHeader() + "\n")
		
		innerW := m.width - 8
		if innerW < 10 {
			innerW = 10
		}
		renderedVP := stylePanel.Width(innerW).Render(m.viewport.View())
		b.WriteString(renderedVP + "\n")
	} else {
		helpStr := styleMuted.Render("  Type a word in the search box above and press Enter.")
		b.WriteString(helpStr + "\n")
	}

	// Status Bar
	b.WriteString("\n")
	b.WriteString(m.viewStatusBar())

	return b.String()
}

func (m Model) viewHeader() string {
	logo := styleTitle.Render("📖  mean") + styleSubtext.Render("  —  Dictionary & Vocabulary TUI")
	return styleHeader.Render(logo)
}

func (m Model) viewSearch() string {
	var style lipgloss.Style
	if m.searching {
		style = styleSearchBox.BorderForeground(colorPrimary)
	} else {
		style = styleSearchBox.BorderForeground(colorBorder)
	}
	
	tiWidth := m.width - 12
	if tiWidth < 10 {
		tiWidth = 10
	}
	m.input.Width = tiWidth

	innerSearchW := m.width - 8
	if innerSearchW < 10 {
		innerSearchW = 10
	}
	box := style.Width(innerSearchW).Render(m.input.View())
	return box
}

func (m Model) viewWordHeader() string {
	w := m.word
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
			Render("copied to clipboard!")
		header += "  " + copiedBadge
	}
	return "  " + header + "\n"
}

func (m Model) viewStatusBar() string {
	var focusHelp string
	if m.searching {
		focusHelp = keyBind("Tab", "scroll definition")
	} else {
		focusHelp = keyBind("Tab", "edit search")
	}

	keys := []string{
		keyBind("/", "focus search"),
		focusHelp,
		keyBind("s", "star"),
		keyBind("c", "copy"),
		keyBind("p", "pronounce"),
		keyBind("↑↓/scroll", "scroll definition"),
		keyBind("q", "quit"),
	}
	bar := styleStatusBar.Width(m.width - 4).Render(strings.Join(keys, "  "))
	return "  " + bar
}

func keyBind(key, action string) string {
	return styleStatusKey.Render(key) + " " + styleMuted.Render(action)
}

// ─── Layout helpers ───────────────────────────────────────────────────────────

func (m *Model) resizeViewport() {
	innerPanelW := m.width - 10
	if innerPanelW < 10 {
		innerPanelW = 10
	}
	contentH := m.height - 11
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
