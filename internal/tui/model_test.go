package tui

import (
	"context"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/llm"
	"github.com/mopemope/git-rovo/internal/logger"
)

func setupTUITest(t *testing.T) (*config.Config, *git.Repository, *llm.Client) {
	// Initialize logger for testing
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	if err := logger.Init(logPath, "info"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "openai",
			OpenAI: config.OpenAIConfig{
				APIKey:      "test-api-key",
				Model:       "gpt-4o-mini",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			Language: "english",
		},
	}

	// Create mock repository (we'll use a real one for testing)
	// In a real test, you might want to create a temporary git repo
	repo := &git.Repository{}

	// Create LLM client
	client := llm.NewClient()
	mockProvider := &MockLLMProvider{name: "test"}
	_ = client.RegisterProvider("test", mockProvider)

	return cfg, repo, client
}

// MockLLMProvider for testing
type MockLLMProvider struct {
	name string
}

func (m *MockLLMProvider) GenerateCommitMessage(ctx context.Context, request *llm.CommitMessageRequest) (*llm.CommitMessageResponse, error) {
	return &llm.CommitMessageResponse{
		Message:    "feat: add new feature",
		Confidence: 0.9,
		TokensUsed: 50,
		Provider:   m.name,
	}, nil
}

func (m *MockLLMProvider) GetProviderName() string {
	return m.name
}

func (m *MockLLMProvider) Close() error {
	return nil
}

func TestNewModel(t *testing.T) {
	cfg, repo, client := setupTUITest(t)
	defer func() { _ = logger.Close() }()

	model := NewModel(cfg, repo, client)

	if model == nil {
		t.Fatal("Expected model to be created")
	}

	if model.config != cfg {
		t.Error("Expected config to be set")
	}

	if model.repo != repo {
		t.Error("Expected repository to be set")
	}

	if model.llmClient != client {
		t.Error("Expected LLM client to be set")
	}

	if model.currentView != ViewModeStatus {
		t.Error("Expected default view to be status")
	}

	if model.selected == nil {
		t.Error("Expected selected map to be initialized")
	}
}

func TestNewStyles(t *testing.T) {
	styles := NewStyles()

	// Test that styles are initialized (basic check)
	if styles.Header.GetForeground() == nil {
		t.Error("Expected header style to have foreground color")
	}
}

func TestViewModeToString(t *testing.T) {
	testCases := []struct {
		mode     ViewMode
		expected string
	}{
		{ViewModeStatus, "status"},
		{ViewModeDiff, "diff"},
		{ViewModeLog, "log"},
		{ViewModeHelp, "help"},
		{ViewMode(999), "unknown"},
	}

	for _, tc := range testCases {
		result := viewModeToString(tc.mode)
		if result != tc.expected {
			t.Errorf("viewModeToString(%v) = %s, expected %s", tc.mode, result, tc.expected)
		}
	}
}

func TestSwitchView(t *testing.T) {
	cfg, repo, client := setupTUITest(t)
	defer func() { _ = logger.Close() }()

	model := NewModel(cfg, repo, client)

	// Test switching to different views
	testViews := []ViewMode{ViewModeDiff, ViewModeLog, ViewModeHelp, ViewModeStatus}

	for _, view := range testViews {
		model = model.switchView(view)
		if model.currentView != view {
			t.Errorf("Expected current view to be %v, got %v", view, model.currentView)
		}

		// Check that cursor is reset
		if model.cursor != 0 {
			t.Error("Expected cursor to be reset to 0 when switching views")
		}

		// Check that messages are cleared
		if model.errorMessage != "" {
			t.Error("Expected error message to be cleared when switching views")
		}

		if model.statusMessage != "" {
			t.Error("Expected status message to be cleared when switching views")
		}
	}
}

func TestHandleKeyPress(t *testing.T) {
	cfg, repo, client := setupTUITest(t)
	defer func() { _ = logger.Close() }()

	model := NewModel(cfg, repo, client)

	// Test global quit key
	quitMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	newModel, cmd := model.handleKeyPress(quitMsg)

	if newModel == nil {
		t.Error("Expected model to be returned")
	}

	// The command should be tea.Quit, but we can't easily test that
	if cmd == nil {
		t.Error("Expected command to be returned for quit")
	}

	// Test help key
	helpMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}
	newModel, _ = model.handleKeyPress(helpMsg)

	if newModel.(*Model).currentView != ViewModeHelp {
		t.Error("Expected view to switch to help")
	}
}

func TestUpdate(t *testing.T) {
	cfg, repo, client := setupTUITest(t)
	defer func() { _ = logger.Close() }()

	model := NewModel(cfg, repo, client)

	// Test window size message
	sizeMsg := tea.WindowSizeMsg{Width: 80, Height: 24}
	newModel, cmd := model.Update(sizeMsg)

	if cmd != nil {
		t.Error("Expected no command for window size message")
	}

	updatedModel := newModel.(*Model)
	if updatedModel.width != 80 {
		t.Errorf("Expected width to be 80, got %d", updatedModel.width)
	}

	if updatedModel.height != 24 {
		t.Errorf("Expected height to be 24, got %d", updatedModel.height)
	}

	// Test error message
	errorMsg := errorMsg{error: "test error"}
	newModel, _ = model.Update(errorMsg)

	updatedModel = newModel.(*Model)
	if updatedModel.errorMessage != "test error" {
		t.Errorf("Expected error message to be 'test error', got '%s'", updatedModel.errorMessage)
	}

	if updatedModel.loading != false {
		t.Error("Expected loading to be false after error")
	}
}

func TestView(t *testing.T) {
	cfg, repo, client := setupTUITest(t)
	defer func() { _ = logger.Close() }()

	model := NewModel(cfg, repo, client)

	// Test with zero dimensions (should return loading message)
	view := model.View()
	if view != "Loading..." {
		t.Errorf("Expected 'Loading...' for zero dimensions, got '%s'", view)
	}

	// Set dimensions
	model.width = 80
	model.height = 24

	// Test status view
	model.currentView = ViewModeStatus
	view = model.View()
	if view == "" {
		t.Error("Expected non-empty view for status mode")
	}

	// Test other views
	views := []ViewMode{ViewModeDiff, ViewModeLog, ViewModeHelp}
	for _, viewMode := range views {
		model.currentView = viewMode
		view = model.View()
		if view == "" {
			t.Errorf("Expected non-empty view for mode %v", viewMode)
		}
	}
}
