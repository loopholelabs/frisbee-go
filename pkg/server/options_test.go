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

	assert.Equal(t, false, options.Multicore)
	assert.Equal(t, false, options.Async)
	assert.Equal(t, 16, options.Loops)
	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, &DefaultLogger, options.Logger)
}

func TestWithOptions(t *testing.T) {
	option := WithOptions(Options{
		Multicore: true,
		Async:     true,
		Loops:     12,
		KeepAlive: time.Minute * 3,
		Logger:    nil,
	})

	options := LoadOptions(option)

	assert.Equal(t, true, options.Multicore)
	assert.Equal(t, true, options.Async)
	assert.Equal(t, 12, options.Loops)
	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, &DefaultLogger, options.Logger)
}

func TestIndividualOptions(t *testing.T) {
	multicoreOption := WithMulticore(true)
	asyncOption := WithAsync(true)
	loopsOption := WithLoops(12)
	keepAliveOption := WithKeepAlive(time.Minute * 3)

	logger := zerolog.New(ioutil.Discard)

	loggerOption := WithLogger(&logger)

	options := LoadOptions(multicoreOption, asyncOption, loopsOption, keepAliveOption, loggerOption)

	assert.Equal(t, true, options.Multicore)
	assert.Equal(t, true, options.Async)
	assert.Equal(t, 12, options.Loops)
	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, &logger, options.Logger)
}
