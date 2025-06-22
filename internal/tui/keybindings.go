package tui

import (
	"fmt"
	"strings"

	"github.com/mopemope/git-rovo/internal/config"
)

// KeyBinding represents a key binding with its action and description
type KeyBinding struct {
	Key         string
	Action      string
	Description string
	Context     []ViewMode // Which views this binding applies to
}

// KeyBindingManager manages all key bindings for the application
type KeyBindingManager struct {
	bindings map[string][]KeyBinding // Changed to slice to support multiple bindings per key
	config   *config.Config
}

// NewKeyBindingManager creates a new key binding manager
func NewKeyBindingManager(cfg *config.Config) *KeyBindingManager {
	manager := &KeyBindingManager{
		bindings: make(map[string][]KeyBinding), // Changed to slice
		config:   cfg,
	}
	manager.initializeDefaultBindings()
	manager.loadCustomBindings()
	return manager
}

// initializeDefaultBindings sets up the default key bindings
func (kbm *KeyBindingManager) initializeDefaultBindings() {
	defaultBindings := []KeyBinding{
		// Global bindings
		{"q", "quit", "Quit application", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog, ViewModeHelp}},
		{"ctrl+c", "quit", "Quit application", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog, ViewModeHelp}},
		{"h", "help", "Show help", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"r", "refresh", "Refresh current view", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"s", "status", "Switch to status view", []ViewMode{ViewModeDiff, ViewModeLog, ViewModeHelp}},
		{"d", "diff", "Switch to diff view", []ViewMode{ViewModeStatus, ViewModeLog, ViewModeHelp}},
		{"l", "log", "Switch to log view", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeHelp}},

		// Navigation bindings
		{"up", "nav_up", "Move cursor up", []ViewMode{ViewModeStatus, ViewModeLog}},
		{"down", "nav_down", "Move cursor down", []ViewMode{ViewModeStatus, ViewModeLog}},
		{"home", "nav_home", "Go to top", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"end", "nav_end", "Go to bottom", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"pgup", "nav_page_up", "Page up", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"ctrl+u", "nav_page_up", "Page up", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"pgdown", "nav_page_down", "Page down", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},
		{"ctrl+d", "nav_page_down", "Page down", []ViewMode{ViewModeStatus, ViewModeDiff, ViewModeLog}},

		// Status view specific
		{"space", "toggle_file", "Toggle file staging", []ViewMode{ViewModeStatus}},
		{"enter", "toggle_file", "Toggle file staging", []ViewMode{ViewModeStatus}},
		{"s", "stage_file", "Stage current file", []ViewMode{ViewModeStatus}},
		{"a", "stage_all", "Stage all files", []ViewMode{ViewModeStatus}},
		{"A", "unstage_all", "Unstage all files", []ViewMode{ViewModeStatus}},
		{"u", "unstage_file", "Unstage current file", []ViewMode{ViewModeStatus}},
		{"c", "commit", "Commit staged changes", []ViewMode{ViewModeStatus}},
		{"C", "quick_commit", "Quick commit with generated message", []ViewMode{ViewModeStatus}},
		{"1", "amend_commit", "Amend last commit", []ViewMode{ViewModeStatus}},
		{"g", "generate_message", "Generate commit message", []ViewMode{ViewModeStatus}},
		{"G", "regenerate_message", "Regenerate commit message", []ViewMode{ViewModeStatus}},
		{"R", "reset_file", "Reset current file", []ViewMode{ViewModeStatus}},
		{"k", "discard_changes", "Discard changes to current file", []ViewMode{ViewModeStatus}},
		{"tab", "toggle_section", "Toggle section", []ViewMode{ViewModeStatus}},
		{"ctrl+a", "select_all", "Select all files", []ViewMode{ViewModeStatus}},
		{"ctrl+n", "clear_selection", "Clear selection", []ViewMode{ViewModeStatus}},

		// Diff view specific
		{"left", "diff_prev_file", "Previous file", []ViewMode{ViewModeDiff}},
		{"right", "diff_next_file", "Next file", []ViewMode{ViewModeDiff}},
		{"n", "toggle_line_numbers", "Toggle line numbers", []ViewMode{ViewModeDiff}},
		{"w", "toggle_line_wrap", "Toggle line wrapping", []ViewMode{ViewModeDiff}},
		{"m", "cycle_diff_mode", "Cycle diff view mode", []ViewMode{ViewModeDiff}},
		{"t", "toggle_stats", "Toggle statistics", []ViewMode{ViewModeDiff}},
		{"T", "toggle_staged_unstaged", "Toggle staged/unstaged diff", []ViewMode{ViewModeDiff}},
		{"D", "show_staged_diff", "Show staged diff", []ViewMode{ViewModeStatus}},

		// Log view specific
		{"enter", "show_commit_details", "Show commit details", []ViewMode{ViewModeLog}},
		{"ctrl+c", "copy_commit_hash", "Copy commit hash", []ViewMode{ViewModeLog}},
		{"ctrl+g", "toggle_graph", "Toggle commit graph", []ViewMode{ViewModeLog}},

		// Help view
		{"any", "return_to_status", "Return to status view", []ViewMode{ViewModeHelp}},
	}

	for _, binding := range defaultBindings {
		kbm.bindings[binding.Key] = append(kbm.bindings[binding.Key], binding)
	}
}

