package logger

import (
	"log/slog"
	"os"
)

// TextLogger returns a text logger than can write to file if filePath is provided or just log to stdout otherwise.
func TextLogger(filePath string) (*slog.Logger, error) {
	if filePath == "" {
		return slog.New(slog.NewTextHandler(os.Stdout, nil)), nil
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return slog.New(slog.NewTextHandler(f, nil)), nil
}
// JSONLogger returns a JSON logger than can write to file if filePath is provided or just log to stdout otherwise.
func JSONLogger(filePath string) (*slog.Logger, error) {
	if filePath == "" {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil)), nil
	}
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return slog.New(slog.NewJSONHandler(f, nil)), nil
}
