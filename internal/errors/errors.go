package errors

import "github.com/pkg/errors"

type ErrorContext string

func WithContext(err error, context ErrorContext) error {
	return errors.Wrap(err, string(context))
}

func New(reason string) error {
	return errors.New(reason)
}
