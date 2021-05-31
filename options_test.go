/*
	Copyright 2021 Loophole Labs

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
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

func TestWithoutOptions(t *testing.T) {
	options := loadOptions()

	assert.Equal(t, time.Minute*3, options.KeepAlive)
	assert.Equal(t, time.Second*5, options.Heartbeat)
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

func TestDisableOptions(t *testing.T) {
	option := WithOptions(Options{
		KeepAlive: -1,
		Heartbeat: -1,
	})

	options := loadOptions(option)

	assert.Equal(t, time.Duration(-1), options.KeepAlive)
	assert.Equal(t, time.Duration(-1), options.Heartbeat)
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
