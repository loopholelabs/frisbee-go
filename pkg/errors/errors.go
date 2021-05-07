package errors

import "github.com/pkg/errors"

var (
	InvalidContentLength     = errors.New("invalid content length")
	ConnectionClosed         = errors.New("connection closed")
	UnexpectedEOF            = errors.New("unexpected EOF")
	InvalidBufferContents    = errors.New("invalid buffer contents")
	InvalidBufferLength      = errors.New("invalid buffer length")
	ConnectionNotInitialized = errors.New("connection not initialized")
)

func NewDialError(err error) error {
	return errors.Wrap(err, "error while dialing connection")
}

func NewCloseError(err error) error {
	return errors.Wrap(err, "error while closing connection")
}

func NewShortWriteError(err error) error {
	return errors.Wrap(err, "short write to write buffer")
}

func NewPushError(err error) error {
	return errors.Wrap(err, "error while pushing packet to message queue")
}

func NewPopError(err error) error {
	return errors.Wrap(err, "unable to pop packet from message queue")
}

func NewAcceptError(err error) error {
	return errors.Wrap(err, "unable to accept connections")
}
