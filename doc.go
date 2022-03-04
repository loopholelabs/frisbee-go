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

// Package frisbee is the core package for using the frisbee messaging framework. The frisbee framework
// is a messaging framework designed around the aspect of "bring your own protocol", and can be used by
// simply defining your packet types and their accompanying logic.
//
// This package provides methods for defining packet types and logic, as well as functionality
// for implementing frisbee servers and clients. Useful features like automatic heartbeats and
// automatic reconnections are provided as well.
//
// In depth documentation and examples can be found at https://loopholelabs.io/docs/frisbee
//
//
// An Echo Example
//
// As a starting point, a very basic echo server:
//
//	package main
//
//	import (
//		"github.com/loopholelabs/frisbee"
//      "github.com/loopholelabs/packet"
//		"github.com/rs/zerolog/log"
//		"os"
//		"os/signal"
//	)
//
//	const PING = uint16(10)
//	const PONG = uint16(11)
//
//	func handlePing(_ *frisbee.Async, incoming *packet.Packet) (outgoing *packet.Packet, action frisbee.Action) {
//		if incoming.Metadata.ContentLength > 0 {
//			log.Printf("Server Received Metadata: %s\n", incoming.Content)
//          incoming.Metadata.Operation = PONG
//			outgoing = incoming
//		}
//
//		return
//	}
//
//	func main() {
//		handlerTable := make(frisbee.ServerRouter)
//		handlerTable[PING] = handlePing
//		exit := make(chan os.Signal)
//		signal.Notify(exit, os.Interrupt)
//
//		s := frisbee.NewServer(":8192", handlerTable, 0)
//		err := s.Start()
//		if err != nil {
//			panic(err)
//		}
//
//		<-exit
//		err = s.Shutdown()
//		if err != nil {
//			panic(err)
//		}
//	}
//
// And an accompanying echo client:
//	package main
//
//	import (
//		"fmt"
//		"github.com/loopholelabs/frisbee"
//		"github.com/loopholelabs/packet"
//		"github.com/rs/zerolog/log"
//		"os"
//		"os/signal"
//		"time"
//	)
//
//	const PING = uint16(10)
//	const PONG = uint16(11)
//
//	func handlePong(incoming *packet.Packet) (outgoing *packet.Packet, action frisbee.Action) {
//		if incoming.Metadata.ContentLength > 0 {
//			log.Printf("Client Received Metadata: %s\n", incoming.Content)
//		}
//		return
//	}
//
//	func main() {
//		handlerTable := make(frisbee.ClientRouter)
//		handlerTable[PONG] = handlePong
//		exit := make(chan os.Signal)
//		signal.Notify(exit, os.Interrupt)
//
//		c, err := frisbee.NewClient("127.0.0.1:8192", handlerTable)
//		if err != nil {
//			panic(err)
//		}
//		err = c.ConnectAsync()
//		if err != nil {
//			panic(err)
//		}
//
//		go func() {
//			i := 0
//			p := packet.Get()
//			p.Metadata.Operation = PING
//			for {
//				p.Write([]byte(fmt.Sprintf("ECHO MESSAGE: %d", i)))
//				p.Metadata.ContentLength = uint32(len(p.Content))
//				err := c.WritePacket(p)
//				if err != nil {
//					panic(err)
//				}
//				i++
//				time.Sleep(time.Second)
//			}
//		}()
//
//		<-exit
//		err = c.Close()
//		if err != nil {
//			panic(err)
//		}
//	}
//
// (Examples taken from https://github.com/loopholelabs/frisbee-examples/)
//
// This example is a simple echo client/server, where the client will repeatedly send packets to the server,
// and the server will echo them back. Its purpose is to describe the flow of packets from Frisbee Client to Server,
// as well as give an example of how a Frisbee application must be implemented.
package frisbee
