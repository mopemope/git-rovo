package llm

import (
	"context"
	"fmt"
	"strings"
)

// Provider represents an LLM provider interface
type Provider interface {
	// GenerateCommitMessage generates a commit message based on the diff
	GenerateCommitMessage(ctx context.Context, request *CommitMessageRequest) (*CommitMessageResponse, error)

	// GetProviderName returns the name of the provider
	GetProviderName() string

	// Close closes any resources used by the provider
	Close() error
}

// CommitMessageRequest represents a request to generate a commit message
type CommitMessageRequest struct {
	// Diff contains the git diff content
	Diff string

	// Language specifies the language for the commit message (e.g., "english", "japanese")
	Language string

	// AdditionalContext provides extra context for the commit message generation
	AdditionalContext string

	// MaxTokens specifies the maximum number of tokens in the response
	MaxTokens int

	// Temperature controls the randomness of the response (0.0 to 1.0)
	Temperature float32
}

// CommitMessageResponse represents the response from commit message generation
type CommitMessageResponse struct {
	// Message contains the generated commit message
	Message string

	// Confidence represents the confidence level of the generated message (0.0 to 1.0)
	Confidence float32

	// TokensUsed represents the number of tokens used in the request
	TokensUsed int

	// Provider contains the name of the provider that generated the message
	Provider string
}

// ProviderConfig represents configuration for LLM providers
type ProviderConfig struct {
	// Provider name (e.g., "openai", "anthropic", "gemini")
	Name string

	// API key for authentication
	APIKey string

	// Model name to use
	Model string

	// Base URL for API requests (optional, for custom endpoints)
	BaseURL string

	// Default temperature
	Temperature float32

	// Default max tokens
	MaxTokens int

	// Additional options specific to the provider
	Options map[string]interface{}
}

// Client manages LLM providers and provides a unified interface
type Client struct {
	providers       map[string]Provider
	defaultProvider string
}

// NewClient creates a new LLM client
func NewClient() *Client {
	return &Client{
		providers: make(map[string]Provider),
	}
}

// RegisterProvider registers a new LLM provider
func (c *Client) RegisterProvider(name string, provider Provider) error {
	if name == "" {
		return fmt.Errorf("provider name cannot be empty")
	}

	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}

	c.providers[name] = provider

	// Set as default if it's the first provider
	if c.defaultProvider == "" {
		c.defaultProvider = name
	}

	return nil
}

// SetDefaultProvider sets the default provider
func (c *Client) SetDefaultProvider(name string) error {
	if _, exists := c.providers[name]; !exists {
		return fmt.Errorf("provider %s not found", name)
	}

	c.defaultProvider = name
	return nil
}

// GetProvider returns a specific provider
func (c *Client) GetProvider(name string) (Provider, error) {
	if name == "" {
		name = c.defaultProvider
	}

	provider, exists := c.providers[name]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", name)
	}

	return provider, nil
}

// GenerateCommitMessage generates a commit message using the specified or default provider
func (c *Client) GenerateCommitMessage(ctx context.Context, request *CommitMessageRequest, providerName string) (*CommitMessageResponse, error) {
	provider, err := c.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return provider.GenerateCommitMessage(ctx, request)
}

// ListProviders returns a list of registered provider names
func (c *Client) ListProviders() []string {
	var names []string
	for name := range c.providers {
		names = append(names, name)
	}
	return names
}

// GetDefaultProvider returns the name of the default provider
func (c *Client) GetDefaultProvider() string {
	return c.defaultProvider
}

// Close closes all registered providers
func (c *Client) Close() error {
	var lastErr error
	for _, provider := range c.providers {
		if err := provider.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ValidateRequest validates a commit message request
func ValidateRequest(request *CommitMessageRequest) error {
	if request == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if request.Diff == "" {
		return fmt.Errorf("diff cannot be empty")
	}

	if request.Language == "" {
		request.Language = "english" // Default to English
	}

	if request.MaxTokens <= 0 {
		request.MaxTokens = 1000 // Default max tokens
	}

	if request.Temperature <= 0 || request.Temperature > 1 {
		request.Temperature = 0.7 // Default temperature
	}

	return nil
}

// BuildPrompt builds a prompt for commit message generation
func BuildPrompt(request *CommitMessageRequest) string {
	prompt := fmt.Sprintf(`You are an expert software developer.
Generate a concise and descriptive commit message following the Conventional Commits specification.
And then one empty line. Then detailed description of all changes.

Language: %s
Format: <type>(<scope>): <description>

<detailed description of all changes>

Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build, revert

Rules:
1. Use lowercase for type and description
2. Keep the first line under 50 characters
3. Be specific and descriptive
4. Focus on what changed and why
5. Use imperative mood (e.g., "add" not "added")
6. Do NOT use markdown formatting (no asterisks, underscores, backticks, etc.)
7. Use plain text only
8. Must insert a blank line after the first line before detailing the changes

<EXAMPLE>
feat: initial project setup and core feature implementation

- Update .gitignore to ignore logs, config, and application artifacts
- Add Makefile with build, test, coverage, lint, install, release, and dev-setup targets
- Expand README with installation, usage, configuration, key bindings, and contribution guidelines
- Initialize go.mod and go.sum with required dependencies
- Implement internal/config for TOML-based configuration loading, validation, and saving
- Create internal/git wrapper for Git operations: status, diff parsing, staging, commit, history,
branch detectionr
</EXAMPLE>

Git diff:
%s`, request.Language, request.Diff)

	if request.AdditionalContext != "" {
		prompt += fmt.Sprintf("\n\nAdditional context:\n%s", request.AdditionalContext)
	}

	prompt += "\n\nGenerate only the commit message in plain text format, no explanations, no markdown formatting:"

	return prompt
}

// CleanMarkdownFromCommitMessage removes markdown formatting from commit message
func CleanMarkdownFromCommitMessage(message string) string {
	// Remove common markdown formatting
	cleaned := message

	// Remove strikethrough formatting (~~text~~)
	for strings.Contains(cleaned, "~~") {
		start := strings.Index(cleaned, "~~")
		if start == -1 {
			break
		}
		end := strings.Index(cleaned[start+2:], "~~")
		if end == -1 {
			// Remove remaining ~~
			cleaned = strings.ReplaceAll(cleaned, "~~", "")
			break
		}
		// Remove the strikethrough text entirely
		cleaned = cleaned[:start] + cleaned[start+end+4:]
	}

	// Remove bold formatting (**text** or __text__)
	cleaned = strings.ReplaceAll(cleaned, "**", "")
	cleaned = strings.ReplaceAll(cleaned, "__", "")

	// Remove italic formatting (*text* or _text_)
	// Be careful not to remove single underscores that might be part of variable names
	cleaned = strings.ReplaceAll(cleaned, "*", "")

	// Remove backticks for code formatting
	cleaned = strings.ReplaceAll(cleaned, "`", "")

	// Remove markdown headers (# ## ###)
	lines := strings.Split(cleaned, "\n")
	var cleanedLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove leading # characters
		for strings.HasPrefix(line, "#") {
			line = strings.TrimSpace(line[1:])
		}
		if line != "" {
			cleanedLines = append(cleanedLines, line)
		}
	}

	if len(cleanedLines) > 0 {
		cleaned = strings.Join(cleanedLines, "\n")
	}

	// Remove any remaining markdown-like formatting
	cleaned = strings.ReplaceAll(cleaned, "---", "") // horizontal rule
	cleaned = strings.ReplaceAll(cleaned, "***", "") // bold+italic

	// Clean up extra whitespace
	cleaned = strings.TrimSpace(cleaned)

	return cleaned
}