// loadCustomBindings loads custom key bindings from configuration
func (kbm *KeyBindingManager) loadCustomBindings() {
	if kbm.config == nil || kbm.config.UI.KeyBindings == nil {
		return
	}

	for action, key := range kbm.config.UI.KeyBindings {
		// Find existing binding with this action
		for existingKey, bindingList := range kbm.bindings {
			for i, binding := range bindingList {
				if binding.Action == action {
					// Remove old binding
					kbm.bindings[existingKey] = append(bindingList[:i], bindingList[i+1:]...)
					if len(kbm.bindings[existingKey]) == 0 {
						delete(kbm.bindings, existingKey)
					}
					// Add new binding with custom key
					binding.Key = key
					kbm.bindings[key] = append(kbm.bindings[key], binding)
					break
				}
			}
		}
	}
}

// GetBinding returns the key binding for a given key and view
func (kbm *KeyBindingManager) GetBinding(key string) (KeyBinding, bool) {
	bindings, exists := kbm.bindings[key]
	if !exists || len(bindings) == 0 {
		return KeyBinding{}, false
	}
	// Return the first binding (for backward compatibility)
	return bindings[0], true
}

// GetBindingsForView returns all key bindings applicable to a specific view
func (kbm *KeyBindingManager) GetBindingsForView(view ViewMode) []KeyBinding {
	var viewBindings []KeyBinding
	for _, bindingList := range kbm.bindings {
		for _, binding := range bindingList {
			for _, context := range binding.Context {
				if context == view {
					viewBindings = append(viewBindings, binding)
					break
				}
			}
		}
	}
	return viewBindings
}

// GetHelpText returns formatted help text for a specific view
func (kbm *KeyBindingManager) GetHelpText(view ViewMode) string {
	viewBindings := kbm.GetBindingsForView(view)

	// Group bindings by category
	categories := map[string][]KeyBinding{
		"Navigation":     {},
		"File Actions":   {},
		"View Control":   {},
		"Git Operations": {},
		"Other":          {},
	}

	for _, binding := range viewBindings {
		category := kbm.categorizeBinding(binding)
		categories[category] = append(categories[category], binding)
	}

	var help strings.Builder

	for category, categoryBindings := range categories {
		if len(categoryBindings) == 0 {
			continue
		}

		help.WriteString(fmt.Sprintf("%s:\n", category))
		for _, binding := range categoryBindings {
			help.WriteString(fmt.Sprintf("  %-12s %s\n", binding.Key, binding.Description))
		}
		help.WriteString("\n")
	}

	return help.String()
}

// categorizeBinding categorizes a key binding
func (kbm *KeyBindingManager) categorizeBinding(binding KeyBinding) string {
	switch {
	case strings.Contains(binding.Action, "nav_"):
		return "Navigation"
	case strings.Contains(binding.Action, "stage") || strings.Contains(binding.Action, "commit") || strings.Contains(binding.Action, "reset"):
		return "Git Operations"
	case strings.Contains(binding.Action, "toggle") || strings.Contains(binding.Action, "show") || strings.Contains(binding.Action, "cycle"):
		return "View Control"
	case binding.Action == "status" || binding.Action == "diff" || binding.Action == "log" || binding.Action == "help":
		return "View Control"
	case strings.Contains(binding.Action, "file") || strings.Contains(binding.Action, "select"):
		return "File Actions"
	default:
		return "Other"
	}
}

// HandleKeyPress processes a key press and returns the corresponding action
func (kbm *KeyBindingManager) HandleKeyPress(key string, currentView ViewMode) (string, bool) {
	bindings, exists := kbm.bindings[key]
	if !exists {
		return "", false
	}

	// Find the binding that applies to the current view
	for _, binding := range bindings {
		for _, context := range binding.Context {
			if context == currentView {
				return binding.Action, true
			}
		}
	}

	return "", false
}

