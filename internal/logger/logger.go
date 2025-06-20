package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Logger wraps slog.Logger with additional functionality
type Logger struct {
	*slog.Logger
	filePath string
	file     *os.File
}

// New creates a new logger instance with JSON format output to file
func New(filePath string, level string) (*Logger, error) {
	// Parse log level
	var logLevel slog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open or create log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create JSON handler
	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
	})

	// Create logger
	logger := slog.New(handler)

	return &Logger{
		Logger:   logger,
		filePath: filePath,
		file:     file,
	}, nil
}

// Close closes the log file
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// GetFilePath returns the log file path
func (l *Logger) GetFilePath() string {
	return l.filePath
}

// LogGitOperation logs Git operations with structured data
func (l *Logger) LogGitOperation(operation string, args []string, workDir string, success bool, output string, err error) {
	if err != nil {
		l.Error("Git operation failed",
			slog.String("operation", operation),
			slog.Any("args", args),
			slog.String("work_dir", workDir),
			slog.Bool("success", success),
			slog.String("output", output),
			slog.String("error", err.Error()),
		)
	} else {
		l.Info("Git operation completed",
			slog.String("operation", operation),
			slog.Any("args", args),
			slog.String("work_dir", workDir),
			slog.Bool("success", success),
			slog.String("output", output),
		)
	}
}

// LogLLMRequest logs LLM API requests with structured data
func (l *Logger) LogLLMRequest(provider string, model string, prompt string, response string, success bool, err error) {
	if err != nil {
		l.Error("LLM request failed",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("prompt_preview", truncateString(prompt, 200)),
			slog.String("response_preview", truncateString(response, 200)),
			slog.Bool("success", success),
			slog.String("error", err.Error()),
		)
	} else {
		l.Info("LLM request completed",
			slog.String("provider", provider),
			slog.String("model", model),
			slog.String("prompt_preview", truncateString(prompt, 200)),
			slog.String("response_preview", truncateString(response, 200)),
			slog.Bool("success", success),
		)
	}
}

// LogUIAction logs UI actions and user interactions
func (l *Logger) LogUIAction(action string, context map[string]interface{}) {
	args := []any{slog.String("action", action)}
	for key, value := range context {
		args = append(args, slog.Any(key, value))
	}
	l.Info("UI action performed", args...)
}

// LogAppStart logs application startup
func (l *Logger) LogAppStart(version string, configPath string) {
	l.Info("Application started",
		slog.String("version", version),
		slog.String("config_path", configPath),
		slog.String("log_path", l.filePath),
	)
}

// LogAppStop logs application shutdown
func (l *Logger) LogAppStop() {
	l.Info("Application stopped")
}

// LogConfigLoad logs configuration loading
func (l *Logger) LogConfigLoad(configPath string, success bool, err error) {
	if err != nil {
		l.Error("Configuration loading failed",
			slog.String("config_path", configPath),
			slog.Bool("success", success),
			slog.String("error", err.Error()),
		)
	} else {
		l.Info("Configuration loaded",
			slog.String("config_path", configPath),
			slog.Bool("success", success),
		)
	}
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger
func Init(filePath string, level string) error {
	logger, err := New(filePath, level)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// Get returns the global logger instance
func Get() *Logger {
	return globalLogger
}

// Close closes the global logger
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}

// Convenience functions for global logger
func Debug(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Debug(msg, args...)
	}
}

func Info(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Info(msg, args...)
	}
}

func Warn(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Warn(msg, args...)
	}
}

func Error(msg string, args ...any) {
	if globalLogger != nil {
		globalLogger.Error(msg, args...)
	}
}

func LogGitOperation(operation string, args []string, workDir string, success bool, output string, err error) {
	if globalLogger != nil {
		globalLogger.LogGitOperation(operation, args, workDir, success, output, err)
	}
}

func LogLLMRequest(provider string, model string, prompt string, response string, success bool, err error) {
	if globalLogger != nil {
		globalLogger.LogLLMRequest(provider, model, prompt, response, success, err)
	}
}

func LogUIAction(action string, context map[string]interface{}) {
	if globalLogger != nil {
		globalLogger.LogUIAction(action, context)
	}
}

func LogAppStart(version string, configPath string) {
	if globalLogger != nil {
		globalLogger.LogAppStart(version, configPath)
	}
}

func LogAppStop() {
	if globalLogger != nil {
		globalLogger.LogAppStop()
	}
}

func LogConfigLoad(configPath string, success bool, err error) {
	if globalLogger != nil {
		globalLogger.LogConfigLoad(configPath, success, err)
	}
}
