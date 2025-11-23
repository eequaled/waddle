package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Logger struct {
	baseDir string
}

func NewLogger(baseDir string) (*Logger, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	return &Logger{baseDir: baseDir}, nil
}

func (l *Logger) Log(content string) error {
	fileName := time.Now().Format("2006-01-02") + ".txt"
	filePath := filepath.Join(l.baseDir, fileName)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}
