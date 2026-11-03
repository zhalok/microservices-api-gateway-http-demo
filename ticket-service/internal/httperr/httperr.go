package httperr

import (
	"errors"
	"fmt"
)

// Error is an error carrying the HTTP status to respond with.
type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string { return e.Message }

func New(status int, format string, args ...any) *Error {
	return &Error{Status: status, Message: fmt.Sprintf(format, args...)}
}

// Status returns the HTTP status for err (500 if it isn't an *Error).
func Status(err error) int {
	var he *Error
	if errors.As(err, &he) {
		return he.Status
	}
	return 500
}
