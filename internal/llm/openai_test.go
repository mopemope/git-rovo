package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopemope/git-rovo/internal/logger"
)

func setupOpenAITest(t *testing.T) {
	// Initialize logger for testing
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	if err := logger.Init(logPath, "info"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
}

func TestNewOpenAIProvider(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	// Test with valid config
	config := ProviderConfig{
		APIKey:      "test-api-key",
		Model:       "gpt-4o-mini",
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.GetProviderName() != "openai" {
		t.Errorf("Expected provider name to be 'openai', got '%s'", provider.GetProviderName())
	}

	if provider.GetModel() != "gpt-4o-mini" {
		t.Errorf("Expected model to be 'gpt-4o-mini', got '%s'", provider.GetModel())
	}

	// Test with empty API key
	config.APIKey = ""
	_, err = NewOpenAIProvider(config)
	if err == nil {
		t.Error("Expected error when API key is empty")
	}

	// Test with defaults
	config.APIKey = "test-api-key"
	config.Model = ""
	config.Temperature = 0
	config.MaxTokens = 0

	provider, err = NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider with defaults: %v", err)
	}

	if provider.GetModel() != "gpt-4o-mini" {
		t.Errorf("Expected default model to be 'gpt-4o-mini', got '%s'", provider.GetModel())
	}

	if provider.GetTemperature() != 0.7 {
		t.Errorf("Expected default temperature to be 0.7, got %f", provider.GetTemperature())
	}

	if provider.GetMaxTokens() != 1000 {
		t.Errorf("Expected default max tokens to be 1000, got %d", provider.GetMaxTokens())
	}
}

func TestOpenAIProviderSetters(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey: "test-api-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Test SetModel
	provider.SetModel("gpt-4")
	if provider.GetModel() != "gpt-4" {
		t.Errorf("Expected model to be 'gpt-4', got '%s'", provider.GetModel())
	}

	// Test SetTemperature
	provider.SetTemperature(0.5)
	if provider.GetTemperature() != 0.5 {
		t.Errorf("Expected temperature to be 0.5, got %f", provider.GetTemperature())
	}

	// Test SetMaxTokens
	provider.SetMaxTokens(500)
	if provider.GetMaxTokens() != 500 {
		t.Errorf("Expected max tokens to be 500, got %d", provider.GetMaxTokens())
	}
}

func TestListAvailableModels(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey: "test-api-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	models := provider.ListAvailableModels()
	if len(models) == 0 {
		t.Error("Expected at least one available model")
	}

	// Check if common models are included
	expectedModels := []string{"gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"}
	modelMap := make(map[string]bool)
	for _, model := range models {
		modelMap[model] = true
	}

	for _, expected := range expectedModels {
		if !modelMap[expected] {
			t.Errorf("Expected model '%s' to be in available models", expected)
		}
	}
}

func TestValidateModel(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey: "test-api-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Test valid models
	validModels := []string{"gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"}
	for _, model := range validModels {
		if !provider.ValidateModel(model) {
			t.Errorf("Expected model '%s' to be valid", model)
		}
	}

	// Test invalid model
	if provider.ValidateModel("invalid-model") {
		t.Error("Expected 'invalid-model' to be invalid")
	}
}

func TestIsConventionalCommit(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey: "test-api-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	testCases := []struct {
		message  string
		expected bool
	}{
		{"feat: add new feature", true},
		{"fix: resolve bug in authentication", true},
		{"docs: update README", true},
		{"feat(auth): add login functionality", true},
		{"refactor(ui): improve component structure", true},
		{"Add new feature", false},
		{"Fixed a bug", false},
		{"", false},
		{"random text", false},
	}

	for _, tc := range testCases {
		result := provider.isConventionalCommit(tc.message)
		if result != tc.expected {
			t.Errorf("isConventionalCommit(%q) = %v, expected %v", tc.message, result, tc.expected)
		}
	}
}

func TestCalculateConfidence(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey: "test-api-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// We can't easily test this without importing openai types,
	// but we can test the isConventionalCommit helper function
	// which is part of the confidence calculation

	// Test conventional commit detection
	conventionalMessage := "feat: add new authentication system"
	if !provider.isConventionalCommit(conventionalMessage) {
		t.Error("Expected conventional commit to be detected")
	}

	nonConventionalMessage := "Added some new stuff"
	if provider.isConventionalCommit(nonConventionalMessage) {
		t.Error("Expected non-conventional commit to not be detected")
	}
}

// Integration test - only runs if OPENAI_API_KEY is set
func TestGenerateCommitMessageIntegration(t *testing.T) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("Skipping integration test: OPENAI_API_KEY not set")
	}

	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey:      apiKey,
		Model:       "gpt-3.5-turbo", // Use cheaper model for testing
		Temperature: 0.7,
		MaxTokens:   100,
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	request := &CommitMessageRequest{
		Diff: `diff --git a/README.md b/README.md
index 1234567..abcdefg 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
 # My Project
 
 This is a sample project.
+Added new documentation section.`,
		Language:    "english",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	ctx := context.Background()
	response, err := provider.GenerateCommitMessage(ctx, request)
	if err != nil {
		t.Fatalf("Failed to generate commit message: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if response.Message == "" {
		t.Error("Expected commit message to be non-empty")
	}

	if response.Provider != "openai" {
		t.Errorf("Expected provider to be 'openai', got '%s'", response.Provider)
	}

	if response.TokensUsed <= 0 {
		t.Error("Expected tokens used to be greater than 0")
	}

	if response.Confidence <= 0 || response.Confidence > 1 {
		t.Errorf("Expected confidence to be between 0 and 1, got %f", response.Confidence)
	}

	// Check if the message looks like a commit message
	message := strings.ToLower(response.Message)
	if !strings.Contains(message, "doc") && !strings.Contains(message, "readme") && !strings.Contains(message, "add") {
		t.Logf("Generated message: %s", response.Message)
		t.Error("Expected commit message to be related to documentation changes")
	}

	t.Logf("Generated commit message: %s", response.Message)
	t.Logf("Confidence: %f", response.Confidence)
	t.Logf("Tokens used: %d", response.TokensUsed)
}

func TestOpenAIClose(t *testing.T) {
	setupOpenAITest(t)
	defer func() { _ = logger.Close() }()

	config := ProviderConfig{
		APIKey: "test-api-key",
	}

	provider, err := NewOpenAIProvider(config)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Close should not return an error
	err = provider.Close()
	if err != nil {
		t.Errorf("Expected Close() to not return an error, got: %v", err)
	}
}
