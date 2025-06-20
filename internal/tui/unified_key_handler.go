package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/logger"
)

// handleUnifiedKeyPress handles key presses using the key binding manager
func (m *Model) handleUnifiedKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Get action from key binding manager
	action, handled := m.keyBindingManager.HandleKeyPress(key, m.currentView)
	if !handled {
		// If not handled by key binding manager, return as-is
		return m, nil
	}

	// Log the action
	logger.LogUIAction("key_action", map[string]interface{}{
		"key":    key,
		"action": action,
		"view":   viewModeToString(m.currentView),
	})

	// Execute the action
	return m.executeAction(action)
}

// executeAction executes a specific action
func (m *Model) executeAction(action string) (tea.Model, tea.Cmd) {
	switch action {
	// Global actions
	case "quit":
		logger.LogUIAction("app_quit", nil)
		return m, tea.Quit
	case "help":
		return m.switchView(ViewModeHelp), nil
	case "refresh":
		return m, tea.Batch(
			m.refreshStatus(),
			m.refreshCommitHistory(),
		)
	case "status":
		return m.switchView(ViewModeStatus), nil
	case "diff":
		file := m.getCurrentFile()
		if file != nil {
			// Show staged diff if file is staged, otherwise show unstaged diff
			return m.switchView(ViewModeDiff), m.refreshDiff(file.Staged, file.Path)
		}
		// Fallback to unstaged diff if no file selected
		return m.switchView(ViewModeDiff), m.refreshDiff(false)
	case "log":
		return m.switchView(ViewModeLog), nil

	// Navigation actions
	case "nav_up":
		return m.handleNavUp()
	case "nav_down":
		return m.handleNavDown()
	case "nav_home":
		return m.handleNavHome()
	case "nav_end":
		return m.handleNavEnd()
	case "nav_page_up":
		return m.handleNavPageUp()
	case "nav_page_down":
		return m.handleNavPageDown()

	// Status view actions
	case "toggle_file":
		return m, m.toggleCurrentFile()
	case "stage_file":
		return m, m.stageCurrentFile()
	case "unstage_file":
		return m, m.unstageCurrentFile()
	case "stage_all":
		return m, m.stageAllFiles()
	case "unstage_all":
		return m, m.unstageAllFiles()
	case "commit":
		return m, m.commitStagedChanges()
	case "quick_commit":
		return m.handleQuickCommit()
	case "generate_message":
		m.loading = true
		m.loadingMessage = "Generating commit message..."
		return m, m.generateCommitMessage()
	case "regenerate_message":
		m.generatedMessage = ""
		m.messageConfidence = 0
		m.loading = true
		m.loadingMessage = "Regenerating commit message..."
		return m, m.generateCommitMessage()
	case "reset_file":
		return m, m.resetCurrentFile()
	case "toggle_section":
		return m, m.toggleCurrentSection()
	case "select_all":
		return m, m.selectAllFiles()
	case "clear_selection":
		return m, m.clearSelection()

	// Diff view actions
	case "diff_prev_file":
		return m.handleDiffPrevFile()
	case "diff_next_file":
		return m.handleDiffNextFile()
	case "toggle_line_numbers":
		m.diffViewState.showLineNumbers = !m.diffViewState.showLineNumbers
		return m, nil
	case "toggle_line_wrap":
		m.diffViewState.wrapLines = !m.diffViewState.wrapLines
		return m, nil
	case "cycle_diff_mode":
		m.cycleDiffViewMode()
		return m, nil
	case "toggle_stats":
		m.diffViewState.showStats = !m.diffViewState.showStats
		return m, nil
	case "toggle_staged_unstaged":
		// Toggle between staged and unstaged diff for current file
		if len(m.currentDiff) > 0 {
			currentFile := m.currentDiff[m.diffViewState.selectedFile].FilePath
			newStaged := !m.diffViewState.isStaged
			return m, m.refreshDiff(newStaged, currentFile)
		}
		return m, nil
	case "show_staged_diff":
		file := m.getCurrentFile()
		if file != nil {
			return m.switchView(ViewModeDiff), m.refreshDiff(true, file.Path)
		}
		// Show all staged changes if no specific file selected
		return m.switchView(ViewModeDiff), m.refreshDiff(true)

	// Log view actions
	case "show_commit_details":
		return m.handleShowCommitDetails()
	case "copy_commit_hash":
		return m.handleCopyCommitHash()
	case "toggle_graph":
		m.logViewState.showGraph = !m.logViewState.showGraph
		return m, nil

	// Help view actions
	case "return_to_status":
		return m.switchView(ViewModeStatus), nil

	default:
		// Unknown action
		m.errorMessage = fmt.Sprintf("Unknown action: %s", action)
		return m, nil
	}
}

// Navigation handlers
func (m *Model) handleNavUp() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewModeStatus:
		if m.cursor > 0 {
			m.cursor--
		}
	case ViewModeDiff:
		if m.diffViewState.scrollOffset > 0 {
			m.diffViewState.scrollOffset--
		}
	case ViewModeLog:
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.logViewState.scrollOffset {
				m.logViewState.scrollOffset = m.cursor
			}
		}
	}
	return m, nil
}

