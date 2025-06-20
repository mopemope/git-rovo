package llm

import (
	"context"
	"strings"
	"testing"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	name     string
	response *CommitMessageResponse
	err      error
	closed   bool
}

func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name: name,
		response: &CommitMessageResponse{
			Message:    "feat: add new feature",
			Confidence: 0.9,
			TokensUsed: 50,
			Provider:   name,
		},
	}
}

func (m *MockProvider) GenerateCommitMessage(ctx context.Context, request *CommitMessageRequest) (*CommitMessageResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Simulate processing the diff
	response := *m.response
	response.Provider = m.name

	return &response, nil
}

func (m *MockProvider) GetProviderName() string {
	return m.name
}

func (m *MockProvider) Close() error {
	m.closed = true
	return nil
}

func TestNewClient(t *testing.T) {
	client := NewClient()

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if len(client.providers) != 0 {
		t.Error("Expected no providers initially")
	}

	if client.defaultProvider != "" {
		t.Error("Expected no default provider initially")
	}
}

func TestRegisterProvider(t *testing.T) {
	client := NewClient()
	provider := NewMockProvider("test")

	// Test successful registration
	err := client.RegisterProvider("test", provider)
	if err != nil {
		t.Fatalf("Failed to register provider: %v", err)
	}

	// Check if provider was registered
	if len(client.providers) != 1 {
		t.Error("Expected one provider to be registered")
	}

	// Check if it became the default provider
	if client.defaultProvider != "test" {
		t.Error("Expected 'test' to be the default provider")
	}

	// Test registration with empty name
	err = client.RegisterProvider("", provider)
	if err == nil {
		t.Error("Expected error when registering provider with empty name")
	}

	// Test registration with nil provider
	err = client.RegisterProvider("nil", nil)
	if err == nil {
		t.Error("Expected error when registering nil provider")
	}
}

func TestCleanMarkdownFromCommitMessage(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No markdown",
			input:    "feat: add new feature",
			expected: "feat: add new feature",
		},
		{
			name:     "Bold formatting",
			input:    "feat: add **new** feature",
			expected: "feat: add new feature",
		},
		{
			name:     "Italic formatting",
			input:    "feat: add *new* feature",
			expected: "feat: add new feature",
		},
		{
			name:     "Code formatting",
			input:    "feat: add `new` feature",
			expected: "feat: add new feature",
		},
		{
			name:     "Header formatting",
			input:    "# feat: add new feature",
			expected: "feat: add new feature",
		},
		{
			name:     "Multiple headers",
			input:    "## feat: add new feature\n### with details",
			expected: "feat: add new feature\nwith details",
		},
		{
			name:     "Mixed formatting",
			input:    "**feat**: add `new` *feature* with __bold__ text",
			expected: "feat: add new feature with bold text",
		},
		{
			name:     "Strikethrough",
			input:    "feat: add ~~old~~ new feature",
			expected: "feat: add  new feature",
		},
		{
			name:     "Complex markdown",
			input:    "# **feat**: add `new` feature\n\n- *Implement* core functionality\n- **Update** documentation",
			expected: "feat: add new feature\n- Implement core functionality\n- Update documentation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := CleanMarkdownFromCommitMessage(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestSetDefaultProvider(t *testing.T) {
	client := NewClient()
	provider1 := NewMockProvider("provider1")
	provider2 := NewMockProvider("provider2")

	// Register providers
	_ = client.RegisterProvider("provider1", provider1)
	_ = client.RegisterProvider("provider2", provider2)

	// Test setting valid default provider
	err := client.SetDefaultProvider("provider2")
	if err != nil {
		t.Fatalf("Failed to set default provider: %v", err)
	}

	if client.defaultProvider != "provider2" {
		t.Error("Expected 'provider2' to be the default provider")
	}

	// Test setting invalid default provider
	err = client.SetDefaultProvider("nonexistent")
	if err == nil {
		t.Error("Expected error when setting nonexistent provider as default")
	}
}

func TestGetProvider(t *testing.T) {
	client := NewClient()
	provider := NewMockProvider("test")
	_ = client.RegisterProvider("test", provider)

	// Test getting existing provider
	retrieved, err := client.GetProvider("test")
	if err != nil {
		t.Fatalf("Failed to get provider: %v", err)
	}

	if retrieved != provider {
		t.Error("Expected to get the same provider instance")
	}

	// Test getting default provider (empty name)
	retrieved, err = client.GetProvider("")
	if err != nil {
		t.Fatalf("Failed to get default provider: %v", err)
	}

	if retrieved != provider {
		t.Error("Expected to get the default provider")
	}

	// Test getting nonexistent provider
	_, err = client.GetProvider("nonexistent")
	if err == nil {
		t.Error("Expected error when getting nonexistent provider")
	}
}

