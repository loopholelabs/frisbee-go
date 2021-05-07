package errors

import "github.com/pkg/errors"

const (
	Dial     = "error while dialing connection"
	Closing  = "error while closing connection"
	Write    = "error while writing to buffer"
	Push     = "error while pushing packet to message queue"
	Pop      = "error while popping packet from message queue"
	Accept   = "error while accepting connections"
	Encoding = "error while encoding message"
	Decoding = "error while decoding message"
)

var (
	InvalidContentLength     = errors.New("invalid content length")
	ConnectionClosed         = errors.New("connection closed")
	UnexpectedEOF            = errors.New("unexpected EOF")
	InvalidBufferContents    = errors.New("invalid buffer contents")
	InvalidBufferLength      = errors.New("invalid buffer length")
	ConnectionNotInitialized = errors.New("connection not initialized")
	RingerBufferClosed       = errors.New("ring buffer is closed")
	InvalidMessageVersion    = errors.New("invalid message version")
)

func NewDialError(err error) error {
	return errors.Wrap(err, Dial)
}

func NewCloseError(err error) error {
	return errors.Wrap(err, Closing)
}

func NewWriteError(err error) error {
	return errors.Wrap(err, Write)
}

func NewPushError(err error) error {
	return errors.Wrap(err, Push)
}

func NewPopError(err error) error {
	return errors.Wrap(err, Pop)
}

func NewAcceptError(err error) error {
	return errors.Wrap(err, Accept)
}

func NewEncodeError(err error) error {
	return errors.Wrap(err, Encoding)
}

func NewDecodeError(err error) error {
	return errors.Wrap(err, Decoding)
}
