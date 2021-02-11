package client

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/loophole-labs/frisbee/internal/protocol"
	"io"
	"net"
	"sync"
	//"strconv"
)

// Example command: go run client.go
func Run() {
	conn, err := net.Dial("tcp", "127.0.0.1:8192")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	bufConn := bufio.NewWriterSize(conn, 512)
	bufRead := bufio.NewReader(conn)

	//cleanup := make(chan bool)

	go func() {
		for {
			_, err := ClientDecode(bufRead)
			if err != nil {
				// Can be caused by server disconnects
				//log.Printf("ClientDecode error, %v\n", err)
				break
			}
			//log.Printf("Operation: %x, Routing %x, Content: %s", response.Operation, response.Routing, string(response.Content))
			//numResp, _ := strconv.Atoi(string(response.Content))
			//if numResp == 100000-1 {
			//	cleanup <- true
			//	break
			//}
		}
	}()

	var wg sync.Mutex
	var wait sync.WaitGroup

	for i := 0; i < 1000000; i++ {
		data := []byte(fmt.Sprintf("%d", i))
		encodedMessage, err := ClientEncode(protocol.MessagePing, data)
		if err != nil {
			panic(err)
		}
		wait.Add(1)
		go func() {
			wg.Lock()
			_, _ = bufConn.Write(encodedMessage)
			wg.Unlock()
			wait.Done()
		}()
	}
	wait.Wait()
	bufConn.Flush()
	//select {
	//case <- cleanup:
	//}
}

// ClientEncode :
func ClientEncode(operation uint16, data []byte) ([]byte, error) {
	return protocol.EncodeV0(operation, 0, data, true)
}

// ClientDecode :
func ClientDecode(rawConn *bufio.Reader) (*protocol.MessageV0, error) {

	encodedHeader := make([]byte, protocol.HeaderLengthV0)
	n, err := io.ReadFull(rawConn, encodedHeader)
	if n != protocol.HeaderLengthV0 {
		return nil, err
	}

	decodedHeader, err := protocol.DecodeV0(encodedHeader, true, true)
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
