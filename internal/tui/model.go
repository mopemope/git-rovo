package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/llm"
	"github.com/mopemope/git-rovo/internal/logger"
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ViewModeStatus ViewMode = iota
	ViewModeDiff
	ViewModeLog
	ViewModeHelp
)

// Model represents the main TUI model
type Model struct {
	// Core components
	config            *config.Config
	repo              *git.Repository
	llmClient         *llm.Client
	keyBindingManager *KeyBindingManager

	// UI state
	currentView   ViewMode
	mainViewState *MainViewState
	diffViewState *DiffViewState
	logViewState  *LogViewState
	width         int
	height        int
	cursor        int
	selected      map[int]bool

	// Data
	fileStatus    []git.FileStatus
	commitHistory []git.CommitInfo
	currentDiff   []git.DiffInfo
	statusMessage string
	errorMessage  string

	// Loading states
	loading        bool
	loadingMessage string

	// Generated commit message
	generatedMessage  string
	messageConfidence float32

	// Styles
	styles Styles
}

// Styles holds all the styling for the TUI
type Styles struct {
	Base        lipgloss.Style
	Header      lipgloss.Style
	Footer      lipgloss.Style
	Selected    lipgloss.Style
	Unselected  lipgloss.Style
	Error       lipgloss.Style
	Success     lipgloss.Style
	Warning     lipgloss.Style
	Info        lipgloss.Style
	Staged      lipgloss.Style
	Modified    lipgloss.Style
	Untracked   lipgloss.Style
	Deleted     lipgloss.Style
	Loading     lipgloss.Style
	Help        lipgloss.Style
	Diff        lipgloss.Style
	DiffAdd     lipgloss.Style
	DiffRemove  lipgloss.Style
	DiffContext lipgloss.Style
}

// NewModel creates a new TUI model
func NewModel(cfg *config.Config, repo *git.Repository, llmClient *llm.Client) *Model {
	model := &Model{
		config:            cfg,
		repo:              repo,
		llmClient:         llmClient,
		keyBindingManager: NewKeyBindingManager(cfg),
		currentView:       ViewModeStatus,
		selected:          make(map[int]bool),
		styles:            NewStyles(),
	}
	model.initMainViewState()
	model.initDetailedViewStates()
	return model
}

// NewStyles creates the default styles with Catppuccin Mocha theme
func NewStyles() Styles {
	return Styles{
		Base: lipgloss.NewStyle(),
		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(CatppuccinText)).
			Padding(1, 2).
			MarginBottom(1),
		Footer: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinOverlay0)).
			Padding(0, 1),
		Selected: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(CatppuccinText)),
		Unselected: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinSubtext1)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinRed)).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinGreen)).
			Bold(true),
		Warning: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinYellow)).
			Bold(true),
		Info: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinBlue)),
		Staged: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinGreen)),
		Modified: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinYellow)),
		Untracked: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinRed)),
		Deleted: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinRed)).
			Strikethrough(true),
		Loading: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinBlue)).
			Bold(true),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinOverlay0)),
		Diff: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinSubtext1)),
		DiffAdd: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinGreen)),
		DiffRemove: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinRed)),
		DiffContext: lipgloss.NewStyle().
			Foreground(lipgloss.Color(CatppuccinOverlay0)),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	logger.LogUIAction("app_start", map[string]interface{}{
		"view": "status",
	})
	return tea.Batch(
		m.refreshStatus(),
		m.refreshCommitHistory(),
	)
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case statusRefreshedMsg:
		m.fileStatus = msg.files
		m.loading = false
		m.errorMessage = ""
		return m, nil

	case commitHistoryRefreshedMsg:
		m.commitHistory = msg.commits
		return m, nil

	case diffRefreshedMsg:
		m.currentDiff = msg.diffs
		m.diffViewState.isStaged = msg.staged
		m.diffViewState.currentCommit = "" // Clear commit hash for regular diffs
		return m, nil

	case commitDiffRefreshedMsg:
		m.currentDiff = msg.diffs
		m.diffViewState.isStaged = false
		m.diffViewState.selectedFile = 0
		m.diffViewState.scrollOffset = 0
		m.diffViewState.currentCommit = msg.commitHash
		m.statusMessage = fmt.Sprintf("Showing diff for commit: %s", msg.commitHash[:8])
		return m, nil

	case errorMsg:
		m.errorMessage = msg.error
		m.loading = false
		return m, nil

	case commitMessageGeneratedMsg:
		m.generatedMessage = msg.message
		m.messageConfidence = msg.confidence
		m.loading = false
		m.statusMessage = fmt.Sprintf("Generated commit message (confidence: %.1f%%)", msg.confidence*100)
		return m, nil

	case operationCompletedMsg:
		m.statusMessage = msg.message
		m.loading = false
		return m, tea.Batch(
			m.refreshStatus(),
			m.refreshCommitHistory(),
		)

	case autoGenerateAndCommitMsg:
		// Start auto-generation process
		m.loading = true
		m.loadingMessage = "Generating commit message and committing..."
		return m, m.generateCommitMessageForAutoCommit()

	case commitAfterGenerationMsg:
		// Commit with the generated message
		m.generatedMessage = msg.message
		m.messageConfidence = msg.confidence
		m.loading = false
		return m, m.performAutoCommit()
	}

	return m, nil
}

