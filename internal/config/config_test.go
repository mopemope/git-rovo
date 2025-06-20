package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.LLM.Provider != "openai" {
		t.Errorf("Expected default LLM provider to be 'openai', got %s", config.LLM.Provider)
	}

	if config.LLM.Language != "english" {
		t.Errorf("Expected default language to be 'english', got %s", config.LLM.Language)
	}

	if config.LLM.OpenAI.Model != "gpt-4o-mini" {
		t.Errorf("Expected default OpenAI model to be 'gpt-4o-mini', got %s", config.LLM.OpenAI.Model)
	}

	if config.Logger.Level != "info" {
		t.Errorf("Expected default log level to be 'info', got %s", config.Logger.Level)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	_, err := Load("non-existent-config.toml")
	if err == nil {
		t.Error("Expected error when loading non-existent config file")
	}
}

func TestLoadDefaultConfig(t *testing.T) {
	config, err := Load("")
	if err != nil {
		t.Errorf("Expected no error when loading default config, got %v", err)
	}

	if config == nil {
		t.Error("Expected config to be non-nil")
	}
}

func TestConfigValidation(t *testing.T) {
	config := DefaultConfig()

	// Set API key for valid config test
	config.LLM.OpenAI.APIKey = "test-api-key"

	// Test valid config
	if err := config.Validate(); err != nil {
		t.Errorf("Expected valid config to pass validation, got error: %v", err)
	}

	// Test invalid provider
	config.LLM.Provider = ""
	if err := config.Validate(); err == nil {
		t.Error("Expected error when LLM provider is empty")
	}

	// Reset provider and test missing API key
	config.LLM.Provider = "openai"
	config.LLM.OpenAI.APIKey = ""
	if err := config.Validate(); err == nil {
		t.Error("Expected error when OpenAI API key is missing")
	}

	// Reset and test missing OpenAI API key
	config = DefaultConfig()
	config.LLM.OpenAI.APIKey = ""
	_ = os.Unsetenv("OPENAI_API_KEY")
	if err := config.Validate(); err == nil {
		t.Error("Expected error when OpenAI API key is missing")
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	// Save original environment variables
	originalAPIKey := os.Getenv("OPENAI_API_KEY")
	originalGitRovoAPIKey := os.Getenv("GIT_ROVO_OPENAI_API_KEY")
	originalModel := os.Getenv("GIT_ROVO_OPENAI_MODEL")
	originalLanguage := os.Getenv("GIT_ROVO_LANGUAGE")
	originalLogLevel := os.Getenv("GIT_ROVO_LOG_LEVEL")

	// Clean up function
	defer func() {
		if originalAPIKey != "" {
			_ = os.Setenv("OPENAI_API_KEY", originalAPIKey)
		} else {
			_ = os.Unsetenv("OPENAI_API_KEY")
		}
		if originalGitRovoAPIKey != "" {
			_ = os.Setenv("GIT_ROVO_OPENAI_API_KEY", originalGitRovoAPIKey)
		} else {
			_ = os.Unsetenv("GIT_ROVO_OPENAI_API_KEY")
		}
		if originalModel != "" {
			_ = os.Setenv("GIT_ROVO_OPENAI_MODEL", originalModel)
		} else {
			_ = os.Unsetenv("GIT_ROVO_OPENAI_MODEL")
		}
		if originalLanguage != "" {
			_ = os.Setenv("GIT_ROVO_LANGUAGE", originalLanguage)
		} else {
			_ = os.Unsetenv("GIT_ROVO_LANGUAGE")
		}
		if originalLogLevel != "" {
			_ = os.Setenv("GIT_ROVO_LOG_LEVEL", originalLogLevel)
		} else {
			_ = os.Unsetenv("GIT_ROVO_LOG_LEVEL")
		}
	}()

	// Test OPENAI_API_KEY
	_ = os.Setenv("OPENAI_API_KEY", "test-openai-key")
	config, err := Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.LLM.OpenAI.APIKey != "test-openai-key" {
		t.Errorf("Expected API key 'test-openai-key', got '%s'", config.LLM.OpenAI.APIKey)
	}

	// Test GIT_ROVO_OPENAI_API_KEY overrides OPENAI_API_KEY
	_ = os.Setenv("GIT_ROVO_OPENAI_API_KEY", "test-git-rovo-key")
	config, err = Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.LLM.OpenAI.APIKey != "test-git-rovo-key" {
		t.Errorf("Expected API key 'test-git-rovo-key', got '%s'", config.LLM.OpenAI.APIKey)
	}

	// Test other environment variables
	_ = os.Setenv("GIT_ROVO_OPENAI_MODEL", "gpt-4o")
	_ = os.Setenv("GIT_ROVO_LANGUAGE", "japanese")
	_ = os.Setenv("GIT_ROVO_LOG_LEVEL", "debug")

	config, err = Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if config.LLM.OpenAI.Model != "gpt-4o" {
		t.Errorf("Expected model 'gpt-4o', got '%s'", config.LLM.OpenAI.Model)
	}

	if config.LLM.Language != "japanese" {
		t.Errorf("Expected language 'japanese', got '%s'", config.LLM.Language)
	}

	if config.Logger.Level != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", config.Logger.Level)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.toml")

	// Create test config
	config := DefaultConfig()
	config.LLM.OpenAI.APIKey = "test-api-key"
	config.LLM.Language = "japanese"

	// Save config
	if err := config.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedConfig, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify loaded config
	if loadedConfig.LLM.OpenAI.APIKey != "test-api-key" {
		t.Errorf("Expected API key to be 'test-api-key', got %s", loadedConfig.LLM.OpenAI.APIKey)
	}

	if loadedConfig.LLM.Language != "japanese" {
		t.Errorf("Expected language to be 'japanese', got %s", loadedConfig.LLM.Language)
	}
}
