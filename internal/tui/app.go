package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/llm"
	"github.com/mopemope/git-rovo/internal/logger"
)

// App represents the TUI application
type App struct {
	config    *config.Config
	repo      *git.Repository
	llmClient *llm.Client
	program   *tea.Program
}

// NewApp creates a new TUI application
func NewApp(cfg *config.Config, workDir string) (*App, error) {
	// Initialize Git repository
	repo, err := git.New(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// Initialize LLM client
	llmClient, err := llm.CreateClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}

	return &App{
		config:    cfg,
		repo:      repo,
		llmClient: llmClient,
	}, nil
}

// Run starts the TUI application
func (a *App) Run() error {
	// Create the model
	model := NewModel(a.config, a.repo, a.llmClient)

	// Create the program
	a.program = tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Use alternate screen buffer
		tea.WithMouseCellMotion(), // Enable mouse support
	)

	// Log application start
	logger.LogAppStart("dev", a.config.Logger.FilePath)

	// Run the program
	finalModel, err := a.program.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	// Log application stop
	logger.LogAppStop()

	// Clean up
	if err := a.cleanup(finalModel); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	return nil
}

// cleanup performs cleanup operations
func (a *App) cleanup(finalModel tea.Model) error {
	// Close LLM client
	if err := a.llmClient.Close(); err != nil {
		logger.Error("Failed to close LLM client", "error", err.Error())
	}

	return nil
}

// Quit quits the application programmatically
func (a *App) Quit() {
	if a.program != nil {
		a.program.Quit()
	}
}

// Send sends a message to the application
func (a *App) Send(msg tea.Msg) {
	if a.program != nil {
		a.program.Send(msg)
	}
}

// GetConfig returns the application configuration
func (a *App) GetConfig() *config.Config {
	return a.config
}

// GetRepository returns the git repository
func (a *App) GetRepository() *git.Repository {
	return a.repo
}

// GetLLMClient returns the LLM client
func (a *App) GetLLMClient() *llm.Client {
	return a.llmClient
}
