package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mopemope/git-rovo/internal/git"
)

// MainViewState represents the state of the main view
type MainViewState struct {
	showStagedSection    bool
	showUnstagedSection  bool
	showUntrackedSection bool
	expandedSections     map[string]bool
	sortBy               SortBy
	showStats            bool
}

// SortBy represents different sorting options
type SortBy int

const (
	SortByName SortBy = iota
	SortByStatus
	SortBySize
	SortByModified
)

// NewMainViewState creates a new main view state
func NewMainViewState() *MainViewState {
	return &MainViewState{
		showStagedSection:    true,
		showUnstagedSection:  true,
		showUntrackedSection: true,
		expandedSections:     make(map[string]bool),
		sortBy:               SortByName,
		showStats:            true,
	}
}

// renderEnhancedStatusView renders an enhanced version of the status view
func (m *Model) renderEnhancedStatusView() string {
	if len(m.fileStatus) == 0 {
		return m.renderEmptyRepository()
	}

	var content strings.Builder

	// Repository summary
	content.WriteString(m.renderRepositorySummary())
	content.WriteString("\n")

	// Group and sort files
	staged, unstaged, untracked := m.groupFiles()

	// Render sections
	currentIndex := 0

	if m.mainViewState.showStagedSection && len(staged) > 0 {
		sectionContent, newIndex := m.renderFileSection("Staged Changes", staged, currentIndex, m.styles.Success)
		content.WriteString(sectionContent)
		currentIndex = newIndex
	}

	if m.mainViewState.showUnstagedSection && len(unstaged) > 0 {
		sectionContent, newIndex := m.renderFileSection("Unstaged Changes", unstaged, currentIndex, m.styles.Warning)
		content.WriteString(sectionContent)
		currentIndex = newIndex
	}

	if m.mainViewState.showUntrackedSection && len(untracked) > 0 {
		sectionContent, _ := m.renderFileSection("Untracked Files", untracked, currentIndex, m.styles.Untracked)
		content.WriteString(sectionContent)
	}

	// Show generated commit message section
	if m.generatedMessage != "" {
		content.WriteString(m.renderCommitMessageSection())
	}

	// Show quick actions
	content.WriteString(m.renderQuickActions())

	return content.String()
}

// renderEmptyRepository renders the view for an empty/clean repository
func (m *Model) renderEmptyRepository() string {
	var content strings.Builder

	content.WriteString(m.renderRepositorySummary())
	content.WriteString("\n")

	// Clean repository message
	cleanMsg := "Working directory is clean"
	content.WriteString(m.styles.Success.Render(cleanMsg))
	content.WriteString("\n\n")

	// Show recent commits if available
	if len(m.commitHistory) > 0 {
		content.WriteString(m.styles.Info.Render("Recent commits:"))
		content.WriteString("\n")

		for i, commit := range m.commitHistory {
			if i >= 3 { // Show only last 3 commits
				break
			}

			commitLine := fmt.Sprintf("  %s %s",
				m.styles.Warning.Render(commit.ShortHash),
				commit.Subject)
			content.WriteString(commitLine)
			content.WriteString("\n")
		}
		content.WriteString("\n")
	}

	// Helpful tips
	content.WriteString(m.styles.Help.Render("Tips:"))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("  • Make changes to files and they will appear here"))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("  • Press 'r' to refresh the status"))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("  • Press 'l' to view commit history"))
	content.WriteString("\n")

	return content.String()
}

// renderRepositorySummary renders a summary of the repository state
func (m *Model) renderRepositorySummary() string {
	var content strings.Builder

	// Branch and repository info
	branch, _ := m.repo.GetCurrentBranch()
	workDir := m.repo.GetWorkDir()

	// Count files by status
	staged, unstaged, untracked := m.groupFiles()

	summaryLine := fmt.Sprintf(" %s [%s] • %d staged, %d modified, %d untracked",
		workDir, branch, len(staged), len(unstaged), len(untracked))

	content.WriteString(m.styles.Info.Render(summaryLine))

	// Show loading indicator if needed
	if m.loading {
		content.WriteString(" ")
		content.WriteString(m.styles.Loading.Render("Loading... " + m.loadingMessage))
	}

	return content.String()
}

// renderFileSection renders a section of files (staged, unstaged, or untracked)
func (m *Model) renderFileSection(title string, files []git.FileStatus, startIndex int, titleStyle lipgloss.Style) (string, int) {
	var content strings.Builder
	currentIndex := startIndex

	// Section header with collapse/expand indicator
	sectionKey := strings.ToLower(strings.ReplaceAll(title, " ", "_"))
	expanded := m.mainViewState.expandedSections[sectionKey]

	var indicator string
	if expanded {
		indicator = "v"
	} else {
		indicator = ">"
	}

	// Build header without complex styling that might cause clipping
	header := fmt.Sprintf(" %s %s (%d)", indicator, title, len(files))
	content.WriteString(titleStyle.Render(header))
	content.WriteString("\n")

	// Only show files if section is expanded (default to expanded)
	if !expanded {
		expanded = true // Default to expanded
		m.mainViewState.expandedSections[sectionKey] = true
	}

	if expanded {
		for _, file := range files {
			line := m.renderEnhancedFileStatusLine(file, currentIndex == m.cursor)
			// Add the line directly without additional spacing
			content.WriteString(line)
			content.WriteString("\n")
			currentIndex++
		}
	}

	content.WriteString("\n")
	return content.String(), currentIndex
}

