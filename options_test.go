/*
	Copyright 2022 Loophole Labs

	Licensed under the Apache License, Version 2.0 (the "License");
	you may not use this file except in compliance with the License.
	You may obtain a copy of the License at

		   http://www.apache.org/licenses/LICENSE-2.0

	Unless required by applicable law or agreed to in writing, software
	distributed under the License is distributed on an "AS IS" BASIS,
	WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	See the License for the specific language governing permissions and
	limitations under the License.
*/

package frisbee

import (
	"crypto/tls"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io"
	"testing"
	"time"
)

func TestWithoutOptions(t *testing.T) {
	t.Parallel()

	options := loadOptions()

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, time.Second*5, options.Heartbeat)
	assert.Equal(t, &DefaultLogger, options.Logger)
	assert.Nil(t, options.TLSConfig)
}

func TestWithOptions(t *testing.T) {
	t.Parallel()

	option := WithOptions(Options{
		KeepAlive: time.Minute * 6,
		Heartbeat: time.Second * 60,
		Logger:    nil,
		TLSConfig: &tls.Config{},
	})

	options := loadOptions(option)

	assert.Equal(t, time.Minute*6, options.KeepAlive)
	assert.Equal(t, time.Second*60, options.Heartbeat)
	assert.Equal(t, &DefaultLogger, options.Logger)
	assert.Equal(t, &tls.Config{}, options.TLSConfig)
}

func TestDisableOptions(t *testing.T) {
	t.Parallel()

	option := WithOptions(Options{
		KeepAlive: -1,
		Heartbeat: -1,
	})

	options := loadOptions(option)

	assert.Equal(t, time.Duration(-1), options.KeepAlive)
	assert.Equal(t, time.Duration(-1), options.Heartbeat)
	assert.Equal(t, &DefaultLogger, options.Logger)
	assert.Nil(t, options.TLSConfig)
}

func TestIndividualOptions(t *testing.T) {
	t.Parallel()

	logger := zerolog.New(io.Discard)
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	keepAliveOption := WithKeepAlive(time.Minute * 6)
	heartbeatOption := WithHeartbeat(time.Second * 60)
	loggerOption := WithLogger(&logger)
	TLSOption := WithTLS(tlsConfig)

	options := loadOptions(keepAliveOption, loggerOption, heartbeatOption, TLSOption)

	assert.Equal(t, time.Minute*6, options.KeepAlive)
	assert.Equal(t, time.Second*60, options.Heartbeat)
	assert.Equal(t, &logger, options.Logger)
	assert.Equal(t, tlsConfig, options.TLSConfig)
}
