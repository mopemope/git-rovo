package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration
type Config struct {
	LLM    LLMConfig    `toml:"llm"`
	Git    GitConfig    `toml:"git"`
	UI     UIConfig     `toml:"ui"`
	Logger LoggerConfig `toml:"logger"`
}

// LLMConfig represents LLM provider configuration
type LLMConfig struct {
	Provider string            `toml:"provider"`
	OpenAI   OpenAIConfig      `toml:"openai"`
	Language string            `toml:"language"`
	Options  map[string]string `toml:"options"`
}

// OpenAIConfig represents OpenAI specific configuration
type OpenAIConfig struct {
	APIKey      string  `toml:"api_key"`
	Model       string  `toml:"model"`
	Temperature float32 `toml:"temperature"`
	MaxTokens   int     `toml:"max_tokens"`
}

// GitConfig represents Git related configuration
type GitConfig struct {
	ShowUntracked bool `toml:"show_untracked"`
}

// UIConfig represents UI related configuration
type UIConfig struct {
	Theme       string            `toml:"theme"`
	KeyBindings map[string]string `toml:"key_bindings"`
}

// LoggerConfig represents logging configuration
type LoggerConfig struct {
	Level    string `toml:"level"`
	FilePath string `toml:"file_path"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		LLM: LLMConfig{
			Provider: "openai",
			OpenAI: OpenAIConfig{
				Model:       "gpt-4o-mini",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			Language: "english",
			Options:  make(map[string]string),
		},
		Git: GitConfig{
			ShowUntracked: true,
		},
		UI: UIConfig{
			Theme: "default",
			KeyBindings: map[string]string{
				"quit":         "q",
				"stage":        "s",
				"unstage":      "u",
				"commit":       "c",
				"refresh":      "r",
				"diff":         "d",
				"log":          "l",
				"generate_msg": "g",
			},
		},
		Logger: LoggerConfig{
			Level:    "info",
			FilePath: filepath.Join(homeDir, ".git-rovo.log"),
		},
	}
}

// Load loads configuration from the specified file path
func Load(configPath string) (*Config, error) {
	config := Default()

	if configPath == "" {
		// Try to find config file in common locations
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return config, nil // Return default config if home dir is not accessible
		}

		possiblePaths := []string{
			filepath.Join(homeDir, ".git-rovo.toml"),
			filepath.Join(homeDir, ".config", "git-rovo", "config.toml"),
			"config.toml",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}

		if configPath == "" {
			// No config file found, load from environment variables only
			loadFromEnvironment(config)
			return config, nil
		}
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, os.ErrNotExist // Return standard not exist error
	}

	// Parse TOML file
	if _, err := toml.DecodeFile(configPath, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables
	loadFromEnvironment(config)

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromEnvironment loads configuration from environment variables
func loadFromEnvironment(config *Config) {
	// OpenAI API Key
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.LLM.OpenAI.APIKey = apiKey
	}

	// Alternative environment variable names
	if apiKey := os.Getenv("GIT_ROVO_OPENAI_API_KEY"); apiKey != "" {
		config.LLM.OpenAI.APIKey = apiKey
	}

	// OpenAI Model
	if model := os.Getenv("GIT_ROVO_OPENAI_MODEL"); model != "" {
		config.LLM.OpenAI.Model = model
	}

	// Language
	if language := os.Getenv("GIT_ROVO_LANGUAGE"); language != "" {
		config.LLM.Language = language
	}

	// Log Level
	if logLevel := os.Getenv("GIT_ROVO_LOG_LEVEL"); logLevel != "" {
		config.Logger.Level = logLevel
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate LLM configuration
	if c.LLM.Provider == "" {
		return fmt.Errorf("llm.provider is required")
	}

	if c.LLM.Provider == "openai" {
		if c.LLM.OpenAI.APIKey == "" {
			return fmt.Errorf("openai.api_key is required when using OpenAI provider (set via config file or OPENAI_API_KEY environment variable)")
		}

		if c.LLM.OpenAI.Model == "" {
			return fmt.Errorf("openai.model is required")
		}
	}

	// Validate logger configuration
	if c.Logger.Level == "" {
		c.Logger.Level = "info"
	}

	if c.Logger.FilePath == "" {
		homeDir, _ := os.UserHomeDir()
		c.Logger.FilePath = filepath.Join(homeDir, ".git-rovo.log")
	}

	return nil
}

// Save saves the configuration to the specified file path
func (c *Config) Save(configPath string) error {
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".git-rovo.toml")
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create or open file
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close config file: %v\n", closeErr)
		}
	}()

	// Encode to TOML
	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}

	return nil
}

// Default creates a default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		LLM: LLMConfig{
			Provider: "openai",
			OpenAI: OpenAIConfig{
				Model:       "gpt-4o-mini",
				Temperature: 0.7,
				MaxTokens:   1000,
			},
			Language: "english",
			Options:  make(map[string]string),
		},
		Git: GitConfig{
			ShowUntracked: true,
		},
		UI: UIConfig{
			Theme:       "default",
			KeyBindings: make(map[string]string),
		},
		Logger: LoggerConfig{
			Level:    "info",
			FilePath: filepath.Join(homeDir, ".local", "share", "git-rovo", "git-rovo.log"),
		},
	}
}

// Save saves the configuration to a file
func Save(cfg *Config, path string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close config file: %v\n", closeErr)
		}
	}()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
