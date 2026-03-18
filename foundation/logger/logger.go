package logger

import (
	"context"
	"io"
	"log"
	"log/slog"
)

type Logger struct {
	handler slog.Handler
	*slog.Logger
}

func New(w io.Writer, minLevel Level, serviceName string) *Logger {
	baseHandler := slog.NewJSONHandler(w, &slog.HandlerOptions{Level: minLevel})
	handler := baseHandler.WithAttrs([]slog.Attr{slog.String("service", serviceName)})

	return &Logger{
		handler: handler,
		Logger:  slog.New(handler),
	}
}

func NewStdLogger(logger *Logger, level Level) *log.Logger {
	return slog.NewLogLogger(logger.handler, level)
}

func (l *Logger) Debug(ctx context.Context, msg string, args ...any) {
	l.Logger.DebugContext(ctx, msg, args...)
}

func (l *Logger) Info(ctx context.Context, msg string, args ...any) {
	l.Logger.InfoContext(ctx, msg, args...)
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...any) {
	l.Logger.WarnContext(ctx, msg, args...)
}

func (l *Logger) Error(ctx context.Context, msg string, args ...any) {
	l.Logger.ErrorContext(ctx, msg, args...)
}
