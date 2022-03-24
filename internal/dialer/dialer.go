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

package dialer

import (
	"crypto/tls"
	"net"
	"time"
)

// Retry is a simple net.Dialer that retries dialing NumRetries times.
type Retry struct {
	*net.Dialer
	NumRetries int
}

// NewRetry returns a Retry Dialer with default values.
func NewRetry() *Retry {
	return &Retry{
		Dialer: &net.Dialer{
			Timeout:   time.Second,
			KeepAlive: time.Second * 15,
		},
		NumRetries: 10,
	}
}

// Dial calls the underlying *net.Dial to dial a net.Conn, but retries on failure
func (r *Retry) Dial(network string, address string) (c net.Conn, err error) {
	for i := 0; i < r.NumRetries; i++ {
		c, err = r.Dialer.Dial(network, address)
		if err == nil {
			return
		}
	}
	return
}

// DialTLS creates a new TLS Dialer using the underlying *net.Dial and uses it to dial a net.Conn, but retries on failure
func (r *Retry) DialTLS(network string, address string, config *tls.Config) (c net.Conn, err error) {
	d := &tls.Dialer{
		NetDialer: r.Dialer,
		Config:    config,
	}
	for i := 0; i < r.NumRetries; i++ {
		c, err = d.Dial(network, address)
		if err == nil {
			return
		}
	}
	return
}
