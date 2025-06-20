package logger

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger, err := New(logPath, "info")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	if logger.GetFilePath() != logPath {
		t.Errorf("Expected log path %s, got %s", logPath, logger.GetFilePath())
	}

	// Test that file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestLogLevels(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	testCases := []struct {
		level    string
		expected string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"warning", "warn"},
		{"error", "error"},
		{"invalid", "info"}, // Should default to info
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			logger, err := New(logPath, tc.level)
			if err != nil {
				t.Fatalf("Failed to create logger: %v", err)
			}
			defer func() { _ = logger.Close() }()

			// Log a message and verify it was written
			logger.Info("test message")
		})
	}
}

func TestLogGitOperation(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger, err := New(logPath, "info")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test successful operation
	logger.LogGitOperation("add", []string{"file.txt"}, "/tmp", true, "added file.txt", nil)

	// Test failed operation
	testErr := fmt.Errorf("file not found")
	logger.LogGitOperation("add", []string{"missing.txt"}, "/tmp", false, "", testErr)

	// Verify log file contains the entries
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "Git operation completed") {
		t.Error("Expected successful git operation log entry")
	}
	if !strings.Contains(logContent, "Git operation failed") {
		t.Error("Expected failed git operation log entry")
	}
	if !strings.Contains(logContent, "file not found") {
		t.Error("Expected error message in log")
	}
}

func TestLogLLMRequest(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger, err := New(logPath, "info")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Test successful LLM request
	prompt := "Generate a commit message for the following changes..."
	response := "feat: add new feature"
	logger.LogLLMRequest("openai", "gpt-4o-mini", prompt, response, true, nil)

	// Test failed LLM request
	testErr := fmt.Errorf("API rate limit exceeded")
	logger.LogLLMRequest("openai", "gpt-4o-mini", prompt, "", false, testErr)

	// Verify log file contains the entries
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "LLM request completed") {
		t.Error("Expected successful LLM request log entry")
	}
	if !strings.Contains(logContent, "LLM request failed") {
		t.Error("Expected failed LLM request log entry")
	}
	if !strings.Contains(logContent, "API rate limit exceeded") {
		t.Error("Expected error message in log")
	}
}

func TestLogUIAction(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger, err := New(logPath, "info")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	context := map[string]interface{}{
		"file":   "main.go",
		"action": "stage",
		"count":  1,
	}
	logger.LogUIAction("file_staged", context)

	// Verify log file contains the entry
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "UI action performed") {
		t.Error("Expected UI action log entry")
	}
	if !strings.Contains(logContent, "file_staged") {
		t.Error("Expected action name in log")
	}
}

func TestJSONFormat(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	logger, err := New(logPath, "info")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Log a simple message
	logger.Info("test message", slog.String("key", "value"))

	// Read and parse JSON
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		t.Fatal("No log entries found")
	}

	// Parse the first line as JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &logEntry); err != nil {
		t.Fatalf("Failed to parse log entry as JSON: %v", err)
	}

	// Verify required fields
	if logEntry["msg"] != "test message" {
		t.Errorf("Expected message 'test message', got %v", logEntry["msg"])
	}
	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level 'INFO', got %v", logEntry["level"])
	}
	if logEntry["key"] != "value" {
		t.Errorf("Expected key 'value', got %v", logEntry["key"])
	}
	if _, exists := logEntry["time"]; !exists {
		t.Error("Expected 'time' field in log entry")
	}
}

func TestGlobalLogger(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")

	// Initialize global logger
	if err := Init(logPath, "info"); err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}
	defer func() { _ = Close() }()

	// Test global logger functions
	Info("test info message")
	Error("test error message")
	LogAppStart("1.0.0", "/path/to/config")

	// Verify log file contains entries
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	if !strings.Contains(logContent, "test info message") {
		t.Error("Expected info message in log")
	}
	if !strings.Contains(logContent, "test error message") {
		t.Error("Expected error message in log")
	}
	if !strings.Contains(logContent, "Application started") {
		t.Error("Expected application start message in log")
	}
}

func TestTruncateString(t *testing.T) {
	testCases := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string that should be truncated", 20, "this is a very long ..."},
		{"", 5, ""},
		{"exact", 5, "exact"},
	}

	for _, tc := range testCases {
		result := truncateString(tc.input, tc.maxLen)
		if result != tc.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tc.input, tc.maxLen, result, tc.expected)
		}
	}
}
