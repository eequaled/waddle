package config

import (
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	DataDir           string
	RetentionDays     int
	BackupHour        int
	EmbeddingModel    string
	OCRBatchSize      int
	SynthesisInterval time.Duration
	Port              string
}

func DefaultConfig() Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".waddle")

	return Config{
		DataDir:           dataDir,
		RetentionDays:     30,
		BackupHour:        2,
		EmbeddingModel:    "nomic-embed-text",
		OCRBatchSize:      10,
		SynthesisInterval: 1 * time.Hour,
		Port:              "8080",
	}
}
