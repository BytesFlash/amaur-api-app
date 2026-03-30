package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New(level string) zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339},
	).Level(lvl).With().Timestamp().Caller().Logger()
}
