package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mopemope/git-rovo/internal/git"
)

// DiffViewState represents the state of the diff view
type DiffViewState struct {
	scrollOffset    int
	selectedFile    int
	showLineNumbers bool
	showContext     bool
	contextLines    int
	wrapLines       bool
	showStats       bool
	viewMode        DiffViewMode
	isStaged        bool   // Track whether showing staged or unstaged diff
	currentCommit   string // Track current commit hash being viewed
}

// DiffViewMode represents different diff view modes
type DiffViewMode int

const (
	DiffViewModeUnified DiffViewMode = iota
	DiffViewModeSideBySide
	DiffViewModeWordDiff
)

// LogViewState represents the state of the log view
type LogViewState struct {
	scrollOffset   int
	selectedCommit int
	showDetails    bool
	showGraph      bool
	showStats      bool
	filterAuthor   string
	maxCommits     int
}

// NewDiffViewState creates a new diff view state
func NewDiffViewState() *DiffViewState {
	return &DiffViewState{
		scrollOffset:    0,
		selectedFile:    0,
		showLineNumbers: true,
		showContext:     true,
		contextLines:    3,
		wrapLines:       false,
		showStats:       true,
		viewMode:        DiffViewModeUnified,
		isStaged:        false, // Default to unstaged
	}
}

// NewLogViewState creates a new log view state
func NewLogViewState() *LogViewState {
	return &LogViewState{
		scrollOffset:   0,
		selectedCommit: 0,
		showDetails:    true,
		showGraph:      false,
		showStats:      true,
		maxCommits:     50,
	}
}

// renderEnhancedDiffView renders an enhanced version of the diff view
func (m *Model) renderEnhancedDiffView() string {
	if len(m.currentDiff) == 0 {
		return m.renderNoDiffAvailable()
	}

	var content strings.Builder

	// Diff header with navigation info
	content.WriteString(m.renderDiffHeader())
	content.WriteString("\n")

	// File tabs if multiple files
	if len(m.currentDiff) > 1 {
		content.WriteString(m.renderFileTabs())
		content.WriteString("\n")
	}

	// Current file diff
	if m.diffViewState.selectedFile < len(m.currentDiff) {
		diff := m.currentDiff[m.diffViewState.selectedFile]
		content.WriteString(m.renderSingleFileDiff(diff))
	}

	return content.String()
}

// renderNoDiffAvailable renders the view when no diff is available
func (m *Model) renderNoDiffAvailable() string {
	var content strings.Builder

	content.WriteString(m.styles.Info.Render("No diff available"))
	content.WriteString("\n\n")

	content.WriteString(m.styles.Help.Render("Tips:"))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("  - Select a file from status view to see changes"))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("  - Press 's' to return to status view"))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("  - Press 'D' to view staged changes"))
	content.WriteString("\n")

	return content.String()
}

// renderDiffHeader renders the diff header with file information
func (m *Model) renderDiffHeader() string {
	if len(m.currentDiff) == 0 {
		return ""
	}

	diff := m.currentDiff[m.diffViewState.selectedFile]

	var headerParts []string

	// Commit information if viewing commit diff
	if m.diffViewState.currentCommit != "" {
		headerParts = append(headerParts, fmt.Sprintf("Commit: %s", m.diffViewState.currentCommit[:8]))
	}

	// File path
	headerParts = append(headerParts, fmt.Sprintf("File: %s", diff.FilePath))

	// Staged/Unstaged indicator (only for regular diffs, not commit diffs)
	if m.diffViewState.currentCommit == "" {
		if m.diffViewState.isStaged {
			headerParts = append(headerParts, m.styles.Success.Render("(staged)"))
		} else {
			headerParts = append(headerParts, m.styles.Warning.Render("(unstaged)"))
		}
	}

	// Status
	statusText := m.getStatusText(diff.Status)
	headerParts = append(headerParts, statusText)

	// Stats
	if m.diffViewState.showStats && (diff.Additions > 0 || diff.Deletions > 0) {
		statsText := fmt.Sprintf("+%d -%d", diff.Additions, diff.Deletions)
		headerParts = append(headerParts, m.styles.Info.Render(statsText))
	}

	// View mode
	viewModeText := m.getDiffViewModeText()
	headerParts = append(headerParts, m.styles.Help.Render(viewModeText))

	header := strings.Join(headerParts, " - ")
	return m.styles.Header.Width(m.width).Render(header)
}

