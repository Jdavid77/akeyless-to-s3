package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init initializes the global logger with the specified log level and format
func Init(logLevel, logFormat string) {
	// Set up zerolog time format
	zerolog.TimeFieldFormat = time.RFC3339

	// Parse log level
	level := parseLogLevel(logLevel)
	zerolog.SetGlobalLevel(level)

	// Configure output writer based on format
	var output io.Writer = os.Stdout
	if strings.ToLower(logFormat) == "console" {
		// Use human-readable console format
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
		}
	}
	// Default to JSON format if not "console"

	// Configure global logger
	log.Logger = zerolog.New(output).
		With().
		Timestamp().
		Caller().
		Logger()
}

// parseLogLevel converts a string log level to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	default:
		return zerolog.InfoLevel
	}
}
