package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

type TraceContextHandler struct {
	slog.Handler
}

func NewTraceContextHandler(h slog.Handler) slog.Handler {
	return &TraceContextHandler{Handler: h}
}

func (h *TraceContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if ctx != nil {
		spanCtx := trace.SpanFromContext(ctx).SpanContext()
		if spanCtx.IsValid() {
			r.AddAttrs(
				slog.String("trace_id", spanCtx.TraceID().String()),
				slog.String("span_id", spanCtx.SpanID().String()),
			)
		}
	}
	return h.Handler.Handle(ctx, r)
}

func NewLogger() *slog.Logger {
	env := getEnv("APP_ENV", "development")
	logLevelStr := getEnv("LOG_LEVEL", "info")
	level := parseLogLevel(logLevelStr)

	addSource := strings.EqualFold(env, "development") || level == slog.LevelDebug
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: addSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			keyLower := strings.ToLower(a.Key)
			if keyLower == "password" || keyLower == "token" || keyLower == "secret" || keyLower == "authorization" {
				a.Value = slog.StringValue("[REDACTED]")
			}

			return a
		},
	}

	baseHandler := slog.NewJSONHandler(os.Stdout, opts)
	handlerWithTrace := NewTraceContextHandler(baseHandler)
	logger := slog.New(handlerWithTrace)
	slog.SetDefault(logger)

	return logger

}

// Helper untuk membaca ENV dengan Fallback Value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}

// Helper untuk parsing string ENV ke slog.Level
func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
