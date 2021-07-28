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
// simply defining your message types and their accompanying logic.
//
// This package provides methods for defining message types and logic, as well as functionality
// for implementing frisbee servers and clients. Useful features like automatic heartbeats and
// automatic reconnections are provided as well.
//
// In depth documentation and examples can be found at https://loopholelabs.io/docs/frisbee
//
// All exported functions and methods are safe to be used concurrently unless
// specified otherwise.
//
// An Echo Example
//
// As a starting point, a very basic echo server:
//
//	package main
//
//	import (
//		"github.com/loophole-labs/frisbee"
//		"github.com/rs/zerolog/log"
//		"os"
//		"os/signal"
//	)
//
//	const PING = uint32(1)
//	const PONG = uint32(2)
//
//	func handlePing(_ *frisbee.Async, incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
//		if incomingMessage.ContentLength > 0 {
//			log.Printf("Server Received Message: %s", incomingContent)
//			outgoingMessage = &frisbee.Message{
//				From:          incomingMessage.From,
//				To:            incomingMessage.To,
//				Id:            incomingMessage.Id,
//				Operation:     PONG,
//				ContentLength: incomingMessage.ContentLength,
//			}
//			outgoingContent = incomingContent
//		}
//
//		return
//	}
//
//	func main() {
//		router := make(frisbee.ServerRouter)
//		router[PING] = handlePing
//		exit := make(chan os.Signal)
//		signal.Notify(exit, os.Interrupt)
//
//		s := frisbee.NewServer(":8192", router)
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
//		"github.com/loophole-labs/frisbee"
//		"github.com/rs/zerolog/log"
//		"os"
//		"os/signal"
//		"time"
//	)
//
//	const PING = uint32(1)
//	const PONG = uint32(2)
//
//	func handlePong(incomingMessage frisbee.Message, incomingContent []byte) (outgoingMessage *frisbee.Message, outgoingContent []byte, action frisbee.Action) {
//		if incomingMessage.ContentLength > 0 {
//			log.Printf("Client Received Message: %s", string(incomingContent))
//		}
//		return
//	}
//
//	func main() {
//		router := make(frisbee.ClientRouter)
//		router[PONG] = handlePong
//		exit := make(chan os.Signal)
//		signal.Notify(exit, os.Interrupt)
//
//		c := frisbee.NewClient("127.0.0.1:8192", router)
//		err := c.ConnectAsync()
//		if err != nil {
//			panic(err)
//		}
//
//		go func() {
//			i := 0
//			for {
//				message := []byte(fmt.Sprintf("ECHO MESSAGE: %d", i))
//				err := c.WriteMessage(&frisbee.Message{
//					To:            0,
//					From:          0,
//					Id:            uint32(i),
//					Operation:     PING,
//					ContentLength: uint64(len(message)),
//				}, &message)
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
// (Examples taken from https://github.com/Loophole-Labs/frisbee-examples/)
//
// This example is a simple echo client/server, where the client will repeatedly send messages to the server,
// and the server will echo them back. Its purpose is to describe the flow of messages from Frisbee Client to Server,
// as well as give an example of how a Frisbee application must be implemented.
package frisbee
