package logger

import (
	"log/slog"
	"os"
)

// New returns a configured slog.Logger with text output and INFO level.
func New() *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}