// View renders the current view
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var content string
	switch m.currentView {
	case ViewModeStatus:
		content = m.renderEnhancedStatusView()
	case ViewModeDiff:
		content = m.renderEnhancedDiffView()
	case ViewModeLog:
		content = m.renderEnhancedLogView()
	case ViewModeHelp:
		content = m.renderHelpView()
	default:
		content = m.renderEnhancedStatusView()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(),
		content,
		m.renderFooter(),
	)
}

// renderHeader renders the header
func (m *Model) renderHeader() string {
	branch, _ := m.repo.GetCurrentBranch()
	workDir := m.repo.GetWorkDir()

	// Choose banner size based on terminal width
	var banner string
	if m.width >= 80 {
		// Full ASCII art banner for larger terminals
		banner = `
 ██████╗ ██╗████████╗    ██████╗  ██████╗ ██╗   ██╗ ██████╗ 
██╔════╝ ██║╚══██╔══╝    ██╔══██╗██╔═══██╗██║   ██║██╔═══██╗
██║  ███╗██║   ██║ █████╗██████╔╝██║   ██║██║   ██║██║   ██║
╚██████╔╝██║   ██║ ╚════╝██║  ██║╚██████╔╝ ╚████╔╝ ╚██████╔╝
 ╚═════╝ ╚═╝   ╚═╝       ╚═╝  ╚═╝ ╚═════╝   ╚═══╝   ╚═════╝ `
	} else {
		// Compact banner for smaller terminals
		banner = `
 ██████╗ ██╗████████╗    ██████╗  ██████╗ ██╗   ██╗ ██████╗ 
██╔════╝ ██║╚══██╔══╝    ██╔══██╗██╔═══██╗██║   ██║██╔═══██╗
╚██████╔╝██║   ██║       ██║  ██║╚██████╔╝ ╚████╔╝ ╚██████╔╝
 ╚═════╝ ╚═╝   ╚═╝       ╚═╝  ╚═╝ ╚═════╝   ╚═══╝   ╚═════╝ `
	}

	// Repository info
	repoInfo := fmt.Sprintf("Repository: %s [%s]", workDir, branch)

	if m.loading {
		repoInfo += " " + m.styles.Loading.Render("(loading...)")
	}

	// Combine banner and repo info with proper spacing
	content := strings.TrimSpace(banner) + "\n\n" + repoInfo

	return m.styles.Header.Width(m.width).Render(content)
}

// renderFooter renders the footer with key bindings
func (m *Model) renderFooter() string {
	var footer strings.Builder

	// Error and status messages
	if m.errorMessage != "" {
		footer.WriteString(m.styles.Error.Render("Error: " + m.errorMessage))
		footer.WriteString("\n")
	}
	if m.statusMessage != "" {
		footer.WriteString(m.styles.Success.Render(m.statusMessage))
		footer.WriteString("\n")
	}

	// Key bindings from key binding manager
	keyBindings := m.keyBindingManager.GetFooterText(m.currentView)
	footer.WriteString(m.styles.Footer.Width(m.width).Render(keyBindings))

	return footer.String()
}

// handleKeyPress handles key press events
func (m *Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Use unified key handling system
	return m.handleUnifiedKeyPress(msg)
}

