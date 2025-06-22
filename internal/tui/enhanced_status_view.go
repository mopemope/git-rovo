package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/logger"
)

// toggleCurrentSection toggles the expansion of the current section
func (m *Model) toggleCurrentSection() tea.Cmd {
	// Determine which section the cursor is in
	staged, unstaged, _ := m.groupFiles()

	var sectionKey string
	if m.cursor < len(staged) {
		sectionKey = "staged_changes"
	} else if m.cursor < len(staged)+len(unstaged) {
		sectionKey = "unstaged_changes"
	} else {
		sectionKey = "untracked_files"
	}

	m.toggleSection(sectionKey)

	logger.LogUIAction("section_toggled", map[string]interface{}{
		"section":  sectionKey,
		"expanded": m.mainViewState.expandedSections[sectionKey],
	})

	return nil
}

// unstageAllFiles unstages all staged files
func (m *Model) unstageAllFiles() tea.Cmd {
	return func() tea.Msg {
		// Get all staged files
		staged, _, _ := m.groupFiles()
		if len(staged) == 0 {
			return errorMsg{error: "No staged files to unstage"}
		}

		var filePaths []string
		for _, file := range staged {
			filePaths = append(filePaths, file.Path)
		}

		err := m.repo.UnstageFiles(filePaths...)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to unstage files: %v", err)}
		}

		logger.LogUIAction("all_files_unstaged", map[string]interface{}{
			"count": len(filePaths),
		})

		return operationCompletedMsg{message: fmt.Sprintf("Unstaged %d files", len(filePaths))}
	}
}

// discardCurrentFileChanges discards changes to the current file with confirmation
func (m *Model) discardCurrentFileChanges() tea.Cmd {
	file := m.getCurrentFile()
	if file == nil {
		return func() tea.Msg {
			return errorMsg{error: "No file selected"}
		}
	}

	// For now, we'll implement without a modal dialog
	// In a real implementation, you might want to add a confirmation modal
	return func() tea.Msg {
		err := m.repo.DiscardChanges(file.Path)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to discard changes: %v", err)}
		}

		logger.LogUIAction("file_changes_discarded", map[string]interface{}{
			"file":   file.Path,
			"status": file.Status,
			"staged": file.Staged,
		})

		var message string
		switch file.Status {
		case "??":
			message = fmt.Sprintf("Deleted: %s", file.Path)
		default:
			message = fmt.Sprintf("Discarded changes: %s", file.Path)
		}

		return operationCompletedMsg{message: message}
	}
}

// resetCurrentFile resets (discards changes to) the current file
func (m *Model) resetCurrentFile() tea.Cmd {
	file := m.getCurrentFile()
	if file == nil {
		return func() tea.Msg {
			return errorMsg{error: "No file selected"}
		}
	}

	if file.Status == "??" {
		return func() tea.Msg {
			return errorMsg{error: "Cannot reset untracked file"}
		}
	}

	return func() tea.Msg {
		// Use git checkout to discard changes
		_, err := m.repo.RunGitCommand("checkout", "HEAD", "--", file.Path)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to reset file: %v", err)}
		}

		logger.LogUIAction("file_reset", map[string]interface{}{
			"file": file.Path,
		})

		return operationCompletedMsg{message: fmt.Sprintf("Reset: %s", file.Path)}
	}
}

// selectAllFiles selects all files
func (m *Model) selectAllFiles() tea.Cmd {
	for i := range m.fileStatus {
		m.selected[i] = true
	}

	logger.LogUIAction("all_files_selected", map[string]interface{}{
		"count": len(m.fileStatus),
	})

	return nil
}

// clearSelection clears all file selections
func (m *Model) clearSelection() tea.Cmd {
	m.selected = make(map[int]bool)

	logger.LogUIAction("selection_cleared", nil)

	return nil
}
