package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/loophole-labs/frisbee/internal/protocol"
)

// Example command: go run client.go
func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8192")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	go func() {
		for {

			response, err := ClientDecode(conn)
			if err != nil {
				// Can be caused by server disconnects
				log.Printf("ClientDecode error, %v\n", err)
			}

			log.Printf("Operation: %x, Routing %x, Content: %s", response.Operation, response.Routing, string(response.Content))

		}
	}()

	data := []byte("hello")
	encodedMessage, err := ClientEncode(protocol.MessagePing, data)
	if err != nil {
		panic(err)
	}
	conn.Write(encodedMessage)

	data = []byte("world")
	encodedMessage, err = ClientEncode(protocol.MessagePing, data)
	if err != nil {
		panic(err)
	}
	conn.Write(encodedMessage)

	select {}
}

// ClientEncode :
func ClientEncode(operation uint16, data []byte) ([]byte, error) {
	return protocol.EncodeV0(operation, 0, data, false)
}

// ClientDecode :
func ClientDecode(rawConn net.Conn) (*protocol.MessageV0, error) {

	encodedHeader := make([]byte, protocol.HeaderLengthV0)
	n, err := io.ReadFull(rawConn, encodedHeader)
	if n != protocol.HeaderLengthV0 {
		return nil, err
	}

	decodedHeader, err := protocol.DecodeV0(encodedHeader, false, true)
	if err != nil {
		return nil, err
	}
	if decodedHeader.ContentLength < 1 {
		return &decodedHeader, nil
	}

	content := make([]byte, decodedHeader.ContentLength)
	contentLength, err := io.ReadFull(rawConn, content)
	if uint32(contentLength) != decodedHeader.ContentLength {
		s := fmt.Sprintf("read data error, %v", err)
		return nil, errors.New(s)
	}

	decodedHeader.Content = content

	return &decodedHeader, nil
}
