package utils

import (
	"errors"
	"strings"

	perrors "github.com/pkg/errors"
)

// ErrorWithStack returns an error with a stacktrace as long as there
// is not one currently attached.
func ErrorWithStack(err error) error {
	type stackTracer interface {
		StackTrace() perrors.StackTrace
	}

	if _, ok := err.(stackTracer); ok {
		return err
	}

	return perrors.WithStack(err)
}

type multiWrappedError interface {
	Unwrap() []error
}

// FilterOutError filters out an error based on the given target. For
// example, if err was context.Canceled and so was the target, this
// would return nil. Furthermore, if err was a multierr containing
// a context.Canceled, it would also be filtered out from a new
// multierr.
// The code in this package originally comes from
// https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/error.go
// which is Apache 2.0 licensed. The following changes are:
// - dont use multierror.
func FilterOutError(err, target error) error {
	if err == nil {
		return nil
	}
	if target == nil {
		return err
	}
	var errs []error
	//nolint:errorlint // we know better?
	switch v := err.(type) {
	case multiWrappedError:
		errs = v.Unwrap()
	default:
		errs = []error{err}
	}
	if len(errs) == 1 {
		if errors.Is(err, target) || strings.Contains(err.Error(), target.Error()) {
			return nil
		}
		return err
	}
	newErrs := make([]error, 0, len(errs))
	for _, e := range errs {
		if FilterOutError(e, target) == nil {
			continue
		}
		newErrs = append(newErrs, e)
	}
	if len(newErrs) == 1 {
		return newErrs[0]
	}
	return errors.Join(newErrs...)
}
