package log

import (
	"github.com/rs/zerolog"
)

type Logger interface {
	Fatalf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Debugf(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Infof(format string, args ...interface{})
}

type logger struct {
	*zerolog.Logger
}

func Convert(zeroLogger *zerolog.Logger) Logger {
	return logger{zeroLogger}
}

func (l logger) Fatalf(format string, args ...interface{}) {
	l.Fatal().Msgf(format, args)
}

func (l logger) Errorf(format string, args ...interface{}) {
	l.Error().Msgf(format, args)
}

func (l logger) Debugf(format string, args ...interface{}) {
	l.Debug().Msgf(format, args)
}

func (l logger) Warnf(format string, args ...interface{}) {
	l.Warn().Msgf(format, args)
}

func (l logger) Infof(format string, args ...interface{}) {
	l.Info().Msgf(format, args)
}
