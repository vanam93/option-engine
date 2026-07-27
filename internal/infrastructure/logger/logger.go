package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ctxKey string

const (
	keyModule        ctxKey = "module"
	keyProvider      ctxKey = "provider"
	keyRequestID     ctxKey = "request_id"
	keyCorrelationID ctxKey = "correlation_id"
)

// Fields carries structured log dimensions.
type Fields struct {
	Module        string
	Provider      string
	RequestID     string
	CorrelationID string
	Latency       time.Duration
	Extra         []any
}

// New creates a structured logger based on level and format.
func New(level, format string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{Key: "timestamp", Value: a.Value}
			}
			return a
		},
	}

	var handler slog.Handler
	if strings.ToLower(format) == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

// WithModule returns a child logger tagged with module name.
func WithModule(log *slog.Logger, module string) *slog.Logger {
	return log.With("module", module)
}

// WithProvider returns a child logger tagged with provider name.
func WithProvider(log *slog.Logger, provider string) *slog.Logger {
	return log.With("provider", provider)
}

// WithFields returns a child logger with standard structured fields.
func WithFields(log *slog.Logger, f Fields) *slog.Logger {
	attrs := make([]any, 0, 8)
	if f.Module != "" {
		attrs = append(attrs, "module", f.Module)
	}
	if f.Provider != "" {
		attrs = append(attrs, "provider", f.Provider)
	}
	if f.RequestID != "" {
		attrs = append(attrs, "request_id", f.RequestID)
	}
	if f.CorrelationID != "" {
		attrs = append(attrs, "correlation_id", f.CorrelationID)
	}
	if f.Latency > 0 {
		attrs = append(attrs, "latency_ms", f.Latency.Milliseconds())
	}
	attrs = append(attrs, f.Extra...)
	if len(attrs) == 0 {
		return log
	}
	return log.With(attrs...)
}

// Context helpers for request-scoped logging.

func ContextWithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, keyRequestID, id)
}

func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		id = uuid.New().String()
	}
	return context.WithValue(ctx, keyCorrelationID, id)
}

func ContextWithModule(ctx context.Context, module string) context.Context {
	return context.WithValue(ctx, keyModule, module)
}

func FromContext(ctx context.Context, log *slog.Logger) *slog.Logger {
	if ctx == nil {
		return log
	}
	f := Fields{}
	if v, ok := ctx.Value(keyModule).(string); ok {
		f.Module = v
	}
	if v, ok := ctx.Value(keyProvider).(string); ok {
		f.Provider = v
	}
	if v, ok := ctx.Value(keyRequestID).(string); ok {
		f.RequestID = v
	}
	if v, ok := ctx.Value(keyCorrelationID).(string); ok {
		f.CorrelationID = v
	}
	return WithFields(log, f)
}
