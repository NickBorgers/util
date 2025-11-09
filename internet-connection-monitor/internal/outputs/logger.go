package outputs

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/config"
	"github.com/nickborgers/monorepo/internet-connection-monitor/internal/models"
)

// Logger outputs test results as JSON to stdout
type Logger struct {
	logger *slog.Logger
	config *config.LoggingConfig
}

// NewLogger creates a new JSON logger
func NewLogger(cfg *config.LoggingConfig) (*Logger, error) {
	// Create a structured logger (only used for text format)
	// For JSON format, we write raw JSON directly in Write() method
	var logger *slog.Logger

	if cfg.Format != "json" {
		logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: parseLogLevel(cfg.Level),
		}))
	}

	return &Logger{
		logger: logger,
		config: cfg,
	}, nil
}

// Write outputs a test result to stdout as JSON
func (l *Logger) Write(result *models.TestResult) error {
	// For JSON format, output the raw JSON directly
	if l.config.Format == "json" {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		// Write directly to stdout for clean JSON lines
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))
		return nil
	}

	// For text format, use structured logging
	l.logger.Info("test_result",
		"site", result.Site.Name,
		"success", result.Status.Success,
		"http_status", result.Status.HTTPStatus,
		"total_ms", result.Timings.TotalDurationMs,
	)

	return nil
}

// Name returns the output module name
func (l *Logger) Name() string {
	return "logger"
}

// parseLogLevel converts string to slog.Level
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
