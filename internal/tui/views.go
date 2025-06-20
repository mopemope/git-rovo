package tui

import (
	"fmt"
	"strings"
)

// renderHelpView renders the help view
func (m *Model) renderHelpView() string {
	var content strings.Builder

	content.WriteString(m.styles.Header.Render("git-rovo Help"))
	content.WriteString("\n\n")

	// Get help text from key binding manager
	helpText := m.keyBindingManager.GetHelpText(m.currentView)
	content.WriteString(helpText)

	// About section
	content.WriteString(m.styles.Success.Render("About:"))
	content.WriteString("\n")
	content.WriteString(m.styles.Base.Render("git-rovo is a TUI-based Git commit assistant that uses LLM to generate"))
	content.WriteString("\n")
	content.WriteString(m.styles.Base.Render("Conventional Commits compliant commit messages."))
	content.WriteString("\n\n")

	// Configuration info
	content.WriteString(m.styles.Success.Render("Configuration:"))
	content.WriteString("\n")
	content.WriteString(m.styles.Base.Render(fmt.Sprintf("LLM Provider: %s", m.config.LLM.Provider)))
	content.WriteString("\n")
	content.WriteString(m.styles.Base.Render(fmt.Sprintf("Model: %s", m.config.LLM.OpenAI.Model)))
	content.WriteString("\n")
	content.WriteString(m.styles.Base.Render(fmt.Sprintf("Language: %s", m.config.LLM.Language)))
	content.WriteString("\n\n")

	content.WriteString(m.styles.Info.Render("Press any key to return to the previous view."))

	return content.String()
}
