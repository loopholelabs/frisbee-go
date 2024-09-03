// SPDX-License-Identifier: Apache-2.0

package frisbee

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/loopholelabs/logging"
)

func TestWithoutOptions(t *testing.T) {
	t.Parallel()

	options := loadOptions()

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.NotNil(t, options.Logger)
	assert.Nil(t, options.TLSConfig)
}

func TestWithOptions(t *testing.T) {
	t.Parallel()

	option := WithOptions(Options{
		KeepAlive: time.Minute * 6,
		Logger:    nil,
		TLSConfig: &tls.Config{},
	})

	options := loadOptions(option)

	assert.Equal(t, time.Minute*6, options.KeepAlive)
	assert.NotNil(t, options.Logger)
	assert.Equal(t, &tls.Config{}, options.TLSConfig)
}

func TestDisableOptions(t *testing.T) {
	t.Parallel()

	option := WithOptions(Options{
		KeepAlive: -1,
	})

	options := loadOptions(option)

	assert.Equal(t, time.Duration(-1), options.KeepAlive)
	assert.NotNil(t, options.Logger)
	assert.Nil(t, options.TLSConfig)
}

func TestIndividualOptions(t *testing.T) {
	t.Parallel()

	logger := logging.Test(t, logging.Noop, t.Name())
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	keepAliveOption := WithKeepAlive(time.Minute * 6)
	loggerOption := WithLogger(logger)
	TLSOption := WithTLS(tlsConfig)

	options := loadOptions(keepAliveOption, loggerOption, TLSOption)

	assert.Equal(t, time.Minute*6, options.KeepAlive)
	assert.Equal(t, logger, options.Logger)
	assert.Equal(t, tlsConfig, options.TLSConfig)
}
