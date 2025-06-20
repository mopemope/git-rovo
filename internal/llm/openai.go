package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/mopemope/git-rovo/internal/logger"
	"github.com/sashabaranov/go-openai"
)

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	client      *openai.Client
	model       string
	temperature float32
	maxTokens   int
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(config ProviderConfig) (*OpenAIProvider, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	if config.Model == "" {
		config.Model = "gpt-4o-mini" // Default model
	}

	if config.Temperature <= 0 {
		config.Temperature = 0.7 // Default temperature
	}

	if config.MaxTokens <= 0 {
		config.MaxTokens = 1000 // Default max tokens
	}

	// Create OpenAI client
	var client *openai.Client
	if config.BaseURL != "" {
		clientConfig := openai.DefaultConfig(config.APIKey)
		clientConfig.BaseURL = config.BaseURL
		client = openai.NewClientWithConfig(clientConfig)
	} else {
		client = openai.NewClient(config.APIKey)
	}

	return &OpenAIProvider{
		client:      client,
		model:       config.Model,
		temperature: config.Temperature,
		maxTokens:   config.MaxTokens,
	}, nil
}

// GenerateCommitMessage generates a commit message using OpenAI
func (p *OpenAIProvider) GenerateCommitMessage(ctx context.Context, request *CommitMessageRequest) (*CommitMessageResponse, error) {
	// Validate request
	if err := ValidateRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Build prompt
	prompt := BuildPrompt(request)

	// Use request-specific parameters if provided, otherwise use provider defaults
	temperature := p.temperature
	if request.Temperature > 0 {
		temperature = request.Temperature
	}

	maxTokens := p.maxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}

	// Create chat completion request
	chatRequest := openai.ChatCompletionRequest{
		Model:       p.model,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are an expert software developer who writes excellent commit messages following Conventional Commits specification. Always respond with plain text only, never use markdown formatting.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	// Make API request
	response, err := p.client.CreateChatCompletion(ctx, chatRequest)
	if err != nil {
		logger.LogLLMRequest("openai", p.model, prompt, "", false, err)
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	// Extract response
	if len(response.Choices) == 0 {
		err := fmt.Errorf("no choices returned from OpenAI")
		logger.LogLLMRequest("openai", p.model, prompt, "", false, err)
		return nil, err
	}

	commitMessage := strings.TrimSpace(response.Choices[0].Message.Content)
	if commitMessage == "" {
		err := fmt.Errorf("empty commit message returned from OpenAI")
		logger.LogLLMRequest("openai", p.model, prompt, commitMessage, false, err)
		return nil, err
	}

	// Clean any markdown formatting from the commit message
	commitMessage = CleanMarkdownFromCommitMessage(commitMessage)

	// Calculate confidence based on finish reason and response quality
	confidence := p.calculateConfidence(response.Choices[0])

	// Create response
	result := &CommitMessageResponse{
		Message:    commitMessage,
		Confidence: confidence,
		TokensUsed: response.Usage.TotalTokens,
		Provider:   "openai",
	}

	// Log successful request
	logger.LogLLMRequest("openai", p.model, prompt, commitMessage, true, nil)

	return result, nil
}

// calculateConfidence calculates confidence based on the OpenAI response
func (p *OpenAIProvider) calculateConfidence(choice openai.ChatCompletionChoice) float32 {
	var confidence float32

	// Adjust based on finish reason
	switch choice.FinishReason {
	case "stop":
		confidence = 0.9 // Normal completion
	case "length":
		confidence = 0.7 // Truncated due to length
	case "content_filter":
		confidence = 0.5 // Content filtered
	default:
		confidence = 0.6 // Unknown reason
	}

	// Additional checks for commit message quality
	message := strings.TrimSpace(choice.Message.Content)

	// Check if it follows conventional commits format
	if p.isConventionalCommit(message) {
		confidence += 0.1
	}

	// Check length (good commit messages are typically 50-72 chars for first line)
	lines := strings.Split(message, "\n")
	if len(lines) > 0 {
		firstLine := lines[0]
		if len(firstLine) >= 10 && len(firstLine) <= 72 {
			confidence += 0.05
		}
	}

	// Ensure confidence is within bounds
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// isConventionalCommit checks if the message follows conventional commits format
func (p *OpenAIProvider) isConventionalCommit(message string) bool {
	// Basic check for conventional commits pattern: type(scope): description
	conventionalTypes := []string{
		"feat", "fix", "docs", "style", "refactor", "test", "chore",
		"perf", "ci", "build", "revert", "merge", "wip",
	}

	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return false
	}

	firstLine := strings.ToLower(strings.TrimSpace(lines[0]))

	for _, commitType := range conventionalTypes {
		if strings.HasPrefix(firstLine, commitType+":") ||
			strings.Contains(firstLine, commitType+"(") {
			return true
		}
	}

	return false
}

// GetProviderName returns the provider name
func (p *OpenAIProvider) GetProviderName() string {
	return "openai"
}

// Close closes any resources (OpenAI client doesn't need explicit closing)
func (p *OpenAIProvider) Close() error {
	// OpenAI client doesn't require explicit cleanup
	return nil
}

// GetModel returns the current model being used
func (p *OpenAIProvider) GetModel() string {
	return p.model
}

// SetModel sets the model to use
func (p *OpenAIProvider) SetModel(model string) {
	p.model = model
}

// GetTemperature returns the current temperature setting
func (p *OpenAIProvider) GetTemperature() float32 {
	return p.temperature
}

// SetTemperature sets the temperature for generation
func (p *OpenAIProvider) SetTemperature(temperature float32) {
	p.temperature = temperature
}

// GetMaxTokens returns the current max tokens setting
func (p *OpenAIProvider) GetMaxTokens() int {
	return p.maxTokens
}

// SetMaxTokens sets the maximum tokens for generation
func (p *OpenAIProvider) SetMaxTokens(maxTokens int) {
	p.maxTokens = maxTokens
}

// ListAvailableModels returns a list of available OpenAI models for chat completion
func (p *OpenAIProvider) ListAvailableModels() []string {
	return []string{
		"gpt-4o",
		"gpt-4o-mini",
		"gpt-4-turbo",
		"gpt-4",
		"gpt-3.5-turbo",
	}
}

// ValidateModel checks if the given model is supported
func (p *OpenAIProvider) ValidateModel(model string) bool {
	availableModels := p.ListAvailableModels()
	for _, availableModel := range availableModels {
		if model == availableModel {
			return true
		}
	}
	return false
}
