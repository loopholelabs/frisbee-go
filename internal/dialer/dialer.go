// SPDX-License-Identifier: Apache-2.0

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
func (r *Retry) Dial(network, address string) (c net.Conn, err error) {
	for i := 0; i < r.NumRetries; i++ {
		c, err = r.Dialer.Dial(network, address)
		if err == nil {
			return
		}
	}
	return
}

// DialTLS creates a new TLS Dialer using the underlying *net.Dial and uses it to dial a net.Conn, but retries on failure
func (r *Retry) DialTLS(network, address string, config *tls.Config) (c net.Conn, err error) {
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
