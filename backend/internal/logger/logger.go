package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger wraps slog.Logger with convenience methods
type Logger struct {
	*slog.Logger
}

// New creates a new logger instance
// In development, it uses text handler for readability
// In production, it uses JSON handler for structured logging
func New() *Logger {
	var handler slog.Handler
	
	// Check if we're in development mode (can be set via LOG_FORMAT env var)
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "text" || (logFormat == "" && os.Getenv("ENV") != "production") {
		// Text handler for development - more readable
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
			AddSource: true,
		}
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		// JSON handler for production - structured logging
		opts := &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	
	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *Logger {
	logger := l.Logger
	
	// Extract request ID from context if present
	if requestID := ctx.Value("request_id"); requestID != nil {
		logger = logger.With("request_id", requestID)
	}
	
	return &Logger{Logger: logger}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// ErrorWithErr logs an error message with an error field
func (l *Logger) ErrorWithErr(msg string, err error, args ...any) {
	args = append(args, "error", err)
	l.Logger.Error(msg, args...)
}

// Default logger instance
var defaultLogger = New()

// Default returns the default logger instance
func Default() *Logger {
	return defaultLogger
}

// SetDefault sets the default logger instance
func SetDefault(l *Logger) {
	defaultLogger = l
}
