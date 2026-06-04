package app

import (
	"io"
	"log/slog"
	"os"
)

func ConfigureLogging(output io.Writer) {
	if output == nil {
		output = os.Stderr
	}
	handler := slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug})
	slog.SetDefault(slog.New(handler))
}