// renderFileTabs renders tabs for multiple files
func (m *Model) renderFileTabs() string {
	var tabs []string

	for i, diff := range m.currentDiff {
		fileName := diff.FilePath
		if len(fileName) > 20 {
			fileName = "..." + fileName[len(fileName)-17:]
		}

		var tab string
		if i == m.diffViewState.selectedFile {
			tab = m.styles.Selected.Render(fmt.Sprintf(" %s ", fileName))
		} else {
			tab = m.styles.Unselected.Render(fmt.Sprintf(" %s ", fileName))
		}
		tabs = append(tabs, tab)
	}

	return strings.Join(tabs, "")
}

// renderSingleFileDiff renders the diff for a single file
func (m *Model) renderSingleFileDiff(diff git.DiffInfo) string {
	if diff.IsBinary {
		return m.renderBinaryFileDiff(diff)
	}

	var content strings.Builder

	// File header
	content.WriteString(m.renderFileHeader(diff))
	content.WriteString("\n")

	// Diff content
	lines := strings.Split(diff.Content, "\n")
	visibleLines := m.getVisibleLines(lines)

	for i, line := range visibleLines {
		renderedLine := m.renderDiffLine(line, i+m.diffViewState.scrollOffset)
		content.WriteString(renderedLine)
		content.WriteString("\n")
	}

	// Scroll indicator
	if len(lines) > m.getMaxVisibleLines() {
		content.WriteString(m.renderScrollIndicator(len(lines)))
	}

	return content.String()
}

// renderBinaryFileDiff renders information for binary files
func (m *Model) renderBinaryFileDiff(diff git.DiffInfo) string {
	var content strings.Builder

	content.WriteString(m.styles.Warning.Render("Binary file"))
	content.WriteString("\n\n")

	content.WriteString(m.styles.Base.Render("File: " + diff.FilePath))
	content.WriteString("\n")
	content.WriteString(m.styles.Base.Render("Status: " + m.getStatusText(diff.Status)))
	content.WriteString("\n\n")

	content.WriteString(m.styles.Help.Render("Binary files cannot be displayed as text diff."))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("Use external tools to compare binary files."))

	return content.String()
}

// renderFileHeader renders the header for a single file diff
func (m *Model) renderFileHeader(diff git.DiffInfo) string {
	var content strings.Builder

	// Old and new file paths
	if diff.OldPath != diff.NewPath && diff.OldPath != "" {
		content.WriteString(m.styles.DiffRemove.Render("--- " + diff.OldPath))
		content.WriteString("\n")
		content.WriteString(m.styles.DiffAdd.Render("+++ " + diff.NewPath))
	} else {
		content.WriteString(m.styles.Info.Render("--- " + diff.FilePath))
	}

	return content.String()
}

// renderDiffLine renders a single line of diff
func (m *Model) renderDiffLine(line string, lineNum int) string {
	var style lipgloss.Style
	var prefix string

	// Line number prefix if enabled
	if m.diffViewState.showLineNumbers {
		prefix = fmt.Sprintf("%4d ", lineNum+1)
	}

	// Determine line type and style
	switch {
	case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
		style = m.styles.DiffAdd
	case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
		style = m.styles.DiffRemove
	case strings.HasPrefix(line, "@@"):
		style = m.styles.Info
		// Parse hunk header for better display
		if hunkInfo := m.parseHunkHeader(line); hunkInfo != "" {
			line = hunkInfo
		}
	case strings.HasPrefix(line, "diff --git"):
		style = m.styles.Header
	case strings.HasPrefix(line, "index "):
		style = m.styles.Help
	default:
		style = m.styles.DiffContext
	}

	// Wrap lines if enabled
	if m.diffViewState.wrapLines && len(line) > m.width-10 {
		line = m.wrapLine(line, m.width-10)
	}

	return style.Render(prefix + line)
}

