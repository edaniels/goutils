package utils

import "github.com/pkg/errors"

// ErrorWithStack returns an error with a stacktrace as long as there
// is not one currently attached.
func ErrorWithStack(err error) error {
	type stackTracer interface {
		StackTrace() errors.StackTrace
	}

	if _, ok := err.(stackTracer); ok {
		return err
	}

	return errors.WithStack(err)
}
