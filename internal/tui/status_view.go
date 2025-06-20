package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/logger"
)

// getCurrentFile returns the currently selected file based on display order
func (m *Model) getCurrentFile() *git.FileStatus {
	if len(m.fileStatus) == 0 {
		return nil
	}

	// Group files to match display order
	staged, unstaged, untracked := m.groupFiles()

	currentIndex := 0

	// Check staged files first
	if m.mainViewState.showStagedSection && len(staged) > 0 {
		if m.cursor >= currentIndex && m.cursor < currentIndex+len(staged) {
			return &staged[m.cursor-currentIndex]
		}
		currentIndex += len(staged)
	}

	// Check unstaged files
	if m.mainViewState.showUnstagedSection && len(unstaged) > 0 {
		if m.cursor >= currentIndex && m.cursor < currentIndex+len(unstaged) {
			return &unstaged[m.cursor-currentIndex]
		}
		currentIndex += len(unstaged)
	}

	// Check untracked files
	if m.mainViewState.showUntrackedSection && len(untracked) > 0 {
		if m.cursor >= currentIndex && m.cursor < currentIndex+len(untracked) {
			return &untracked[m.cursor-currentIndex]
		}
	}

	return nil
}

// stageCurrentFile stages the currently selected file
func (m *Model) stageCurrentFile() tea.Cmd {
	file := m.getCurrentFile()
	if file == nil {
		return func() tea.Msg {
			return errorMsg{error: "No file selected"}
		}
	}

	if file.Staged {
		return func() tea.Msg {
			return errorMsg{error: "File is already staged"}
		}
	}

	return func() tea.Msg {
		err := m.repo.StageFiles(file.Path)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to stage file: %v", err)}
		}

		logger.LogUIAction("file_staged", map[string]interface{}{
			"file": file.Path,
		})

		return operationCompletedMsg{message: fmt.Sprintf("Staged: %s", file.Path)}
	}
}

// unstageCurrentFile unstages the currently selected file
func (m *Model) unstageCurrentFile() tea.Cmd {
	file := m.getCurrentFile()
	if file == nil {
		return func() tea.Msg {
			return errorMsg{error: "No file selected"}
		}
	}

	if !file.Staged {
		return func() tea.Msg {
			return errorMsg{error: "File is not staged"}
		}
	}

	return func() tea.Msg {
		err := m.repo.UnstageFiles(file.Path)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to unstage file: %v", err)}
		}

		logger.LogUIAction("file_unstaged", map[string]interface{}{
			"file": file.Path,
		})

		return operationCompletedMsg{message: fmt.Sprintf("Unstaged: %s", file.Path)}
	}
}

// stageAllFiles stages all modified files
func (m *Model) stageAllFiles() tea.Cmd {
	return func() tea.Msg {
		err := m.repo.StageAll()
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to stage all files: %v", err)}
		}

		logger.LogUIAction("all_files_staged", nil)

		return operationCompletedMsg{message: "Staged all files"}
	}
}

// toggleCurrentFile toggles staging of the current file
func (m *Model) toggleCurrentFile() tea.Cmd {
	file := m.getCurrentFile()
	if file == nil {
		return func() tea.Msg {
			return errorMsg{error: "No file selected"}
		}
	}

	if file.Staged {
		return m.unstageCurrentFile()
	} else {
		return m.stageCurrentFile()
	}
}

// commitStagedChanges commits the staged changes
func (m *Model) commitStagedChanges() tea.Cmd {
	return func() tea.Msg {
		// Check if there are staged changes
		hasStaged, err := m.repo.HasStagedChanges()
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to check staged changes: %v", err)}
		}

		if !hasStaged {
			return errorMsg{error: "No staged changes to commit"}
		}

		// Use generated message if available, otherwise trigger auto-generation
		commitMessage := m.generatedMessage
		if commitMessage == "" {
			return autoGenerateAndCommitMsg{}
		}

		// Perform commit
		err = m.repo.Commit(commitMessage)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to commit: %v", err)}
		}

		logger.LogUIAction("commit_created", map[string]interface{}{
			"message": commitMessage,
		})

		// Clear generated message after successful commit
		m.generatedMessage = ""
		m.messageConfidence = 0

		return operationCompletedMsg{message: "Commit created successfully"}
	}
}
