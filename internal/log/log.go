package log

import (
	"github.com/rs/zerolog"
)

type Logger struct {
	*zerolog.Logger
}

func Convert(logger *zerolog.Logger) Logger {
	return Logger{logger}
}

func (l Logger) Fatalf(format string, args ...interface{}) {
	l.Fatal().Msgf(format, args)
}

func (l Logger) Errorf(format string, args ...interface{}) {
	l.Error().Msgf(format, args)
}

func (l Logger) Debugf(format string, args ...interface{}) {
	l.Debug().Msgf(format, args)
}

func (l Logger) Warnf(format string, args ...interface{}) {
	l.Warn().Msgf(format, args)
}

func (l Logger) Infof(format string, args ...interface{}) {
	l.Info().Msgf(format, args)
}