// GetFooterText returns the footer text with key bindings for the current view
func (kbm *KeyBindingManager) GetFooterText(view ViewMode) string {
	// Select most important bindings for footer
	var footerBindings []string

	switch view {
	case ViewModeStatus:
		importantActions := []string{"toggle_file", "stage_file", "stage_all", "commit", "amend_commit", "generate_message", "discard_changes", "diff", "log", "help", "quit"}
		for _, action := range importantActions {
			if key := kbm.getKeyForAction(action, view); key != "" {
				desc := kbm.getDescriptionForAction(action)
				footerBindings = append(footerBindings, fmt.Sprintf("%s:%s", key, desc))
			}
		}
	case ViewModeDiff:
		importantActions := []string{"status", "diff_prev_file", "diff_next_file", "toggle_line_numbers", "cycle_diff_mode", "help", "quit"}
		for _, action := range importantActions {
			if key := kbm.getKeyForAction(action, view); key != "" {
				desc := kbm.getDescriptionForAction(action)
				footerBindings = append(footerBindings, fmt.Sprintf("%s:%s", key, desc))
			}
		}
	case ViewModeLog:
		importantActions := []string{"show_commit_details", "copy_commit_hash", "status", "diff", "help", "quit"}
		for _, action := range importantActions {
			if key := kbm.getKeyForAction(action, view); key != "" {
				desc := kbm.getDescriptionForAction(action)
				footerBindings = append(footerBindings, fmt.Sprintf("%s:%s", key, desc))
			}
		}
	case ViewModeHelp:
		footerBindings = []string{"any key:return"}
	}

	return strings.Join(footerBindings, " | ")
}

// getKeyForAction finds the key for a given action in a specific view
func (kbm *KeyBindingManager) getKeyForAction(action string, view ViewMode) string {
	for key, bindingList := range kbm.bindings {
		for _, binding := range bindingList {
			if binding.Action == action {
				for _, context := range binding.Context {
					if context == view {
						return key
					}
				}
			}
		}
	}
	return ""
}

// getDescriptionForAction gets a short description for an action
func (kbm *KeyBindingManager) getDescriptionForAction(action string) string {
	shortDescriptions := map[string]string{
		"toggle_file":         "toggle",
		"stage_all":           "stage all",
		"commit":              "commit",
		"amend_commit":        "amend",
		"generate_message":    "generate",
		"discard_changes":     "discard",
		"diff":                "diff",
		"log":                 "log",
		"help":                "help",
		"quit":                "quit",
		"diff_prev_file":      "prev",
		"diff_next_file":      "next",
		"toggle_line_numbers": "numbers",
		"cycle_diff_mode":     "mode",
		"status":              "back",
		"show_commit_details": "details",
		"copy_commit_hash":    "copy",
	}

	if desc, exists := shortDescriptions[action]; exists {
		return desc
	}
	return action
}

// ValidateKeyBinding validates a key binding configuration
func (kbm *KeyBindingManager) ValidateKeyBinding(key, action string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}
	if action == "" {
		return fmt.Errorf("action cannot be empty")
	}

	// Check for conflicts with system keys
	systemKeys := []string{"ctrl+c", "ctrl+z", "ctrl+d"}
	for _, sysKey := range systemKeys {
		if key == sysKey && action != "quit" && action != "nav_page_down" {
			return fmt.Errorf("key %s is reserved for system use", key)
		}
	}

	return nil
}

// AddCustomBinding adds a custom key binding
func (kbm *KeyBindingManager) AddCustomBinding(key, action, description string, contexts []ViewMode) error {
	if err := kbm.ValidateKeyBinding(key, action); err != nil {
		return err
	}

	binding := KeyBinding{
		Key:         key,
		Action:      action,
		Description: description,
		Context:     contexts,
	}

	kbm.bindings[key] = append(kbm.bindings[key], binding)
	return nil
}

// RemoveBinding removes a key binding
func (kbm *KeyBindingManager) RemoveBinding(key string) {
	delete(kbm.bindings, key)
}

// ListAllBindings returns all key bindings
func (kbm *KeyBindingManager) ListAllBindings() map[string][]KeyBinding {
	result := make(map[string][]KeyBinding)
	for k, v := range kbm.bindings {
		result[k] = make([]KeyBinding, len(v))
		copy(result[k], v)
	}
	return result
}