// renderEnhancedFileStatusLine renders an enhanced file status line with more information
func (m *Model) renderEnhancedFileStatusLine(file git.FileStatus, selected bool) string {
	var statusIcon string
	var statusText string

	// Determine status icon and text
	switch {
	case file.Staged && strings.Contains(file.Status, "A"):
		statusIcon = "+"
		statusText = "added"
	case file.Staged && strings.Contains(file.Status, "M"):
		statusIcon = "M"
		statusText = "staged"
	case file.Staged && strings.Contains(file.Status, "D"):
		statusIcon = "-"
		statusText = "deleted"
	case file.Status == "??":
		statusIcon = "?"
		statusText = "untracked"
	case strings.Contains(file.Status, "M"):
		statusIcon = "M"
		statusText = "modified"
	case strings.Contains(file.Status, "D"):
		statusIcon = "D"
		statusText = "deleted"
	case strings.Contains(file.Status, "R"):
		statusIcon = "R"
		statusText = "renamed"
	default:
		statusIcon = " "
		statusText = "unknown"
	}

	// Build the line with absolutely minimal formatting
	var prefix string
	if selected {
		prefix = "> "
	} else {
		prefix = "  "
	}

	// Create the basic line without any styling first
	basicLine := fmt.Sprintf("%s%s %s (%s)", prefix, statusIcon, file.Path, statusText)

	// Apply styling only if needed and only to the entire line
	if selected {
		return m.styles.Selected.Render(basicLine)
	}

	return basicLine
}

// renderCommitMessageSection renders the generated commit message section
func (m *Model) renderCommitMessageSection() string {
	var content strings.Builder

	content.WriteString(m.styles.Success.Render(" Generated Commit Message:"))
	content.WriteString("\n")

	// Message box with proper padding and margin
	messageBox := m.styles.Base.
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(CatppuccinMauve)).
		Padding(1).
		MarginLeft(2).
		Width(m.width - 6).
		Render(m.generatedMessage)

	content.WriteString(messageBox)
	content.WriteString("\n")

	// Confidence and actions with proper spacing
	confidenceText := fmt.Sprintf(" Confidence: %.1f%% • Press 'c' to commit or 'g' to regenerate",
		m.messageConfidence*100)
	content.WriteString(m.styles.Help.Render(confidenceText))
	content.WriteString("\n\n")

	return content.String()
}

// renderQuickActions renders quick action buttons/hints
func (m *Model) renderQuickActions() string {
	var content strings.Builder

	content.WriteString(m.styles.Info.Render(" Quick Actions:"))
	content.WriteString("\n")

	actions := []struct {
		key  string
		desc string
	}{
		{"Space", "Toggle file staging"},
		{"s", "Stage current file"},
		{"u", "Unstage current file"},
		{"a", "Stage all files"},
		{"k", "Discard file changes"},
		{"g", "Generate commit message"},
		{"c", "Commit changes"},
		{"d", "View diff"},
	}

	var actionStrings []string
	for _, action := range actions {
		actionStr := fmt.Sprintf("%s:%s",
			m.styles.Warning.Render(action.key),
			action.desc)
		actionStrings = append(actionStrings, actionStr)
	}

	// Display actions in columns with proper spacing
	actionsPerRow := 3
	for i := 0; i < len(actionStrings); i += actionsPerRow {
		end := i + actionsPerRow
		if end > len(actionStrings) {
			end = len(actionStrings)
		}

		rowActions := actionStrings[i:end]
		actionLine := "  " + strings.Join(rowActions, " | ") // Add indentation
		content.WriteString(actionLine)
		content.WriteString("\n")
	}

	return content.String()
}

// groupFiles groups files by their status
func (m *Model) groupFiles() (staged, unstaged, untracked []git.FileStatus) {
	for _, file := range m.fileStatus {
		if file.Staged {
			staged = append(staged, file)
		} else if file.Status == "??" {
			untracked = append(untracked, file)
		} else {
			unstaged = append(unstaged, file)
		}
	}
	return
}

// toggleSection toggles the expansion state of a section
func (m *Model) toggleSection(sectionKey string) {
	if m.mainViewState.expandedSections == nil {
		m.mainViewState.expandedSections = make(map[string]bool)
	}
	m.mainViewState.expandedSections[sectionKey] = !m.mainViewState.expandedSections[sectionKey]
}

// Initialize main view state in the model
func (m *Model) initMainViewState() {
	if m.mainViewState == nil {
		m.mainViewState = NewMainViewState()
	}
}

// Test comment for commit message display
// Test change for diff view
