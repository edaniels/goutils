package utils

// This code originally comes from https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/runtime.go
// which is Apache 2.0 licensed. The following changes are:
// - Removed some tests

import (
	"context"
	"os"
	"testing"

	"github.com/edaniels/golog"
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"go.viam.com/test"
)

//nolint:dupl
func TestContextualMain(t *testing.T) {
	var captured []interface{}
	fatalf = func(logger golog.Logger, fmtstr string, args ...interface{}) {
		captured = args
	}
	err1 := errors.New("whoops")
	mainWithArgs := func(ctx context.Context, args []string, logger golog.Logger) error {
		return err1
	}
	logger := golog.NewTestLogger(t)
	ContextualMain(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
	captured = nil
	mainWithArgs = func(ctx context.Context, args []string, logger golog.Logger) error {
		return context.Canceled
	}
	ContextualMain(mainWithArgs, logger)
	test.That(t, captured, test.ShouldBeNil)
	mainWithArgs = func(ctx context.Context, args []string, logger golog.Logger) error {
		return multierr.Combine(context.Canceled, err1)
	}
	ContextualMain(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
}

//nolint:dupl
func TestContextualMainQuit(t *testing.T) {
	var captured []interface{}
	fatalf = func(logger golog.Logger, fmtstr string, args ...interface{}) {
		captured = args
	}
	err1 := errors.New("whoops")
	mainWithArgs := func(ctx context.Context, args []string, logger golog.Logger) error {
		return err1
	}
	logger := golog.NewTestLogger(t)
	ContextualMainQuit(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
	captured = nil
	mainWithArgs = func(ctx context.Context, args []string, logger golog.Logger) error {
		return context.Canceled
	}
	ContextualMainQuit(mainWithArgs, logger)
	test.That(t, captured, test.ShouldBeNil)
	mainWithArgs = func(ctx context.Context, args []string, logger golog.Logger) error {
		return multierr.Combine(context.Canceled, err1)
	}
	ContextualMainQuit(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
}

func TestContextWithQuitSignal(t *testing.T) {
	ctx := context.Background()
	sig := make(chan os.Signal, 1)
	ctx = ContextWithQuitSignal(ctx, sig)
	sig2 := ContextMainQuitSignal(context.Background())
	test.That(t, sig2, test.ShouldBeNil)
	sig2 = ContextMainQuitSignal(ctx)
	test.That(t, sig2, test.ShouldEqual, (<-chan os.Signal)(sig))
}

func TestContextWithReadyFunc(t *testing.T) {
	ctx := context.Background()
	sig := make(chan struct{}, 1)
	ctx = ContextWithReadyFunc(ctx, sig)
	func1 := ContextMainReadyFunc(context.Background())
	func1()
	var ok bool
	select {
	case <-sig:
		ok = true
	default:
	}
	test.That(t, ok, test.ShouldBeFalse)
	func1 = ContextMainReadyFunc(ctx)
	func1()
	select {
	case <-sig:
		ok = true
	default:
	}
	test.That(t, ok, test.ShouldBeTrue)
	func1()
	func1()
	select {
	case <-sig:
		ok = true
	default:
	}
	test.That(t, ok, test.ShouldBeTrue)
}

func TestContextWithIterFunc(t *testing.T) {
	ctx := context.Background()
	sig := make(chan struct{}, 1)
	ctx = ContextWithIterFunc(ctx, func() {
		sig <- struct{}{}
	})
	func1 := ContextMainIterFunc(context.Background())
	func1()
	var ok bool
	select {
	case <-sig:
		ok = true
	default:
	}
	test.That(t, ok, test.ShouldBeFalse)
	func1 = ContextMainIterFunc(ctx)
	go func1()
	<-sig
	go func1()
	go func1()
	<-sig
}
