package tui

import (
	"path/filepath"
	"testing"

	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/llm"
	"github.com/mopemope/git-rovo/internal/logger"
)

func setupDetailedViewTest(t *testing.T) *Model {
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

func TestNewDiffViewState(t *testing.T) {
	state := NewDiffViewState()

	if state == nil {
		t.Fatal("Expected diff view state to be created")
	}

	if state.scrollOffset != 0 {
		t.Error("Expected scroll offset to be 0 initially")
	}

	if state.selectedFile != 0 {
		t.Error("Expected selected file to be 0 initially")
	}

	if !state.showLineNumbers {
		t.Error("Expected line numbers to be shown by default")
	}

	if !state.showContext {
		t.Error("Expected context to be shown by default")
	}

	if state.contextLines != 3 {
		t.Error("Expected default context lines to be 3")
	}

	if state.wrapLines {
		t.Error("Expected line wrapping to be disabled by default")
	}

	if !state.showStats {
		t.Error("Expected stats to be shown by default")
	}

	if state.viewMode != DiffViewModeUnified {
		t.Error("Expected default view mode to be unified")
	}
}

func TestNewLogViewState(t *testing.T) {
	state := NewLogViewState()

	if state == nil {
		t.Fatal("Expected log view state to be created")
	}

	if state.scrollOffset != 0 {
		t.Error("Expected scroll offset to be 0 initially")
	}

	if state.selectedCommit != 0 {
		t.Error("Expected selected commit to be 0 initially")
	}

	if !state.showDetails {
		t.Error("Expected details to be shown by default")
	}

	if state.showGraph {
		t.Error("Expected graph to be hidden by default")
	}

	if !state.showStats {
		t.Error("Expected stats to be shown by default")
	}

	if state.maxCommits != 50 {
		t.Error("Expected default max commits to be 50")
	}
}

func TestInitDetailedViewStates(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	if model.diffViewState == nil {
		t.Error("Expected diff view state to be initialized")
	}

	if model.logViewState == nil {
		t.Error("Expected log view state to be initialized")
	}
}

func TestGetStatusText(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	testCases := []struct {
		status   string
		expected string
	}{
		{"A", "Added"},
		{"M", "Modified"},
		{"D", "Deleted"},
		{"R", "Renamed"},
		{"C", "Copied"},
		{"X", "Changed"}, // Unknown status should default to "Changed"
	}

	for _, tc := range testCases {
		result := model.getStatusText(tc.status)
		// We can't easily test the styled text, but we can check it's not empty
		if result == "" {
			t.Errorf("Expected non-empty status text for status '%s'", tc.status)
		}
	}
}

func TestGetDiffViewModeText(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	testCases := []struct {
		mode     DiffViewMode
		expected string
	}{
		{DiffViewModeUnified, "Unified"},
		{DiffViewModeSideBySide, "Side-by-side"},
		{DiffViewModeWordDiff, "Word diff"},
		{DiffViewMode(999), "Unknown"},
	}

	for _, tc := range testCases {
		model.diffViewState.viewMode = tc.mode
		result := model.getDiffViewModeText()
		if result != tc.expected {
			t.Errorf("getDiffViewModeText() for mode %v = %s, expected %s", tc.mode, result, tc.expected)
		}
	}
}

func TestCycleDiffViewMode(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	// Start with unified mode
	if model.diffViewState.viewMode != DiffViewModeUnified {
		t.Error("Expected initial mode to be unified")
	}

	// Cycle to side-by-side
	model.cycleDiffViewMode()
	if model.diffViewState.viewMode != DiffViewModeSideBySide {
		t.Error("Expected mode to cycle to side-by-side")
	}

	// Cycle to word diff
	model.cycleDiffViewMode()
	if model.diffViewState.viewMode != DiffViewModeWordDiff {
		t.Error("Expected mode to cycle to word diff")
	}

	// Cycle back to unified
	model.cycleDiffViewMode()
	if model.diffViewState.viewMode != DiffViewModeUnified {
		t.Error("Expected mode to cycle back to unified")
	}
}

func TestGetMaxVisibleLines(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set height
	model.height = 24

	maxLines := model.getMaxVisibleLines()
	expected := 24 - 8 // height - reserved space
	if maxLines != expected {
		t.Errorf("Expected max visible lines to be %d, got %d", expected, maxLines)
	}
}

func TestGetMaxVisibleCommits(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set height
	model.height = 24

	maxCommits := model.getMaxVisibleCommits()
	expected := (24 - 6) / 4 // (height - reserved) / lines per commit
	if maxCommits != expected {
		t.Errorf("Expected max visible commits to be %d, got %d", expected, maxCommits)
	}
}

func TestGetVisibleLines(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	// Set up test data
	model.height = 20
	model.diffViewState.scrollOffset = 2

	lines := []string{"line1", "line2", "line3", "line4", "line5", "line6", "line7", "line8"}
	visible := model.getVisibleLines(lines)

	maxVisible := model.getMaxVisibleLines()
	expectedStart := 2
	expectedEnd := expectedStart + maxVisible
	if expectedEnd > len(lines) {
		expectedEnd = len(lines)
	}

	expectedLength := expectedEnd - expectedStart
	if len(visible) != expectedLength {
		t.Errorf("Expected %d visible lines, got %d", expectedLength, len(visible))
	}

	// Check first visible line
	if len(visible) > 0 && visible[0] != lines[expectedStart] {
		t.Errorf("Expected first visible line to be '%s', got '%s'", lines[expectedStart], visible[0])
	}
}

func TestParseHunkHeader(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	testCases := []struct {
		input    string
		expected string
	}{
		{"@@ -1,4 +1,6 @@ function test()", "@@ -1,4 +1,6 @@ function test()"},
		{"@@ -10,5 +10,8 @@", "@@ -10,5 +10,8 @@"},
		{"not a hunk header", "not a hunk header"},
		{"@@", "@@"},
	}

	for _, tc := range testCases {
		result := model.parseHunkHeader(tc.input)
		if result != tc.expected {
			t.Errorf("parseHunkHeader(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestRenderNoDiffAvailable(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	content := model.renderNoDiffAvailable()

	if content == "" {
		t.Error("Expected non-empty content for no diff available")
	}

	// Check for expected content
	expectedStrings := []string{
		"No diff available",
		"Tips:",
		"Select a file",
	}

	for _, expected := range expectedStrings {
		if !containsSubstring(content, expected) {
			t.Errorf("Expected content to contain '%s'", expected)
		}
	}
}

func TestRenderNoCommitsAvailable(t *testing.T) {
	model := setupDetailedViewTest(t)
	defer func() { _ = logger.Close() }()

	content := model.renderNoCommitsAvailable()

	if content == "" {
		t.Error("Expected non-empty content for no commits available")
	}

	// Check for expected content
	expectedStrings := []string{
		"No commit history available",
		"new repository",
	}

	for _, expected := range expectedStrings {
		if !containsSubstring(content, expected) {
			t.Errorf("Expected content to contain '%s'", expected)
		}
	}
}