// renderScrollIndicator renders a scroll indicator
func (m *Model) renderScrollIndicator(totalLines int) string {
	maxVisible := m.getMaxVisibleLines()
	if totalLines <= maxVisible {
		return ""
	}

	progress := float64(m.diffViewState.scrollOffset) / float64(totalLines-maxVisible)
	percentage := int(progress * 100)

	indicator := fmt.Sprintf("── %d%% (%d/%d lines) ──",
		percentage,
		m.diffViewState.scrollOffset+maxVisible,
		totalLines)

	return m.styles.Help.Render(indicator)
}

// renderEnhancedLogView renders an enhanced version of the log view
func (m *Model) renderEnhancedLogView() string {
	if len(m.commitHistory) == 0 {
		return m.renderNoCommitsAvailable()
	}

	var content strings.Builder

	// Log header
	content.WriteString(m.renderLogHeader())
	content.WriteString("\n")

	// Commit list
	visibleCommits := m.getVisibleCommits()
	for i, commit := range visibleCommits {
		isSelected := (i + m.logViewState.scrollOffset) == m.cursor
		renderedCommit := m.renderCommitEntry(commit, isSelected, i+m.logViewState.scrollOffset)
		content.WriteString(renderedCommit)
		content.WriteString("\n")
	}

	// Scroll indicator for log
	if len(m.commitHistory) > m.getMaxVisibleCommits() {
		content.WriteString(m.renderLogScrollIndicator())
	}

	return content.String()
}

// renderNoCommitsAvailable renders the view when no commits are available
func (m *Model) renderNoCommitsAvailable() string {
	var content strings.Builder

	content.WriteString(m.styles.Info.Render("No commit history available"))
	content.WriteString("\n\n")

	content.WriteString(m.styles.Help.Render("This might be a new repository with no commits yet."))
	content.WriteString("\n")
	content.WriteString(m.styles.Help.Render("Make your first commit to see history here."))

	return content.String()
}

// renderLogHeader renders the log header
func (m *Model) renderLogHeader() string {
	var headerParts []string

	headerParts = append(headerParts, "Commit History")

	if m.logViewState.showStats {
		headerParts = append(headerParts, fmt.Sprintf("(%d commits)", len(m.commitHistory)))
	}

	if m.logViewState.filterAuthor != "" {
		headerParts = append(headerParts, fmt.Sprintf("Author: %s", m.logViewState.filterAuthor))
	}

	header := strings.Join(headerParts, " - ")
	return m.styles.Header.Width(m.width).Render(header)
}

// renderCommitEntry renders a single commit entry
func (m *Model) renderCommitEntry(commit git.CommitInfo, selected bool, index int) string {
	var content strings.Builder

	// Commit hash and date
	hashLine := fmt.Sprintf("%s %s",
		commit.ShortHash,
		commit.Date.Format("2006-01-02 15:04"))

	if selected {
		hashLine = m.styles.Selected.Render(hashLine)
	} else {
		hashLine = m.styles.Warning.Render(hashLine)
	}
	content.WriteString(hashLine)
	content.WriteString("\n")

	// Author
	authorLine := fmt.Sprintf("Author: %s", commit.Author)
	if selected {
		authorLine = m.styles.Selected.Render(authorLine)
	} else {
		authorLine = m.styles.Info.Render(authorLine)
	}
	content.WriteString(authorLine)
	content.WriteString("\n")

	// Subject
	subjectLine := commit.Subject
	if len(subjectLine) > m.width-4 {
		subjectLine = subjectLine[:m.width-7] + "..."
	}

	if selected {
		subjectLine = m.styles.Selected.Render("Message: " + subjectLine)
	} else {
		subjectLine = m.styles.Base.Render("Message: " + subjectLine)
	}
	content.WriteString(subjectLine)
	content.WriteString("\n")

	// Body if selected and available
	if selected && m.logViewState.showDetails && commit.Body != "" {
		bodyLines := strings.Split(commit.Body, "\n")
		for _, line := range bodyLines {
			if strings.TrimSpace(line) != "" {
				bodyLine := m.styles.Help.Render("   " + line)
				content.WriteString(bodyLine)
				content.WriteString("\n")
			}
		}
	}

	return content.String()
}

