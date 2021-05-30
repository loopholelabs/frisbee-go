package frisbee

func handleHeartbeat(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
	outgoingMessage = &Message{}
	return
}

//func (c *Client) heartbeat() (time.Duration, error) {
//for {
//	select {
//	case <-time.After(s.config.KeepAliveInterval):
//		_, err := s.Ping()
//		if err != nil {
//			if err != ErrSessionShutdown {
//				s.logger.Printf("[ERR] yamux: keepalive failed: %v", err)
//				s.exitErr(ErrKeepAliveTimeout)
//			}
//			return
//		}
//	case <-s.shutdownCh:
//		return
//	}
//}
//
//ch := make(chan struct{})
//
//// Get a new ping id, mark as pending
//s.pingLock.Lock()
//id := s.pingID
//s.pingID++
//s.pings[id] = ch
//s.pingLock.Unlock()
//
//// Send the ping request
//hdr := header(make([]byte, headerSize))
//hdr.encode(typePing, flagSYN, 0, id)
//if err := s.waitForSend(hdr, nil); err != nil {
//	return 0, err
//}
//
//// Wait for a response
//start := time.Now()
//select {
//case <-ch:
//case <-time.After(s.config.ConnectionWriteTimeout):
//	s.pingLock.Lock()
//	delete(s.pings, id) // Ignore it if a response comes later.
//	s.pingLock.Unlock()
//	return 0, ErrTimeout
//case <-s.shutdownCh:
//	return 0, ErrSessionShutdown
//}
//
//// Compute the RTT
//return time.Now().Sub(start), nil

//}
