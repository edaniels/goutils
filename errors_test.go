package utils

import (
	"errors"
	"testing"

	"github.com/edaniels/goutils/test"
)

// The code in this package originally comes from
// https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/error_test.go
// which is Apache 2.0 licensed. The following changes are:
// - dont use multierror.
func TestFilterOutError(t *testing.T) {
	test.That(t, FilterOutError(nil, nil), test.ShouldBeNil)
	err1 := errors.New("error1")
	test.That(t, FilterOutError(err1, nil), test.ShouldEqual, err1)
	test.That(t, FilterOutError(err1, err1), test.ShouldBeNil)
	err2 := errors.New("error2")
	test.That(t, FilterOutError(err1, err2), test.ShouldEqual, err1)
	err3 := errors.New("error") // substring
	test.That(t, FilterOutError(err1, err3), test.ShouldBeNil)
	err4 := errors.New("error4")
	errM := errors.Join(err1, err2, err4)
	test.That(t, FilterOutError(errM, err2), test.ShouldResemble, errors.Join(err1, err4))
	test.That(t, FilterOutError(errM, err3), test.ShouldBeNil)
}
