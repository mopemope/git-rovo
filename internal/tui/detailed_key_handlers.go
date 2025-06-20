package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/logger"
)

// Helper methods for enhanced views

func (m *Model) getMaxDiffScroll() int {
	if len(m.currentDiff) == 0 {
		return 0
	}

	diff := m.currentDiff[m.diffViewState.selectedFile]
	lines := strings.Split(diff.Content, "\n")
	maxVisible := m.getMaxVisibleLines()

	if len(lines) <= maxVisible {
		return 0
	}

	return len(lines) - maxVisible
}

func (m *Model) cycleDiffViewMode() {
	switch m.diffViewState.viewMode {
	case DiffViewModeUnified:
		m.diffViewState.viewMode = DiffViewModeSideBySide
	case DiffViewModeSideBySide:
		m.diffViewState.viewMode = DiffViewModeWordDiff
	case DiffViewModeWordDiff:
		m.diffViewState.viewMode = DiffViewModeUnified
	}

	logger.LogUIAction("diff_view_mode_changed", map[string]interface{}{
		"mode": m.getDiffViewModeText(),
	})
}

func (m *Model) showCommitDiff(commitHash string) tea.Cmd {
	return func() tea.Msg {
		// Get diff for specific commit
		output, err := m.repo.RunGitCommand("show", "--no-color", commitHash)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to get commit diff: %v", err)}
		}

		// Parse the diff output
		diffs := m.repo.ParseDiff(output)

		logger.LogUIAction("commit_diff_viewed", map[string]interface{}{
			"commit": commitHash,
		})

		return commitDiffRefreshedMsg{diffs: diffs, commitHash: commitHash}
	}
}
