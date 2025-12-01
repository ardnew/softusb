package pkg

import (
	"io"
	"log/slog"
	"os"
	"sync"
)

// Component identifies a subsystem for log filtering.
type Component string

// USB stack component identifiers.
const (
	ComponentDevice   Component = "device"
	ComponentHost     Component = "host"
	ComponentStack    Component = "stack"
	ComponentHAL      Component = "hal"
	ComponentTransfer Component = "transfer"
	ComponentEndpoint Component = "endpoint"
)

var (
	// DefaultLogger is the default logger used by the USB stack.
	DefaultLogger *slog.Logger

	// logLevel controls the minimum log level.
	logLevel = new(slog.LevelVar)

	// logMutex protects logger configuration.
	logMutex sync.RWMutex
)

func init() {
	logLevel.Set(slog.LevelInfo)
	DefaultLogger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

// SetLogLevel sets the minimum log level for all USB stack logging.
func SetLogLevel(level slog.Level) {
	logMutex.Lock()
	defer logMutex.Unlock()
	logLevel.Set(level)
}

// GetLogLevel returns the current minimum log level.
func GetLogLevel() slog.Level {
	logMutex.RLock()
	defer logMutex.RUnlock()
	return logLevel.Level()
}

// SetLogger replaces the default logger with a custom logger.
func SetLogger(logger *slog.Logger) {
	logMutex.Lock()
	defer logMutex.Unlock()
	DefaultLogger = logger
}

// NewLogger creates a new text logger writing to the given writer.
func NewLogger(w io.Writer, opts *slog.HandlerOptions) *slog.Logger {
	if opts == nil {
		opts = &slog.HandlerOptions{Level: logLevel}
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

// NewJSONLogger creates a new JSON logger writing to the given writer.
func NewJSONLogger(w io.Writer, opts *slog.HandlerOptions) *slog.Logger {
	if opts == nil {
		opts = &slog.HandlerOptions{Level: logLevel}
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}

// LogDebug logs a debug message with the given component.
func LogDebug(component Component, msg string, args ...any) {
	logMutex.RLock()
	logger := DefaultLogger
	logMutex.RUnlock()
	logger.Debug(msg, append([]any{"component", string(component)}, args...)...)
}

// LogInfo logs an info message with the given component.
func LogInfo(component Component, msg string, args ...any) {
	logMutex.RLock()
	logger := DefaultLogger
	logMutex.RUnlock()
	logger.Info(msg, append([]any{"component", string(component)}, args...)...)
}

// LogWarn logs a warning message with the given component.
func LogWarn(component Component, msg string, args ...any) {
	logMutex.RLock()
	logger := DefaultLogger
	logMutex.RUnlock()
	logger.Warn(msg, append([]any{"component", string(component)}, args...)...)
}

// LogError logs an error message with the given component.
func LogError(component Component, msg string, args ...any) {
	logMutex.RLock()
	logger := DefaultLogger
	logMutex.RUnlock()
	logger.Error(msg, append([]any{"component", string(component)}, args...)...)
}