// Helper methods

func (m *Model) getStatusText(status string) string {
	switch status {
	case "A":
		return m.styles.Staged.Render("Added")
	case "M":
		return m.styles.Modified.Render("Modified")
	case "D":
		return m.styles.Deleted.Render("Deleted")
	case "R":
		return m.styles.Warning.Render("Renamed")
	case "C":
		return m.styles.Info.Render("Copied")
	default:
		return m.styles.Base.Render("Changed")
	}
}

func (m *Model) getDiffViewModeText() string {
	switch m.diffViewState.viewMode {
	case DiffViewModeUnified:
		return "Unified"
	case DiffViewModeSideBySide:
		return "Side-by-side"
	case DiffViewModeWordDiff:
		return "Word diff"
	default:
		return "Unknown"
	}
}

func (m *Model) getVisibleLines(lines []string) []string {
	maxVisible := m.getMaxVisibleLines()
	start := m.diffViewState.scrollOffset
	end := start + maxVisible

	if start >= len(lines) {
		return []string{}
	}
	if end > len(lines) {
		end = len(lines)
	}

	return lines[start:end]
}

func (m *Model) getMaxVisibleLines() int {
	// Reserve space for header, footer, and padding
	return m.height - 8
}

func (m *Model) getVisibleCommits() []git.CommitInfo {
	maxVisible := m.getMaxVisibleCommits()
	start := m.logViewState.scrollOffset
	end := start + maxVisible

	if start >= len(m.commitHistory) {
		return []git.CommitInfo{}
	}
	if end > len(m.commitHistory) {
		end = len(m.commitHistory)
	}

	return m.commitHistory[start:end]
}

func (m *Model) getMaxVisibleCommits() int {
	// Each commit takes about 4 lines, reserve space for header/footer
	return (m.height - 6) / 4
}

func (m *Model) parseHunkHeader(line string) string {
	// Parse @@ -old_start,old_count +new_start,new_count @@ context
	if !strings.HasPrefix(line, "@@") {
		return line
	}

	parts := strings.Split(line, "@@")
	if len(parts) < 3 {
		return line
	}

	ranges := strings.TrimSpace(parts[1])
	context := strings.TrimSpace(parts[2])

	result := fmt.Sprintf("@@ %s @@", ranges)
	if context != "" {
		result += " " + context
	}

	return result
}

func (m *Model) wrapLine(line string, maxWidth int) string {
	if len(line) <= maxWidth {
		return line
	}

	// Simple word wrapping
	words := strings.Fields(line)
	var wrapped strings.Builder
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len()+len(word)+1 > maxWidth {
			if wrapped.Len() > 0 {
				wrapped.WriteString("\n")
			}
			wrapped.WriteString(currentLine.String())
			currentLine.Reset()
		}

		if currentLine.Len() > 0 {
			currentLine.WriteString(" ")
		}
		currentLine.WriteString(word)
	}

	if currentLine.Len() > 0 {
		if wrapped.Len() > 0 {
			wrapped.WriteString("\n")
		}
		wrapped.WriteString(currentLine.String())
	}

	return wrapped.String()
}

func (m *Model) renderLogScrollIndicator() string {
	maxVisible := m.getMaxVisibleCommits()
	if len(m.commitHistory) <= maxVisible {
		return ""
	}

	progress := float64(m.logViewState.scrollOffset) / float64(len(m.commitHistory)-maxVisible)
	percentage := int(progress * 100)

	indicator := fmt.Sprintf("── %d%% (%d/%d commits) ──",
		percentage,
		m.logViewState.scrollOffset+maxVisible,
		len(m.commitHistory))

	return m.styles.Help.Render(indicator)
}

// Initialize view states in the model
func (m *Model) initDetailedViewStates() {
	if m.diffViewState == nil {
		m.diffViewState = NewDiffViewState()
	}
	if m.logViewState == nil {
		m.logViewState = NewLogViewState()
	}
}