func TestGenerateCommitMessage(t *testing.T) {
	client := NewClient()
	provider := NewMockProvider("test")
	_ = client.RegisterProvider("test", provider)

	request := &CommitMessageRequest{
		Diff:        "diff --git a/file.txt b/file.txt\n+new line",
		Language:    "english",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	// Test generating commit message
	response, err := client.GenerateCommitMessage(context.Background(), request, "test")
	if err != nil {
		t.Fatalf("Failed to generate commit message: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if response.Message == "" {
		t.Error("Expected commit message to be non-empty")
	}

	if response.Provider != "test" {
		t.Errorf("Expected provider to be 'test', got '%s'", response.Provider)
	}

	// Test with default provider (empty name)
	response, err = client.GenerateCommitMessage(context.Background(), request, "")
	if err != nil {
		t.Fatalf("Failed to generate commit message with default provider: %v", err)
	}

	if response.Provider != "test" {
		t.Errorf("Expected provider to be 'test', got '%s'", response.Provider)
	}
}

func TestListProviders(t *testing.T) {
	client := NewClient()

	// Initially should be empty
	providers := client.ListProviders()
	if len(providers) != 0 {
		t.Error("Expected no providers initially")
	}

	// Add providers
	_ = client.RegisterProvider("provider1", NewMockProvider("provider1"))
	_ = client.RegisterProvider("provider2", NewMockProvider("provider2"))

	providers = client.ListProviders()
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Check if both providers are in the list
	providerMap := make(map[string]bool)
	for _, name := range providers {
		providerMap[name] = true
	}

	if !providerMap["provider1"] || !providerMap["provider2"] {
		t.Error("Expected both providers to be in the list")
	}
}

func TestClose(t *testing.T) {
	client := NewClient()
	provider1 := NewMockProvider("provider1")
	provider2 := NewMockProvider("provider2")

	_ = client.RegisterProvider("provider1", provider1)
	_ = client.RegisterProvider("provider2", provider2)

	// Close client
	err := client.Close()
	if err != nil {
		t.Fatalf("Failed to close client: %v", err)
	}

	// Check if providers were closed
	if !provider1.closed {
		t.Error("Expected provider1 to be closed")
	}

	if !provider2.closed {
		t.Error("Expected provider2 to be closed")
	}
}

func TestValidateRequest(t *testing.T) {
	// Test nil request
	err := ValidateRequest(nil)
	if err == nil {
		t.Error("Expected error for nil request")
	}

	// Test empty diff
	request := &CommitMessageRequest{}
	err = ValidateRequest(request)
	if err == nil {
		t.Error("Expected error for empty diff")
	}

	// Test valid request
	request = &CommitMessageRequest{
		Diff: "diff --git a/file.txt b/file.txt\n+new line",
	}
	err = ValidateRequest(request)
	if err != nil {
		t.Errorf("Expected no error for valid request, got: %v", err)
	}

	// Check if defaults were set
	if request.Language != "english" {
		t.Errorf("Expected default language to be 'english', got '%s'", request.Language)
	}

	if request.MaxTokens != 1000 {
		t.Errorf("Expected default max tokens to be 1000, got %d", request.MaxTokens)
	}

	if request.Temperature != 0.7 {
		t.Errorf("Expected default temperature to be 0.7, got %f", request.Temperature)
	}
}

func TestBuildPrompt(t *testing.T) {
	request := &CommitMessageRequest{
		Diff:              "diff --git a/file.txt b/file.txt\n+new line",
		Language:          "english",
		AdditionalContext: "This is a test change",
	}

	prompt := BuildPrompt(request)

	if prompt == "" {
		t.Error("Expected prompt to be non-empty")
	}

	// Check if prompt contains expected elements
	expectedElements := []string{
		"Conventional Commits",
		"english",
		"diff --git a/file.txt b/file.txt",
		"This is a test change",
		"feat, fix, docs",
	}

	for _, element := range expectedElements {
		if !strings.Contains(prompt, element) {
			t.Errorf("Expected prompt to contain '%s'", element)
		}
	}
}
