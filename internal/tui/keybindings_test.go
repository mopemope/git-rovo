package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/mopemope/git-rovo/internal/config"
	"github.com/mopemope/git-rovo/internal/logger"
)

func setupKeyBindingTest(t *testing.T) *KeyBindingManager {
	// Initialize logger for testing
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "test.log")
	if err := logger.Init(logPath, "info"); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Create test config without custom bindings for most tests
	cfg := &config.Config{}

	return NewKeyBindingManager(cfg)
}

func TestNewKeyBindingManager(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	if kbm == nil {
		t.Fatal("Expected key binding manager to be created")
	}

	if len(kbm.bindings) == 0 {
		t.Error("Expected default bindings to be loaded")
	}
}

func TestGetBinding(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	// Test existing binding
	binding, exists := kbm.GetBinding("q")
	if !exists {
		t.Error("Expected 'q' binding to exist")
	}
	if binding.Action != "quit" {
		t.Errorf("Expected action 'quit', got '%s'", binding.Action)
	}

	// Test non-existing binding
	_, exists = kbm.GetBinding("nonexistent")
	if exists {
		t.Error("Expected 'nonexistent' binding to not exist")
	}
}

func TestGetBindingsForView(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	// Test status view bindings
	statusBindings := kbm.GetBindingsForView(ViewModeStatus)
	if len(statusBindings) == 0 {
		t.Error("Expected status view to have bindings")
	}

	// Check for specific binding
	found := false
	for _, binding := range statusBindings {
		if binding.Action == "toggle_file" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected status view to have 'toggle_file' binding")
	}

	// Test diff view bindings
	diffBindings := kbm.GetBindingsForView(ViewModeDiff)
	if len(diffBindings) == 0 {
		t.Error("Expected diff view to have bindings")
	}
}

func TestKeyBindingHandleKeyPress(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	// Test valid key press in correct context
	action, handled := kbm.HandleKeyPress("q", ViewModeStatus)
	if !handled {
		t.Error("Expected 'q' to be handled in status view")
	}
	if action != "quit" {
		t.Errorf("Expected action 'quit', got '%s'", action)
	}

	// Test key press in wrong context
	_, handled = kbm.HandleKeyPress("space", ViewModeDiff)
	if handled {
		t.Error("Expected 'space' to not be handled in diff view")
	}

	// Test non-existent key
	_, handled = kbm.HandleKeyPress("nonexistent", ViewModeStatus)
	if handled {
		t.Error("Expected 'nonexistent' key to not be handled")
	}
}

func TestGetHelpText(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	helpText := kbm.GetHelpText(ViewModeStatus)
	if helpText == "" {
		t.Error("Expected non-empty help text for status view")
	}

	// Check for expected categories
	expectedCategories := []string{"Navigation", "Git Operations", "View Control"}
	for _, category := range expectedCategories {
		if !strings.Contains(helpText, category) {
			t.Errorf("Expected help text to contain category '%s'", category)
		}
	}
}

func TestGetFooterText(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	footerText := kbm.GetFooterText(ViewModeStatus)
	if footerText == "" {
		t.Error("Expected non-empty footer text for status view")
	}

	// Check for expected actions
	expectedActions := []string{"toggle", "commit", "quit"}
	for _, action := range expectedActions {
		if !strings.Contains(footerText, action) {
			t.Errorf("Expected footer text to contain action '%s'", action)
		}
	}
}

func TestCategorizeBinding(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	testCases := []struct {
		binding  KeyBinding
		expected string
	}{
		{KeyBinding{Action: "nav_up"}, "Navigation"},
		{KeyBinding{Action: "stage_file"}, "Git Operations"},
		{KeyBinding{Action: "toggle_file"}, "View Control"}, // toggle actions are categorized as View Control
		{KeyBinding{Action: "status"}, "View Control"},
		{KeyBinding{Action: "unknown_action"}, "Other"},
	}

	for _, tc := range testCases {
		result := kbm.categorizeBinding(tc.binding)
		if result != tc.expected {
			t.Errorf("categorizeBinding(%v) = %s, expected %s", tc.binding.Action, result, tc.expected)
		}
	}
}

func TestValidateKeyBinding(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	// Test valid binding
	err := kbm.ValidateKeyBinding("x", "custom_action")
	if err != nil {
		t.Errorf("Expected valid binding to pass validation, got error: %v", err)
	}

	// Test empty key
	err = kbm.ValidateKeyBinding("", "action")
	if err == nil {
		t.Error("Expected error for empty key")
	}

	// Test empty action
	err = kbm.ValidateKeyBinding("key", "")
	if err == nil {
		t.Error("Expected error for empty action")
	}

	// Test system key conflict
	err = kbm.ValidateKeyBinding("ctrl+c", "custom_action")
	if err == nil {
		t.Error("Expected error for system key conflict")
	}

	// Test allowed system key usage
	err = kbm.ValidateKeyBinding("ctrl+c", "quit")
	if err != nil {
		t.Errorf("Expected ctrl+c to be allowed for quit action, got error: %v", err)
	}
}

func TestAddCustomBinding(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	// Add custom binding
	err := kbm.AddCustomBinding("F1", "custom_help", "Show custom help", []ViewMode{ViewModeStatus})
	if err != nil {
		t.Fatalf("Failed to add custom binding: %v", err)
	}

	// Verify binding was added
	binding, exists := kbm.GetBinding("F1")
	if !exists {
		t.Error("Expected custom binding to be added")
	}
	if binding.Action != "custom_help" {
		t.Errorf("Expected action 'custom_help', got '%s'", binding.Action)
	}
}

func TestRemoveBinding(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	// Verify binding exists
	_, exists := kbm.GetBinding("q")
	if !exists {
		t.Fatal("Expected 'q' binding to exist initially")
	}

	// Remove binding
	kbm.RemoveBinding("q")

	// Verify binding was removed
	_, exists = kbm.GetBinding("q")
	if exists {
		t.Error("Expected 'q' binding to be removed")
	}
}

func TestListAllBindings(t *testing.T) {
	kbm := setupKeyBindingTest(t)
	defer func() { _ = logger.Close() }()

	allBindings := kbm.ListAllBindings()
	if len(allBindings) == 0 {
		t.Error("Expected at least some bindings to be listed")
	}

	// Verify it's a copy (modifying shouldn't affect original)
	originalCount := len(kbm.bindings)
	delete(allBindings, "q")
	if len(kbm.bindings) != originalCount {
		t.Error("Expected ListAllBindings to return a copy")
	}
}

func TestCustomBindingOverride(t *testing.T) {
	// Create config with custom binding
	cfg := &config.Config{
		UI: config.UIConfig{
			KeyBindings: map[string]string{
				"quit": "x", // Override quit key from 'q' to 'x'
			},
		},
	}

	kbm := NewKeyBindingManager(cfg)

	// Test that old binding is removed
	_, exists := kbm.GetBinding("q")
	if exists {
		t.Error("Expected old 'q' binding to be removed")
	}

	// Test that new binding exists
	binding, exists := kbm.GetBinding("x")
	if !exists {
		t.Error("Expected new 'x' binding to exist")
	}
	if binding.Action != "quit" {
		t.Errorf("Expected action 'quit', got '%s'", binding.Action)
	}
}
