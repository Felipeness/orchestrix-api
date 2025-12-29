package observability

import (
	"context"
	"log/slog"
	"os"
)

// Logger is the global structured logger
var Logger *slog.Logger

// InitLogger initializes the structured logger
func InitLogger(level string, format string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename fields for better compatibility with log aggregators
			if a.Key == slog.TimeKey {
				a.Key = "timestamp"
			}
			if a.Key == slog.LevelKey {
				a.Key = "level"
			}
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	}

	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	Logger = slog.New(handler)
	slog.SetDefault(Logger)
}

// WithContext adds common context fields to the logger
func WithContext(ctx context.Context) *slog.Logger {
	logger := Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Add trace ID if available
	if traceID := ctx.Value("trace_id"); traceID != nil {
		logger = logger.With("trace_id", traceID)
	}

	// Add request ID if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		logger = logger.With("request_id", requestID)
	}

	// Add tenant ID if available
	if tenantID := ctx.Value("tenant_id"); tenantID != nil {
		logger = logger.With("tenant_id", tenantID)
	}

	// Add user ID if available
	if userID := ctx.Value("user_id"); userID != nil {
		logger = logger.With("user_id", userID)
	}

	return logger
}

// LogError logs an error with context
func LogError(ctx context.Context, msg string, err error, attrs ...slog.Attr) {
	logger := WithContext(ctx)
	args := make([]any, 0, len(attrs)*2+2)
	args = append(args, "error", err.Error())
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value.Any())
	}
	logger.Error(msg, args...)
}

// LogInfo logs an info message with context
func LogInfo(ctx context.Context, msg string, attrs ...slog.Attr) {
	logger := WithContext(ctx)
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value.Any())
	}
	logger.Info(msg, args...)
}

// LogDebug logs a debug message with context
func LogDebug(ctx context.Context, msg string, attrs ...slog.Attr) {
	logger := WithContext(ctx)
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value.Any())
	}
	logger.Debug(msg, args...)
}

// LogWarn logs a warning message with context
func LogWarn(ctx context.Context, msg string, attrs ...slog.Attr) {
	logger := WithContext(ctx)
	args := make([]any, 0, len(attrs)*2)
	for _, attr := range attrs {
		args = append(args, attr.Key, attr.Value.Any())
	}
	logger.Warn(msg, args...)
}
