package frisbee

import "time"

var heartbeatReceived = make(chan struct{}, 1)
var heartbeatMessageType = uint32(0)

func handleHeartbeatClient(_ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
	heartbeatReceived <- struct{}{}
	outgoingMessage = &Message{
		Operation: heartbeatMessageType,
	}
	return
}

func handleHeartbeatServer(c *Conn, _ Message, _ []byte) (outgoingMessage *Message, outgoingContent []byte, action Action) {
	c.Logger().Printf("GOT THE HEARTBEAT AT THE SERVER!")
	heartbeatReceived <- struct{}{}
	outgoingMessage = &Message{
		Operation: heartbeatMessageType,
	}
	return
}

func (c *Client) heartbeat() {
	for {
		<-time.After(c.options.Heartbeat)
		if c.writeQueue() == 0 {
			c.Logger().Debug().Msgf("Sending heartbeat")
			err := c.Write(&Message{
				Operation: heartbeatMessageType,
			}, nil)
			if err != nil {
				c.Logger().Error().Err(err).Msg("")
				return
			}
			start := time.Now()
			c.Logger().Debug().Msgf("Heartbeat sent at %s", start)
			<-heartbeatReceived
			c.Logger().Debug().Msgf("Heartbeat Received with RTT: %d", time.Now().Sub(start))
		} else {
			c.Logger().Debug().Msgf("Skipping heartbeat because writeQueue > 0")
		}
	}
}
