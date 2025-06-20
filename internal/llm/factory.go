package llm

import (
	"fmt"

	"github.com/mopemope/git-rovo/internal/config"
)

// CreateProvider creates a provider based on the configuration
func CreateProvider(cfg *config.Config) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	switch cfg.LLM.Provider {
	case "openai":
		return createOpenAIProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.LLM.Provider)
	}
}

// createOpenAIProvider creates an OpenAI provider from configuration
func createOpenAIProvider(cfg *config.Config) (*OpenAIProvider, error) {
	providerConfig := ProviderConfig{
		Name:        "openai",
		APIKey:      cfg.LLM.OpenAI.APIKey,
		Model:       cfg.LLM.OpenAI.Model,
		Temperature: cfg.LLM.OpenAI.Temperature,
		MaxTokens:   cfg.LLM.OpenAI.MaxTokens,
		Options:     make(map[string]interface{}),
	}

	// Copy additional options
	for key, value := range cfg.LLM.Options {
		providerConfig.Options[key] = value
	}

	return NewOpenAIProvider(providerConfig)
}

// CreateClient creates a client with providers based on configuration
func CreateClient(cfg *config.Config) (*Client, error) {
	client := NewClient()

	// Create and register the configured provider
	provider, err := CreateProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	if err := client.RegisterProvider(cfg.LLM.Provider, provider); err != nil {
		return nil, fmt.Errorf("failed to register provider: %w", err)
	}

	// Set the default provider
	if err := client.SetDefaultProvider(cfg.LLM.Provider); err != nil {
		return nil, fmt.Errorf("failed to set default provider: %w", err)
	}

	return client, nil
}

// GetSupportedProviders returns a list of supported provider names
func GetSupportedProviders() []string {
	return []string{
		"openai",
		// Add more providers here as they are implemented
		// "anthropic",
		// "gemini",
	}
}

// IsProviderSupported checks if a provider is supported
func IsProviderSupported(providerName string) bool {
	supported := GetSupportedProviders()
	for _, name := range supported {
		if name == providerName {
			return true
		}
	}
	return false
}
