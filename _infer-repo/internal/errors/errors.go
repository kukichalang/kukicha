package errors

import (
	"errors"
	"fmt"
)

func New(msg string) error {
	return errors.New(msg)
}

func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}
