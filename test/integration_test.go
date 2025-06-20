package test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/git"
	"github.com/mopemope/git-rovo/internal/llm"
	"github.com/mopemope/git-rovo/internal/logger"
	"github.com/mopemope/git-rovo/internal/tui"
)

// TestIntegration runs integration tests for the entire application
func TestIntegration(t *testing.T) {
	// Create temporary directory for test repository
	tempDir := t.TempDir()

	// Initialize logger
	logPath := filepath.Join(tempDir, "test.log")
	if err := logger.Init(logPath, "debug"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Create test Git repository
	repoDir := filepath.Join(tempDir, "test-repo")
	if err := createTestRepository(repoDir); err != nil {
		t.Fatalf("Failed to create test repository: %v", err)
	}

	// Test configuration loading
	t.Run("ConfigurationLoading", func(t *testing.T) {
		testConfigurationLoading(t, tempDir)
	})

	// Test Git operations
	t.Run("GitOperations", func(t *testing.T) {
		testGitOperations(t, repoDir)
	})

	// Test LLM integration
	t.Run("LLMIntegration", func(t *testing.T) {
		testLLMIntegration(t, tempDir)
	})

	// Test TUI components
	t.Run("TUIComponents", func(t *testing.T) {
		testTUIComponents(t, repoDir)
	})

	// Test end-to-end workflow
	t.Run("EndToEndWorkflow", func(t *testing.T) {
		testEndToEndWorkflow(t, repoDir)
	})
}

func createTestRepository(repoDir string) error {
	// Create directory
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return err
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return err
	}

	// Create initial file
	readmePath := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmePath, []byte("# Test Repository\n\nThis is a test repository.\n"), 0644); err != nil {
		return err
	}

	// Add and commit initial file
	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return err
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func testConfigurationLoading(t *testing.T, tempDir string) {
	configPath := filepath.Join(tempDir, "config.toml")

	// Test default configuration
	cfg := config.Default()
	if cfg == nil {
		t.Fatal("Expected default configuration to be created")
	}

	// Test saving configuration
	if err := config.Save(cfg, configPath); err != nil {
		t.Fatalf("Failed to save configuration: %v", err)
	}

	// Test loading configuration
	loadedCfg, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// Verify configuration values
	if loadedCfg.LLM.Provider != cfg.LLM.Provider {
		t.Errorf("Expected LLM provider %s, got %s", cfg.LLM.Provider, loadedCfg.LLM.Provider)
	}

	if loadedCfg.LLM.Language != cfg.LLM.Language {
		t.Errorf("Expected language %s, got %s", cfg.LLM.Language, loadedCfg.LLM.Language)
	}
}

func testGitOperations(t *testing.T, repoDir string) {
	// Create Git repository instance
	repo, err := git.New(repoDir)
	if err != nil {
		t.Fatalf("Failed to create Git repository: %v", err)
	}

	// Test getting status
	status, err := repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	// Initially should be clean
	if len(status) != 0 {
		t.Errorf("Expected clean repository, got %d files", len(status))
	}

	// Create a new file
	testFile := filepath.Join(repoDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test status after adding file
	status, err = repo.GetStatus()
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if len(status) != 1 {
		t.Errorf("Expected 1 file in status, got %d", len(status))
	}

	if status[0].Path != "test.txt" {
		t.Errorf("Expected file 'test.txt', got '%s'", status[0].Path)
	}

	// Test staging file
	if err := repo.StageFiles("test.txt"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Test getting diff
	diffs, err := repo.GetDiff(true) // staged diff
	if err != nil {
		t.Fatalf("Failed to get diff: %v", err)
	}

	if len(diffs) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(diffs))
	}

	// Test commit
	if err := repo.Commit("Add test file"); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Test commit history
	commits, err := repo.GetCommitHistory(5)
	if err != nil {
		t.Fatalf("Failed to get commit history: %v", err)
	}

	if len(commits) != 2 { // Initial commit + test commit
		t.Errorf("Expected 2 commits, got %d", len(commits))
	}
}

