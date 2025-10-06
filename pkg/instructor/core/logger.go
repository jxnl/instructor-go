package core

import (
	"context"
	"io"
	"log/slog"
	"os"
)

// Logger interface allows users to inject their own logger implementation
// By default, the library uses a no-op logger (silent)
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// slogLogger wraps slog.Logger to implement our Logger interface
type slogLogger struct {
	logger *slog.Logger
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{logger: l.logger.With(args...)}
}

// noopLogger is a no-op logger that does nothing (default behavior)
type noopLogger struct{}

func (l *noopLogger) Debug(msg string, args ...any) {}
func (l *noopLogger) Info(msg string, args ...any)  {}
func (l *noopLogger) Warn(msg string, args ...any)  {}
func (l *noopLogger) Error(msg string, args ...any) {}
func (l *noopLogger) With(args ...any) Logger       { return l }

// NewNoopLogger returns a logger that does nothing (silent)
// This is the default logger for the library
func NewNoopLogger() Logger {
	return &noopLogger{}
}

// NewLogger creates a new structured logger using slog
// Example: NewLogger(os.Stderr, slog.LevelDebug)
func NewLogger(w io.Writer, level slog.Level) Logger {
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return &slogLogger{logger: slog.New(handler)}
}

// NewTextLogger creates a human-readable text logger
// Example: NewTextLogger(os.Stderr, slog.LevelInfo)
func NewTextLogger(w io.Writer, level slog.Level) Logger {
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return &slogLogger{logger: slog.New(handler)}
}

// FromSlog wraps an existing *slog.Logger
// Example: FromSlog(slog.Default())
func FromSlog(logger *slog.Logger) Logger {
	return &slogLogger{logger: logger}
}

// NewLoggerFromString creates a logger from a string specification
// Supported formats:
//   - "debug", "info", "warn", "error" - Text logger at specified level to stderr
//   - "json" - JSON logger at INFO level to stderr
//   - "json:debug", "json:info", "json:warn", "json:error" - JSON logger at specified level
//   - "off" or "" - No-op logger (silent)
//
// Examples:
//   - NewLoggerFromString("debug") - Text DEBUG to stderr
//   - NewLoggerFromString("json") - JSON INFO to stderr
//   - NewLoggerFromString("json:debug") - JSON DEBUG to stderr
func NewLoggerFromString(spec string) Logger {
	if spec == "" || spec == "off" {
		return NewNoopLogger()
	}

	// Check for json format
	if spec == "json" {
		return NewLogger(getStderr(), slog.LevelInfo)
	}

	// Parse "json:level" or "level" format
	var format string
	var levelStr string

	if idx := len(spec); idx > 5 && spec[:5] == "json:" {
		format = "json"
		levelStr = spec[5:]
	} else {
		format = "text"
		levelStr = spec
	}

	// Parse level
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		// Default to info for unrecognized levels
		level = slog.LevelInfo
	}

	// Create appropriate logger
	if format == "json" {
		return NewLogger(getStderr(), level)
	}

	return NewTextLogger(getStderr(), level)
}

// getStderr returns os.Stderr as io.Writer
func getStderr() io.Writer {
	return os.Stderr
}

// contextKey is the type for context keys used in this package
type contextKey string

const loggerContextKey contextKey = "instructor-logger"

// ContextWithLogger adds a logger to the context
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}

// LoggerFromContext retrieves the logger from context
// Returns a no-op logger if none is set
func LoggerFromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerContextKey).(Logger); ok {
		return logger
	}
	return NewNoopLogger()
}
