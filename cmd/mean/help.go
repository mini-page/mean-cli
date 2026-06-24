package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	styleHelpTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Bold(true)
	styleHelpSec   = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	styleHelpCmd   = lipgloss.NewStyle().Foreground(lipgloss.Color("#34D399")).Bold(true)
	styleHelpFlag  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FBBF24")).Bold(true)
	styleHelpDesc  = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF"))
	styleHelpNote  = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true)
)

// SetCustomHelpFunc overrides Cobra's default help output with a gorgeous Lipgloss styled layout.
func SetCustomHelpFunc(cmd *cobra.Command) {
	cmd.SetHelpFunc(func(c *cobra.Command, s []string) {
		var sb strings.Builder
		sep := styleHelpNote.Render(strings.Repeat("─", 65))

		sb.WriteString("\n" + sep + "\n")
		// Title banner
		sb.WriteString("  " + styleHelpTitle.Render("📖  MEAN CLI  -  Interactive Dictionary & Vocab Suite") + "\n")
		sb.WriteString("  " + styleHelpNote.Render("Version 1.0.0") + "\n")
		sb.WriteString(sep + "\n\n")

		// 1. Usage
		sb.WriteString(styleHelpSec.Render("  Usage:") + "\n")
		if c.Runnable() {
			sb.WriteString(fmt.Sprintf("    $ %s %s\n\n", styleHelpCmd.Render(c.UseLine()), styleHelpNote.Render("[args]")))
		}

		// 2. Subcommands
		subCmds := c.Commands()
		if len(subCmds) > 0 {
			sb.WriteString(styleHelpSec.Render("  Commands:") + "\n")
			
			examples := map[string]string{
				"cache":      "mean cache clear",
				"export":     "mean export md",
				"favorites":  "mean favorites",
				"flashcards": "mean flashcards",
				"game":       "mean game",
				"history":    "mean history",
				"quiz":       "mean quiz",
				"star":       "mean star ephemeral",
				"completion": "mean completion powershell",
			}
			
			for _, sub := range subCmds {
				// Censor internal help commands
				if sub.Name() == "help" {
					continue
				}
				
				eg := ""
				if ex, ok := examples[sub.Name()]; ok {
					eg = fmt.Sprintf("  %s", styleHelpNote.Render("(e.g., "+ex+")"))
				}
				
				sb.WriteString(fmt.Sprintf("    %-18s %-45s%s\n", 
					styleHelpCmd.Render(sub.Name()), 
					styleHelpDesc.Render(sub.Short),
					eg,
				))
			}
			sb.WriteString("\n")
		}

		// 3. Flags
		flags := c.LocalFlags().FlagUsages()
		if flags != "" {
			sb.WriteString(styleHelpSec.Render("  Flags:") + "\n")
			lines := strings.Split(strings.TrimSpace(flags), "\n")
			for _, line := range lines {
				sb.WriteString("    " + styleHelpFlag.Render(line) + "\n")
			}
			sb.WriteString("\n")
		}

		// 4. Detailed Guide Notes
		if c.Long != "" {
			sb.WriteString(styleHelpSec.Render("  Guide & Shortcuts:") + "\n")
			lines := strings.Split(c.Long, "\n")
			for _, line := range lines {
				sb.WriteString("  " + styleHelpDesc.Render(line) + "\n")
			}
			sb.WriteString("\n")
		}

		sb.WriteString(sep + "\n")
		fmt.Println(sb.String())
	})
}

func applyCustomHelp(cmd *cobra.Command) {
	SetCustomHelpFunc(cmd)
	for _, sub := range cmd.Commands() {
		applyCustomHelp(sub)
	}
}
