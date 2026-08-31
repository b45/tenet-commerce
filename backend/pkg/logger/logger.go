package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type contextKey string

const loggerContextKey contextKey = "logger_context_key"

var globalLogger *slog.Logger

func init() {
	globalLogger = NewLogger()
	slog.SetDefault(globalLogger)
}

// NewLogger initializes an slog.Logger instance based on environment variables:
// - LOG_LEVEL: debug | info | warn | error (default: info)
// - LOG_FORMAT: json | text (default: json)
func NewLogger() *slog.Logger {
	var level slog.Level
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug, // Include source file:line in debug mode
	}

	var handler slog.Handler
	if strings.ToLower(os.Getenv("LOG_FORMAT")) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// SetGlobalLogger replaces the global default logger
func SetGlobalLogger(l *slog.Logger) {
	globalLogger = l
	slog.SetDefault(l)
}

// GetLogger returns the current global logger
func GetLogger() *slog.Logger {
	return globalLogger
}

// NewContext embeds an slog.Logger into a standard Go context.Context
func NewContext(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, l)
}

// FromContext extracts an slog.Logger from context.Context.
// If no logger is present in context, it falls back to the global logger.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return globalLogger
	}
	if l, ok := ctx.Value(loggerContextKey).(*slog.Logger); ok && l != nil {
		return l
	}
	return globalLogger
}

// Convenience package-level logging helpers
func Debug(msg string, args ...any) {
	globalLogger.Debug(msg, args...)
}

func Info(msg string, args ...any) {
	globalLogger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	globalLogger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	globalLogger.Error(msg, args...)
}

func With(args ...any) *slog.Logger {
	return globalLogger.With(args...)
}