// switchView switches to a different view mode
func (m *Model) switchView(view ViewMode) *Model {
	m.currentView = view
	m.cursor = 0
	m.errorMessage = ""
	m.statusMessage = ""

	logger.LogUIAction("view_switch", map[string]interface{}{
		"view": viewModeToString(view),
	})

	return m
}

// viewModeToString converts ViewMode to string
func viewModeToString(mode ViewMode) string {
	switch mode {
	case ViewModeStatus:
		return "status"
	case ViewModeDiff:
		return "diff"
	case ViewModeLog:
		return "log"
	case ViewModeHelp:
		return "help"
	default:
		return "unknown"
	}
}

// Message types for async operations
type statusRefreshedMsg struct {
	files []git.FileStatus
}

type commitHistoryRefreshedMsg struct {
	commits []git.CommitInfo
}

type diffRefreshedMsg struct {
	diffs  []git.DiffInfo
	staged bool
}

type commitDiffRefreshedMsg struct {
	diffs      []git.DiffInfo
	commitHash string
}

type errorMsg struct {
	error string
}

type commitMessageGeneratedMsg struct {
	message    string
	confidence float32
}

type operationCompletedMsg struct {
	message string
}

type autoGenerateAndCommitMsg struct{}

type commitAfterGenerationMsg struct {
	message    string
	confidence float32
}

// refreshStatus refreshes the git status
func (m *Model) refreshStatus() tea.Cmd {
	return func() tea.Msg {
		files, err := m.repo.GetStatus()
		if err != nil {
			return errorMsg{error: err.Error()}
		}
		return statusRefreshedMsg{files: files}
	}
}

// refreshCommitHistory refreshes the commit history
func (m *Model) refreshCommitHistory() tea.Cmd {
	return func() tea.Msg {
		commits, err := m.repo.GetCommitHistory(20) // Get last 20 commits
		if err != nil {
			return errorMsg{error: err.Error()}
		}
		return commitHistoryRefreshedMsg{commits: commits}
	}
}

// refreshDiff refreshes the diff for selected files
func (m *Model) refreshDiff(staged bool, files ...string) tea.Cmd {
	return func() tea.Msg {
		var diffs []git.DiffInfo
		var err error

		// Check if any of the files are untracked
		if len(files) > 0 {
			status, statusErr := m.repo.GetStatus()
			if statusErr != nil {
				return errorMsg{error: statusErr.Error()}
			}

			// Separate tracked and untracked files
			var trackedFiles, untrackedFiles []string
			untrackedMap := make(map[string]bool)

			for _, file := range status {
				if file.Status == "??" {
					untrackedMap[file.Path] = true
				}
			}

			for _, file := range files {
				if untrackedMap[file] {
					untrackedFiles = append(untrackedFiles, file)
				} else {
					trackedFiles = append(trackedFiles, file)
				}
			}

			// Get diff for tracked files
			if len(trackedFiles) > 0 {
				trackedDiffs, err := m.repo.GetDiff(staged, trackedFiles...)
				if err != nil {
					return errorMsg{error: err.Error()}
				}
				diffs = append(diffs, trackedDiffs...)
			}

			// Get diff for untracked files
			if len(untrackedFiles) > 0 {
				untrackedDiffs, err := m.repo.GetUntrackedFileDiff(untrackedFiles...)
				if err != nil {
					return errorMsg{error: err.Error()}
				}
				diffs = append(diffs, untrackedDiffs...)
			}
		} else {
			// Get all diffs (no specific files)
			diffs, err = m.repo.GetDiff(staged)
			if err != nil {
				return errorMsg{error: err.Error()}
			}

			// Also get untracked file diffs if not showing staged changes
			if !staged {
				untrackedDiffs, err := m.repo.GetUntrackedFileDiff()
				if err != nil {
					return errorMsg{error: err.Error()}
				}
				diffs = append(diffs, untrackedDiffs...)
			}
		}

		return diffRefreshedMsg{diffs: diffs, staged: staged}
	}
}

