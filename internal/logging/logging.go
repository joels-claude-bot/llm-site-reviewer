package logging

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

func InitFromEnv() {
	Init(os.Getenv("LOG_LEVEL"), os.Stdout)
}

func Init(levelName string, output io.Writer) {
	var level slog.Level
	switch strings.ToLower(levelName) {
	case "error":
		level = slog.LevelError
	case "warn":
		level = slog.LevelWarn
	case "debug":
		level = slog.LevelDebug
	default:
		level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level})))
}
