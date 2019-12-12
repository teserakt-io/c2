package services

import (
	"fmt"
)

// ErrClientNotFound is a database error when a requested record cannot be found
// A message containing the missing entity type will be returned to the user
type ErrClientNotFound struct {
}

func (e ErrClientNotFound) Error() string {
	return fmt.Sprintf("could not find client record in database")
}

// ErrTopicNotFound is a database error when a requested record cannot be found
// A message containing the missing entity type will be returned to the user
type ErrTopicNotFound struct {
}

func (e ErrTopicNotFound) Error() string {
	return fmt.Sprintf("could not find topic record in database")
}

// ErrInternal describe an internal error
// A generic error message will be returned to the user
type ErrInternal struct {
}

func (e ErrInternal) Error() string {
	return "a internal error occured. check application logs for details"
}

// ErrValidation describes an error when validating service's input parameters
// its Err will get returned to the user
type ErrValidation struct {
	Err error
}

func (e ErrValidation) Error() string {
	return fmt.Sprintf("validation error: %v", e.Err)
}

// ErrInvalidCryptoMode is returned when trying to execute a method retricted by the current mode
type ErrInvalidCryptoMode struct {
	Err error
}

func (e ErrInvalidCryptoMode) Error() string {
	return fmt.Sprintf("invalid crypto mode: %v", e.Err)
}