// generateCommitMessage generates a commit message using LLM
func (m *Model) generateCommitMessage() tea.Cmd {
	return func() tea.Msg {
		// Get staged diff
		diffs, err := m.repo.GetDiff(true)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to get diff: %v", err)}
		}

		// Also get staged untracked files (newly added files)
		status, err := m.repo.GetStatus()
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to get status: %v", err)}
		}

		// Find staged untracked files (status "A")
		var stagedUntrackedFiles []string
		for _, file := range status {
			if file.Status == "A" {
				stagedUntrackedFiles = append(stagedUntrackedFiles, file.Path)
			}
		}

		// Get diff for staged untracked files
		if len(stagedUntrackedFiles) > 0 {
			untrackedDiffs, err := m.repo.GetUntrackedFileDiff(stagedUntrackedFiles...)
			if err == nil {
				diffs = append(diffs, untrackedDiffs...)
			}
		}

		if len(diffs) == 0 {
			return errorMsg{error: "No staged changes to generate commit message for"}
		}

		// Build diff content
		var diffContent strings.Builder
		for _, diff := range diffs {
			diffContent.WriteString(diff.Content)
			diffContent.WriteString("\n")
		}

		// Create request
		request := &llm.CommitMessageRequest{
			Diff:        diffContent.String(),
			Language:    m.config.LLM.Language,
			MaxTokens:   m.config.LLM.OpenAI.MaxTokens,
			Temperature: m.config.LLM.OpenAI.Temperature,
		}

		// Generate message
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := m.llmClient.GenerateCommitMessage(ctx, request, "")
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to generate commit message: %v", err)}
		}

		return commitMessageGeneratedMsg{
			message:    formatCommitMessage(response.Message),
			confidence: response.Confidence,
		}
	}
}

// generateCommitMessageForAutoCommit generates a commit message for auto-commit
func (m *Model) generateCommitMessageForAutoCommit() tea.Cmd {
	return func() tea.Msg {
		// Get staged diff
		diffs, err := m.repo.GetDiff(true)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to get diff: %v", err)}
		}

		if len(diffs) == 0 {
			return errorMsg{error: "No staged changes to generate commit message for"}
		}

		// Build diff content
		var diffContent strings.Builder
		for _, diff := range diffs {
			diffContent.WriteString(fmt.Sprintf("File: %s\n", diff.FilePath))
			diffContent.WriteString(fmt.Sprintf("Status: %s\n", diff.Status))
			if diff.Content != "" {
				diffContent.WriteString("Changes:\n")
				diffContent.WriteString(diff.Content)
				diffContent.WriteString("\n\n")
			}
		}

		// Create request
		request := &llm.CommitMessageRequest{
			Diff:     diffContent.String(),
			Language: m.config.LLM.Language,
		}

		// Generate message
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		response, err := m.llmClient.GenerateCommitMessage(ctx, request, "")
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to generate commit message: %v", err)}
		}

		return commitAfterGenerationMsg{
			message:    formatCommitMessage(response.Message),
			confidence: response.Confidence,
		}
	}
}

// performAutoCommit performs the actual commit after message generation
func (m *Model) performAutoCommit() tea.Cmd {
	return func() tea.Msg {
		// Perform commit
		err := m.repo.Commit(m.generatedMessage)
		if err != nil {
			return errorMsg{error: fmt.Sprintf("Failed to commit: %v", err)}
		}

		logger.LogUIAction("auto_commit_created", map[string]interface{}{
			"message":    m.generatedMessage,
			"confidence": m.messageConfidence,
		})

		// Clear generated message after successful commit
		m.generatedMessage = ""
		m.messageConfidence = 0

		return operationCompletedMsg{message: "Auto-commit completed successfully"}
	}
}

// formatCommitMessage ensures the commit message follows proper Git commit format
// It checks if there's a blank line between the subject (first line) and body (subsequent lines)
// and adds one if missing
func formatCommitMessage(message string) string {
	lines := strings.Split(message, "\n")

	// If there's only one line, no formatting needed
	if len(lines) <= 1 {
		return message
	}

	// Remove any trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	// If there's only one line after trimming, no formatting needed
	if len(lines) <= 1 {
		return strings.Join(lines, "\n")
	}

	// Check if the second line is empty (proper format)
	if len(lines) >= 2 && strings.TrimSpace(lines[1]) == "" {
		// Already properly formatted
		return strings.Join(lines, "\n")
	}

	// Need to add blank line between subject and body
	subject := lines[0]
	body := lines[1:]

	// Create properly formatted message
	formattedLines := []string{subject, ""} // subject + blank line
	formattedLines = append(formattedLines, body...)

	return strings.Join(formattedLines, "\n")
}
