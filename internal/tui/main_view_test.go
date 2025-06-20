package tui

import (
	"path/filepath"
	"testing"

	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/llm"
	"github.com/mopemope/git-rovo/internal/logger"
)

func setupMainViewTest(t *testing.T) *Model {
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

	// Create mock repository
	repo := &git.Repository{}

	// Create LLM client
	client := llm.NewClient()
	mockProvider := &MockLLMProvider{name: "test"}
	_ = client.RegisterProvider("test", mockProvider)

	return NewModel(cfg, repo, client)
}

func TestNewMainViewState(t *testing.T) {
	state := NewMainViewState()

	if state == nil {
		t.Fatal("Expected main view state to be created")
	}

	if !state.showStagedSection {
		t.Error("Expected staged section to be shown by default")
	}

	if !state.showUnstagedSection {
		t.Error("Expected unstaged section to be shown by default")
	}

	if !state.showUntrackedSection {
		t.Error("Expected untracked section to be shown by default")
	}

	if state.expandedSections == nil {
		t.Error("Expected expanded sections map to be initialized")
	}

	if state.sortBy != SortByName {
		t.Error("Expected default sort to be by name")
	}

	if !state.showStats {
		t.Error("Expected stats to be shown by default")
	}
}

func TestInitMainViewState(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	if model.mainViewState == nil {
		t.Error("Expected main view state to be initialized")
	}

	// Test that it's properly initialized
	if !model.mainViewState.showStagedSection {
		t.Error("Expected staged section to be shown")
	}
}

func TestGroupFiles(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set up test file status
	model.fileStatus = []git.FileStatus{
		{Path: "staged.txt", Status: "M ", Staged: true, Modified: false},
		{Path: "modified.txt", Status: " M", Staged: false, Modified: true},
		{Path: "untracked.txt", Status: "??", Staged: false, Modified: false},
		{Path: "staged_new.txt", Status: "A ", Staged: true, Modified: false},
	}

	staged, unstaged, untracked := model.groupFiles()

	if len(staged) != 2 {
		t.Errorf("Expected 2 staged files, got %d", len(staged))
	}

	if len(unstaged) != 1 {
		t.Errorf("Expected 1 unstaged file, got %d", len(unstaged))
	}

	if len(untracked) != 1 {
		t.Errorf("Expected 1 untracked file, got %d", len(untracked))
	}

	// Check specific files
	if staged[0].Path != "staged.txt" && staged[1].Path != "staged.txt" {
		t.Error("Expected staged.txt to be in staged files")
	}

	if unstaged[0].Path != "modified.txt" {
		t.Error("Expected modified.txt to be in unstaged files")
	}

	if untracked[0].Path != "untracked.txt" {
		t.Error("Expected untracked.txt to be in untracked files")
	}
}

func TestToggleSection(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	sectionKey := "test_section"

	// Initially should be false (not expanded)
	if model.mainViewState.expandedSections[sectionKey] {
		t.Error("Expected section to be collapsed initially")
	}

	// Toggle to expand
	model.toggleSection(sectionKey)
	if !model.mainViewState.expandedSections[sectionKey] {
		t.Error("Expected section to be expanded after toggle")
	}

	// Toggle to collapse
	model.toggleSection(sectionKey)
	if model.mainViewState.expandedSections[sectionKey] {
		t.Error("Expected section to be collapsed after second toggle")
	}
}

func TestRenderEmptyRepository(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set dimensions
	model.width = 80
	model.height = 24

	// Render empty repository view
	content := model.renderEmptyRepository()

	if content == "" {
		t.Error("Expected non-empty content for empty repository")
	}

	// Check for expected content
	expectedStrings := []string{
		"Working directory is clean",
		"Tips:",
	}

	for _, expected := range expectedStrings {
		if !contains(content, expected) {
			t.Errorf("Expected content to contain '%s'", expected)
		}
	}
}

func TestRenderRepositorySummary(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set up test file status
	model.fileStatus = []git.FileStatus{
		{Path: "staged.txt", Status: "M ", Staged: true, Modified: false},
		{Path: "modified.txt", Status: " M", Staged: false, Modified: true},
		{Path: "untracked.txt", Status: "??", Staged: false, Modified: false},
	}

	summary := model.renderRepositorySummary()

	if summary == "" {
		t.Error("Expected non-empty repository summary")
	}

	// Check for file counts
	expectedStrings := []string{
		"1 staged",
		"1 modified",
		"1 untracked",
	}

	for _, expected := range expectedStrings {
		if !contains(summary, expected) {
			t.Errorf("Expected summary to contain '%s'", expected)
		}
	}
}

func TestRenderCommitMessageSection(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set dimensions
	model.width = 80
	model.height = 24

	// Set generated message
	model.generatedMessage = "feat: add new feature"
	model.messageConfidence = 0.85

	content := model.renderCommitMessageSection()

	if content == "" {
		t.Error("Expected non-empty commit message section")
	}

	// Check for expected content
	expectedStrings := []string{
		"Generated Commit Message",
		"feat: add new feature",
		"85.0%",
	}

	for _, expected := range expectedStrings {
		if !contains(content, expected) {
			t.Errorf("Expected content to contain '%s'", expected)
		}
	}
}

func TestRenderQuickActions(t *testing.T) {
	model := setupMainViewTest(t)
	defer func() { _ = logger.Close() }()

	content := model.renderQuickActions()

	if content == "" {
		t.Error("Expected non-empty quick actions")
	}

	// Check for expected actions
	expectedActions := []string{
		"Space",
		"Stage current file",
		"Generate commit message",
	}

	for _, expected := range expectedActions {
		if !contains(content, expected) {
			t.Errorf("Expected content to contain '%s'", expected)
		}
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
