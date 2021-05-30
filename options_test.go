package frisbee

import (
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestWithoutOptions(t *testing.T) {
	options := loadOptions()

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, time.Second*30, options.Heartbeat)
	assert.Equal(t, &DefaultLogger, options.Logger)
}

func TestWithOptions(t *testing.T) {
	option := WithOptions(Options{
		KeepAlive: time.Minute * 6,
		Heartbeat: time.Second * 60,
		Logger:    nil,
	})

	options := loadOptions(option)

	assert.Equal(t, time.Minute*6, options.KeepAlive)
	assert.Equal(t, time.Second*60, options.Heartbeat)
	assert.Equal(t, &DefaultLogger, options.Logger)
}

func TestIndividualOptions(t *testing.T) {
	keepAliveOption := WithKeepAlive(time.Minute * 6)
	heartbeatOption := WithHeartbeat(time.Second * 60)

	logger := zerolog.New(ioutil.Discard)

	loggerOption := WithLogger(&logger)

	options := loadOptions(keepAliveOption, loggerOption, heartbeatOption)

	assert.Equal(t, time.Minute*6, options.KeepAlive)
	assert.Equal(t, time.Second*60, options.Heartbeat)
	assert.Equal(t, &logger, options.Logger)
}
