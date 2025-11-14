package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/lmittmann/tint"
	"github.com/rmitchellscott/stationmaster/internal/config"
)

var logger *slog.Logger

// Custom log levels
const (
	LevelBrowserless = slog.Level(-6) // More verbose than DEBUG (-4)
)

// ComponentTintHandler wraps tint.Handler to format component attributes as bracketed prefixes
type ComponentTintHandler struct {
	Handler slog.Handler
}

// Handle formats log records, extracting component attributes and formatting them as bracketed prefixes
func (h *ComponentTintHandler) Handle(ctx context.Context, r slog.Record) error {
	var component string
	var filteredAttrs []slog.Attr

	// Extract component attribute and filter out from other attributes
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "component" {
			component = a.Value.String()
		} else {
			filteredAttrs = append(filteredAttrs, a)
		}
		return true
	})

	// Create new record with modified message if component is present
	if component != "" {
		// Format component as uppercase bracketed prefix
		componentUpper := strings.ToUpper(strings.ReplaceAll(component, "-", " "))
		newMessage := fmt.Sprintf("[%s] %s", componentUpper, r.Message)
		
		// Create new record with the modified message
		newRecord := slog.NewRecord(r.Time, r.Level, newMessage, r.PC)
		
		// Add back the filtered attributes
		for _, attr := range filteredAttrs {
			newRecord.AddAttrs(attr)
		}
		
		return h.Handler.Handle(ctx, newRecord)
	}

	// If no component, handle as normal
	return h.Handler.Handle(ctx, r)
}

// Enabled delegates to the wrapped handler
func (h *ComponentTintHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, level)
}

// WithAttrs delegates to the wrapped handler
func (h *ComponentTintHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ComponentTintHandler{Handler: h.Handler.WithAttrs(attrs)}
}

// WithGroup delegates to the wrapped handler
func (h *ComponentTintHandler) WithGroup(name string) slog.Handler {
	return &ComponentTintHandler{Handler: h.Handler.WithGroup(name)}
}

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
		handler = &ComponentTintHandler{
			Handler: tint.NewHandler(os.Stderr, &tint.Options{
				Level:      level,
				TimeFormat: "15:04:05",
				NoColor:    false,
				ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
					if a.Key == slog.LevelKey {
						level := a.Value.Any().(slog.Level)
						if level == LevelBrowserless {
							return slog.Attr{Key: a.Key, Value: slog.StringValue("BROWSERLESS")}
						}
					}
					return a
				},
			}),
		}
	}

	logger = slog.New(handler)
	slog.SetDefault(logger)
}

// parseLogLevel converts string log level to slog.Level
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToUpper(levelStr) {
	case "BROWSERLESS":
		return LevelBrowserless
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

// Browserless logs a message at the custom BROWSERLESS level (-6)
// This is more verbose than DEBUG and specifically for browserless debugging
func Browserless(msg string, args ...any) {
	logger.Log(context.Background(), LevelBrowserless, msg, args...)
}

// Component-aware logging functions
// These functions automatically add the component attribute for structured logging

// DebugWithComponent logs a debug message with a component attribute
func DebugWithComponent(component, msg string, args ...any) {
	logger.Debug(msg, append([]any{"component", component}, args...)...)
}

// InfoWithComponent logs an info message with a component attribute
func InfoWithComponent(component, msg string, args ...any) {
	logger.Info(msg, append([]any{"component", component}, args...)...)
}

// WarnWithComponent logs a warning message with a component attribute
func WarnWithComponent(component, msg string, args ...any) {
	logger.Warn(msg, append([]any{"component", component}, args...)...)
}

// ErrorWithComponent logs an error message with a component attribute
func ErrorWithComponent(component, msg string, args ...any) {
	logger.Error(msg, append([]any{"component", component}, args...)...)
}

// ComponentLogger returns a logger pre-configured with a component attribute
func ComponentLogger(component string) *slog.Logger {
	return logger.With("component", component)
}

// Pre-configured component loggers for common components
// These provide easy access to loggers for frequently used components

// StartupLogger returns a logger pre-configured for startup operations
func StartupLogger() *slog.Logger {
	return ComponentLogger(ComponentStartup)
}

// ModelPollerLogger returns a logger pre-configured for model poller operations
func ModelPollerLogger() *slog.Logger {
	return ComponentLogger(ComponentModelPoller)
}

// FirmwarePollerLogger returns a logger pre-configured for firmware poller operations
func FirmwarePollerLogger() *slog.Logger {
	return ComponentLogger(ComponentFirmwarePoller)
}

// DatabaseLogger returns a logger pre-configured for database operations
func DatabaseLogger() *slog.Logger {
	return ComponentLogger(ComponentDatabase)
}

// AuthLogger returns a logger pre-configured for authentication operations
func AuthLogger() *slog.Logger {
	return ComponentLogger(ComponentAuth)
}

// APILogger returns a logger pre-configured for a specific API endpoint
func APILogger(endpoint string) *slog.Logger {
	var component string
	switch endpoint {
	case "setup":
		component = ComponentAPISetup
	case "display":
		component = ComponentAPIDisplay
	case "logs":
		component = ComponentAPILogs
	case "current_screen":
		component = ComponentAPIScreen
	default:
		component = fmt.Sprintf("api-%s", endpoint)
	}
	return ComponentLogger(component)
}