func testLLMIntegration(t *testing.T, tempDir string) {
	// Create test configuration
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

	// Create LLM client
	client, err := llm.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Test provider registration
	providers := client.ListProviders()
	if len(providers) == 0 {
		t.Error("Expected at least one provider to be registered")
	}

	// Test request validation
	request := &llm.CommitMessageRequest{
		Diff:        "diff --git a/test.txt b/test.txt\n+test content",
		Language:    "english",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	if err := llm.ValidateRequest(request); err != nil {
		t.Errorf("Expected valid request, got error: %v", err)
	}

	// Test prompt building
	prompt := llm.BuildPrompt(request)
	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}

	if !strings.Contains(prompt, "Conventional Commits") {
		t.Error("Expected prompt to mention Conventional Commits")
	}
}

func testTUIComponents(t *testing.T, repoDir string) {
	// Create test configuration
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

	// Create Git repository
	repo, err := git.New(repoDir)
	if err != nil {
		t.Fatalf("Failed to create Git repository: %v", err)
	}

	// Create LLM client
	llmClient, err := llm.CreateClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create LLM client: %v", err)
	}
	defer func() { _ = llmClient.Close() }()

	// Create TUI model
	model := tui.NewModel(cfg, repo, llmClient)
	if model == nil {
		t.Fatal("Expected TUI model to be created")
	}

	// Test model initialization
	cmd := model.Init()
	if cmd == nil {
		t.Error("Expected initialization command")
	}

	// Test view rendering (with dimensions set)
	model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := model.View()
	if view == "" {
		t.Error("Expected non-empty view")
	}
}

func testEndToEndWorkflow(t *testing.T, repoDir string) {
	// This test simulates a complete workflow:
	// 1. Load configuration
	// 2. Initialize Git repository
	// 3. Create and stage files
	// 4. Generate commit message
	// 5. Create commit

	// Create configuration
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

	// Initialize Git repository
	repo, err := git.New(repoDir)
	if err != nil {
		t.Fatalf("Failed to initialize Git repository: %v", err)
	}

	// Create a new file to simulate changes
	testFile := filepath.Join(repoDir, "feature.js")
	content := `function greet(name) {
    return "Hello, " + name + "!";
}

module.exports = { greet };`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stage the file
	if err := repo.StageFiles("feature.js"); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// Get staged diff
	diffs, err := repo.GetDiff(true)
	if err != nil {
		t.Fatalf("Failed to get diff: %v", err)
	}

	if len(diffs) == 0 {
		t.Fatal("Expected at least one diff")
	}

	// Create LLM client with mock provider for testing
	client := llm.NewClient()
	mockProvider := &MockProvider{
		response: &llm.CommitMessageResponse{
			Message:    "feat: add greet function for user greeting",
			Confidence: 0.9,
			TokensUsed: 25,
			Provider:   "mock",
		},
	}
	_ = client.RegisterProvider("mock", mockProvider)

	// Generate commit message
	request := &llm.CommitMessageRequest{
		Diff:        diffs[0].Content,
		Language:    cfg.LLM.Language,
		MaxTokens:   cfg.LLM.OpenAI.MaxTokens,
		Temperature: cfg.LLM.OpenAI.Temperature,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.GenerateCommitMessage(ctx, request, "mock")
	if err != nil {
		t.Fatalf("Failed to generate commit message: %v", err)
	}

	// Verify commit message
	if response.Message == "" {
		t.Error("Expected non-empty commit message")
	}

	if !strings.HasPrefix(response.Message, "feat:") {
		t.Errorf("Expected commit message to start with 'feat:', got: %s", response.Message)
	}

	// Create commit
	if err := repo.Commit(response.Message); err != nil {
		t.Fatalf("Failed to create commit: %v", err)
	}

	// Verify commit was created
	commits, err := repo.GetCommitHistory(1)
	if err != nil {
		t.Fatalf("Failed to get commit history: %v", err)
	}

	if len(commits) == 0 {
		t.Fatal("Expected at least one commit")
	}

	if commits[0].Subject != response.Message {
		t.Errorf("Expected commit subject '%s', got '%s'", response.Message, commits[0].Subject)
	}
}

// MockProvider for testing
type MockProvider struct {
	response *llm.CommitMessageResponse
	err      error
}

func (m *MockProvider) GenerateCommitMessage(ctx context.Context, request *llm.CommitMessageRequest) (*llm.CommitMessageResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *MockProvider) GetProviderName() string {
	return "mock"
}

func (m *MockProvider) Close() error {
	return nil
}
