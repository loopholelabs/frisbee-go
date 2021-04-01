package server

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestWithoutOptions(t *testing.T) {
	options := LoadOptions()

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, &DefaultLogger, options.Logger)
}

func TestWithOptions(t *testing.T) {
	option := WithOptions(Options{
		KeepAlive: time.Minute * 3,
		Logger:    nil,
	})

	options := LoadOptions(option)

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, &DefaultLogger, options.Logger)
}

func TestIndividualOptions(t *testing.T) {
	keepAliveOption := WithKeepAlive(time.Minute * 3)

	logger := zerolog.New(ioutil.Discard)

	loggerOption := WithLogger(&logger)

	options := LoadOptions(keepAliveOption, loggerOption)

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, &logger, options.Logger)
}
