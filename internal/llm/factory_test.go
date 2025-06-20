package llm

import (
	"path/filepath"
	"testing"

	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/logger"
)

func setupFactoryTest(t *testing.T) {
	// Initialize logger for testing
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	if err := logger.Init(logPath, "info"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
}

func TestCreateProvider(t *testing.T) {
	setupFactoryTest(t)
	defer func() { _ = logger.Close() }()

	// Test with OpenAI configuration
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "openai",
			OpenAI: config.OpenAIConfig{
				APIKey:      "test-api-key",
				Model:       "gpt-4o-mini",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			Options: map[string]string{
				"custom_option": "value",
			},
		},
	}

	provider, err := CreateProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	if provider.GetProviderName() != "openai" {
		t.Errorf("Expected provider name to be 'openai', got '%s'", provider.GetProviderName())
	}

	// Test with nil configuration
	_, err = CreateProvider(nil)
	if err == nil {
		t.Error("Expected error when configuration is nil")
	}

	// Test with unsupported provider
	cfg.LLM.Provider = "unsupported"
	_, err = CreateProvider(cfg)
	if err == nil {
		t.Error("Expected error for unsupported provider")
	}
}

func TestCreateClient(t *testing.T) {
	setupFactoryTest(t)
	defer func() { _ = logger.Close() }()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "openai",
			OpenAI: config.OpenAIConfig{
				APIKey:      "test-api-key",
				Model:       "gpt-4o-mini",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
		},
	}

	client, err := CreateClient(cfg)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	// Check if provider was registered
	providers := client.ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	if providers[0] != "openai" {
		t.Errorf("Expected provider to be 'openai', got '%s'", providers[0])
	}

	// Check if it's the default provider
	if client.GetDefaultProvider() != "openai" {
		t.Errorf("Expected default provider to be 'openai', got '%s'", client.GetDefaultProvider())
	}
}

func TestGetSupportedProviders(t *testing.T) {
	providers := GetSupportedProviders()

	if len(providers) == 0 {
		t.Error("Expected at least one supported provider")
	}

	// Check if OpenAI is supported
	found := false
	for _, provider := range providers {
		if provider == "openai" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected 'openai' to be in supported providers")
	}
}

func TestIsProviderSupported(t *testing.T) {
	// Test supported provider
	if !IsProviderSupported("openai") {
		t.Error("Expected 'openai' to be supported")
	}

	// Test unsupported provider
	if IsProviderSupported("unsupported") {
		t.Error("Expected 'unsupported' to not be supported")
	}

	// Test empty string
	if IsProviderSupported("") {
		t.Error("Expected empty string to not be supported")
	}
}

func TestCreateOpenAIProvider(t *testing.T) {
	setupFactoryTest(t)
	defer func() { _ = logger.Close() }()

	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "openai",
			OpenAI: config.OpenAIConfig{
				APIKey:      "test-api-key",
				Model:       "gpt-4",
				Temperature: 0.5,
				MaxTokens:   500,
			},
			Options: map[string]string{
				"option1": "value1",
				"option2": "value2",
			},
		},
	}

	provider, err := createOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider to be created")
	}

	// Check if configuration was applied correctly
	if provider.GetModel() != "gpt-4" {
		t.Errorf("Expected model to be 'gpt-4', got '%s'", provider.GetModel())
	}

	if provider.GetTemperature() != 0.5 {
		t.Errorf("Expected temperature to be 0.5, got %f", provider.GetTemperature())
	}

	if provider.GetMaxTokens() != 500 {
		t.Errorf("Expected max tokens to be 500, got %d", provider.GetMaxTokens())
	}
}
