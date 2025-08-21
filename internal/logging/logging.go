package logging

import (
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"github.com/rmitchellscott/stationmaster/internal/config"
)

var logger *slog.Logger

func init() {
	setupLogger()
}

// setupLogger initializes the structured logger with tint handler
func setupLogger() {
	level := parseLogLevel(config.Get("LOG_LEVEL", "INFO"))
	format := strings.ToLower(config.Get("LOG_FORMAT", "text"))

	var handler slog.Handler

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = tint.NewHandler(os.Stderr, &tint.Options{
			Level:      level,
			TimeFormat: "15:04:05",
		})
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// GetLogger returns the configured structured logger
func GetLogger() *slog.Logger {
	return logger
}

// Debug logs a debug message with optional key-value pairs
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs an info message with optional key-value pairs
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs a warning message with optional key-value pairs
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs an error message with optional key-value pairs
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

// WithUser returns a logger with user context
func WithUser(username string) *slog.Logger {
	if username == "" {
		return logger
	}
	return logger.With("user", username)
}



// IsDebugEnabled returns true if debug logging is enabled
// This helps maintain compatibility with existing debug checks
func IsDebugEnabled() bool {
	return logger.Enabled(nil, slog.LevelDebug)
}
