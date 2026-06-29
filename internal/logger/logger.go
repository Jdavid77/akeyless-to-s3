package logger

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Init configures the global logger and returns it for injection into packages.
func Init(logLevel, logFormat string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	level := parseLogLevel(logLevel)
	zerolog.SetGlobalLevel(level)

	var output io.Writer = os.Stdout
	if strings.ToLower(logFormat) == "console" {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
		}
	}

	l := zerolog.New(output).With().Timestamp().Caller().Logger()
	log.Logger = l
	return l
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
