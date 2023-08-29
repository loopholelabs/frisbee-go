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

// Package pair is used to create a pair of
// established TCP connections for testing
// networking logic
package pair

import (
	"github.com/loopholelabs/testing/conn"
	"net"
	"sync"
)

// New creates a pair of TCP connections that are bound to one another.
//
// It also cleans up the listener after itself so as not to interfere
// with other tests
func New() (net.Conn, net.Conn, error) {
	listener, err := net.Listen(conn.Network, conn.Listen)
	if err != nil {
		return nil, nil, err
	}
	acceptChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		c, err := listener.Accept()
		if err != nil {
			errChan <- err
		} else {
			acceptChan <- c
		}
	}()

	c1, err := net.Dial(listener.Addr().Network(), listener.Addr().String())
	if err != nil {
		return nil, nil, err
	}

	select {
	case err = <-errChan:
		return nil, nil, err
	case c2 := <-acceptChan:
		err = listener.Close()
		if err != nil {
			return nil, nil, err
		}

		wg.Wait()
		close(errChan)
		close(acceptChan)

		return c1, c2, nil
	}
}

// Cleanup simply closes the given TCP connections and returns
// an error if closing was unsuccessful
func Cleanup(c1 net.Conn, c2 net.Conn) error {
	if err := c1.Close(); err != nil {
		return err
	}
	if err := c2.Close(); err != nil {
		return err
	}
	return nil
}
