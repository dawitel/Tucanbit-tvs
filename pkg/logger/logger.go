package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Level      string `yaml:"level"`
	TimeFormat string `yaml:"time_format"`
	Pretty     bool   `yaml:"pretty"`
}

func New() zerolog.Logger {
	return NewWithConfig(Config{
		Level:      "info",
		TimeFormat: time.RFC3339,
		Pretty:     false,
	})
}

func NewWithConfig(config Config) zerolog.Logger {
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	if config.TimeFormat != "" {
		zerolog.TimeFieldFormat = config.TimeFormat
	}

	var logger zerolog.Logger

	if config.Pretty {
		logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			FormatLevel: func(i interface{}) string {
				return colorizeLevel(i.(string))
			},
		})
	} else {
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}

	logger = logger.With().
		Str("service", "tvs").
		Str("version", "1.0.0").
		Logger()

	return logger
}

func colorizeLevel(level string) string {
	switch level {
	case "trace":
		return "\033[35m" + level + "\033[0m" // Magenta
	case "debug":
		return "\033[36m" + level + "\033[0m" // Cyan
	case "info":
		return "\033[32m" + level + "\033[0m" // Green
	case "warn":
		return "\033[33m" + level + "\033[0m" // Yellow
	case "error":
		return "\033[31m" + level + "\033[0m" // Red
	case "fatal":
		return "\033[91m" + level + "\033[0m" // Bright Red
	case "panic":
		return "\033[91m" + level + "\033[0m" // Bright Red
	default:
		return level
	}
}
