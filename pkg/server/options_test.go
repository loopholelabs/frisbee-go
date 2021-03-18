package server

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestWithOptions(t *testing.T) {
	option := WithOptions(Options{
		multicore: true,
		async:     true,
		loops:     12,
		keepAlive: time.Minute * 3,
		logger:    nil,
	})

	options := loadOptions(option)

	assert.Equal(t, true, options.multicore)
	assert.Equal(t, true, options.async)
	assert.Equal(t, 12, options.loops)
	assert.Equal(t, time.Minute*3, options.keepAlive)
	assert.Equal(t, &DefaultLogger, options.logger)
}

func TestIndividualOptions(t *testing.T) {
	multicoreOption := WithMulticore(true)
	asyncOption := WithAsync(true)
	loopsOption := WithLoops(12)
	keepAliveOption := WithKeepAlive(time.Minute * 3)

	logger := zerolog.New(ioutil.Discard)

	loggerOption := WithLogger(&logger)

	options := loadOptions(multicoreOption, asyncOption, loopsOption, keepAliveOption, loggerOption)

	assert.Equal(t, true, options.multicore)
	assert.Equal(t, true, options.async)
	assert.Equal(t, 12, options.loops)
	assert.Equal(t, time.Minute*3, options.keepAlive)
	assert.Equal(t, &logger, options.logger)
}