func (m *Model) handleNavDown() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewModeStatus:
		if m.cursor < len(m.fileStatus)-1 {
			m.cursor++
		}
	case ViewModeDiff:
		maxScroll := m.getMaxDiffScroll()
		if m.diffViewState.scrollOffset < maxScroll {
			m.diffViewState.scrollOffset++
		}
	case ViewModeLog:
		if m.cursor < len(m.commitHistory)-1 {
			m.cursor++
			maxVisible := m.getMaxVisibleCommits()
			if m.cursor >= m.logViewState.scrollOffset+maxVisible {
				m.logViewState.scrollOffset = m.cursor - maxVisible + 1
			}
		}
	}
	return m, nil
}

func (m *Model) handleNavHome() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewModeStatus:
		m.cursor = 0
	case ViewModeDiff:
		m.diffViewState.scrollOffset = 0
	case ViewModeLog:
		m.cursor = 0
		m.logViewState.scrollOffset = 0
	}
	return m, nil
}

func (m *Model) handleNavEnd() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewModeStatus:
		if len(m.fileStatus) > 0 {
			m.cursor = len(m.fileStatus) - 1
		}
	case ViewModeDiff:
		m.diffViewState.scrollOffset = m.getMaxDiffScroll()
	case ViewModeLog:
		if len(m.commitHistory) > 0 {
			m.cursor = len(m.commitHistory) - 1
			maxVisible := m.getMaxVisibleCommits()
			m.logViewState.scrollOffset = len(m.commitHistory) - maxVisible
			if m.logViewState.scrollOffset < 0 {
				m.logViewState.scrollOffset = 0
			}
		}
	}
	return m, nil
}

func (m *Model) handleNavPageUp() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewModeStatus:
		pageSize := 10
		m.cursor -= pageSize
		if m.cursor < 0 {
			m.cursor = 0
		}
	case ViewModeDiff:
		pageSize := m.getMaxVisibleLines() / 2
		m.diffViewState.scrollOffset -= pageSize
		if m.diffViewState.scrollOffset < 0 {
			m.diffViewState.scrollOffset = 0
		}
	case ViewModeLog:
		pageSize := m.getMaxVisibleCommits()
		m.cursor -= pageSize
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.logViewState.scrollOffset = m.cursor
	}
	return m, nil
}

func (m *Model) handleNavPageDown() (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewModeStatus:
		pageSize := 10
		m.cursor += pageSize
		if m.cursor >= len(m.fileStatus) {
			m.cursor = len(m.fileStatus) - 1
		}
	case ViewModeDiff:
		pageSize := m.getMaxVisibleLines() / 2
		maxScroll := m.getMaxDiffScroll()
		m.diffViewState.scrollOffset += pageSize
		if m.diffViewState.scrollOffset > maxScroll {
			m.diffViewState.scrollOffset = maxScroll
		}
	case ViewModeLog:
		pageSize := m.getMaxVisibleCommits()
		m.cursor += pageSize
		if m.cursor >= len(m.commitHistory) {
			m.cursor = len(m.commitHistory) - 1
		}
		maxVisible := m.getMaxVisibleCommits()
		if m.cursor >= m.logViewState.scrollOffset+maxVisible {
			m.logViewState.scrollOffset = m.cursor - maxVisible + 1
		}
	}
	return m, nil
}

// Diff view handlers
func (m *Model) handleDiffPrevFile() (tea.Model, tea.Cmd) {
	if len(m.currentDiff) > 1 && m.diffViewState.selectedFile > 0 {
		m.diffViewState.selectedFile--
		m.diffViewState.scrollOffset = 0
	}
	return m, nil
}

func (m *Model) handleDiffNextFile() (tea.Model, tea.Cmd) {
	if len(m.currentDiff) > 1 && m.diffViewState.selectedFile < len(m.currentDiff)-1 {
		m.diffViewState.selectedFile++
		m.diffViewState.scrollOffset = 0
	}
	return m, nil
}

// Log view handlers
func (m *Model) handleShowCommitDetails() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.commitHistory) {
		commit := m.commitHistory[m.cursor]
		// Switch to diff view and show commit diff
		m = m.switchView(ViewModeDiff)
		return m, m.showCommitDiff(commit.Hash)
	}
	return m, nil
}

func (m *Model) handleCopyCommitHash() (tea.Model, tea.Cmd) {
	if m.cursor < len(m.commitHistory) {
		commit := m.commitHistory[m.cursor]
		logger.LogUIAction("commit_hash_copied", map[string]interface{}{
			"hash": commit.Hash,
		})
		m.statusMessage = "Commit hash copied: " + commit.ShortHash
	}
	return m, nil
}

// Status view handlers
func (m *Model) handleQuickCommit() (tea.Model, tea.Cmd) {
	if m.generatedMessage != "" {
		return m, m.commitStagedChanges()
	} else {
		m.loading = true
		m.loadingMessage = "Generating commit message..."
		return m, tea.Sequence(
			m.generateCommitMessage(),
			func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
			},
		)
	}
}
