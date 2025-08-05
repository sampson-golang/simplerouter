package simplerouter

import (
	"context"
	"log/slog"
	"os"
)

var verbose bool = os.Getenv("DEBUG_SIMPLEROUTER") == "1"

func Verbose() {
	verbose = true
}

func Silent() {
	verbose = false
}

type loggerClient struct {
	slog.Handler
}

func (l *loggerClient) Enabled(ctx context.Context, level slog.Level) bool {
	return verbose
}

var logger *slog.Logger = slog.New(&loggerClient{slog.NewTextHandler(os.Stdout, nil)})
