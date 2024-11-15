package utils

// This code originally comes from https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/runtime_test.go
// which is Apache 2.0 licensed. The following changes are:
// - Removed some tests

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/edaniels/goutils/test"
)

//nolint:dupl
func TestContextualMain(t *testing.T) {
	var captured []interface{}
	fatalf = func(_ *zap.SugaredLogger, _ string, args ...interface{}) {
		captured = args
	}
	err1 := errors.New("whoops")
	mainWithArgs := func(_ context.Context, _ []string, _ *zap.SugaredLogger) error {
		return err1
	}
	logger := NewTestLogger(t)
	ContextualMain(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
	captured = nil
	mainWithArgs = func(_ context.Context, _ []string, _ *zap.SugaredLogger) error {
		return context.Canceled
	}
	ContextualMain(mainWithArgs, logger)
	test.That(t, captured, test.ShouldBeNil)
	mainWithArgs = func(_ context.Context, _ []string, _ *zap.SugaredLogger) error {
		return errors.Join(context.Canceled, err1)
	}
	ContextualMain(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
}

//nolint:dupl
func TestContextualMainQuit(t *testing.T) {
	var captured []interface{}
	fatalf = func(_ *zap.SugaredLogger, _ string, args ...interface{}) {
		captured = args
	}
	err1 := errors.New("whoops")
	mainWithArgs := func(_ context.Context, _ []string, _ *zap.SugaredLogger) error {
		return err1
	}
	logger := NewTestLogger(t)
	ContextualMainQuit(mainWithArgs, logger)
	test.That(t, captured, test.ShouldResemble, []interface{}{err1})
	captured = nil
	mainWithArgs = func(_ context.Context, _ []string, _ *zap.SugaredLogger) error {
		return context.Canceled
	}
	ContextualMainQuit(mainWithArgs, logger)
	test.That(t, captured, test.ShouldBeNil)
	mainWithArgs = func(_ context.Context, _ []string, _ *zap.SugaredLogger) error {
		return errors.Join(context.Canceled, err1)
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

func TestPanicCapturingGo(t *testing.T) {
	running := make(chan struct{})
	PanicCapturingGo(func() {
		close(running)
		panic("dead")
	})
	<-running
	time.Sleep(time.Second)
	test.That(t, true, test.ShouldBeTrue)
}

func TestPanicCapturingGoWithCallback(t *testing.T) {
	running := make(chan struct{})
	errCh := make(chan interface{})
	PanicCapturingGoWithCallback(func() {
		close(running)
		panic("dead")
	}, func(err interface{}) {
		errCh <- err
	})
	<-running
	test.That(t, <-errCh, test.ShouldEqual, "dead")
}

func TestSelectContextOrWait(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ok := SelectContextOrWait(ctx, time.Hour)
	test.That(t, ok, test.ShouldBeFalse)

	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	ok = SelectContextOrWait(ctx, time.Hour)
	test.That(t, ok, test.ShouldBeFalse)

	ok = SelectContextOrWait(context.Background(), time.Second)
	test.That(t, ok, test.ShouldBeTrue)
}

func TestSelectContextOrWaitChan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	timer := time.NewTimer(time.Second)
	timer.Stop()
	ok := SelectContextOrWaitChan(ctx, timer.C)
	test.That(t, ok, test.ShouldBeFalse)

	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	ok = SelectContextOrWaitChan(ctx, timer.C)
	test.That(t, ok, test.ShouldBeFalse)

	timer = time.NewTimer(time.Second)
	defer timer.Stop()
	ok = SelectContextOrWaitChan(context.Background(), timer.C)
	test.That(t, ok, test.ShouldBeTrue)
}

func TestSelectContextOrWaitChanVal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	timer := time.NewTimer(time.Second)
	timer.Stop()
	_, ok := SelectContextOrWaitChanVal(ctx, timer.C)
	test.That(t, ok, test.ShouldBeFalse)

	ctx, cancel = context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Second)
		cancel()
	}()
	_, ok = SelectContextOrWaitChanVal(ctx, timer.C)
	test.That(t, ok, test.ShouldBeFalse)

	timer = time.NewTimer(time.Second)
	defer timer.Stop()
	val, ok := SelectContextOrWaitChanVal(context.Background(), timer.C)
	test.That(t, ok, test.ShouldBeTrue)
	test.That(t, val, test.ShouldNotBeZeroValue)
}

func TestManagedGo(t *testing.T) {
	dieCount := 3
	done := make(chan struct{})
	ManagedGo(func() {
		time.Sleep(50 * time.Millisecond)
		if dieCount == 0 {
			return
		}
		dieCount--
		panic(dieCount)
	}, func() {
		close(done)
	})
	<-done
	test.That(t, true, test.ShouldBeTrue)
}

func TestSlowGoroutineWatcher(t *testing.T) {
	logger, observedLogs := NewObservedTestLogger(t)
	ch, cancel := SlowGoroutineWatcher(2*time.Second, "hello", logger)
	cancel()
	<-ch
	test.That(t, observedLogs.All(), test.ShouldHaveLength, 0)

	ch, cancel = SlowGoroutineWatcher(2*time.Second, "hello", logger)
	<-ch
	cancel()
	test.That(t, len(observedLogs.All()), test.ShouldBeGreaterThan, 0)
	test.That(t, observedLogs.All()[0].Message, test.ShouldContainSubstring, "hello")
	test.That(t, observedLogs.All()[0].Message, test.ShouldContainSubstring, "[chan receive]")
	test.That(t, observedLogs.All()[0].Message, test.ShouldContainSubstring, "github.com/edaniels/goutils.TestSlowGoroutineWatcher(")
}

func TestSlowGoroutineWatcherAfterContext(t *testing.T) {
	logger, observedLogs := NewObservedTestLogger(t)
	ch, cancel := SlowGoroutineWatcherAfterContext(context.Background(), 2*time.Second, "hello", logger)
	cancel()
	<-ch
	test.That(t, observedLogs.All(), test.ShouldHaveLength, 0)

	ctx, ctxCancel := context.WithCancel(context.Background())
	ch, cancel = SlowGoroutineWatcherAfterContext(ctx, 2*time.Second, "hello", logger)
	ctxCancel()
	cancel()
	<-ch
	test.That(t, observedLogs.All(), test.ShouldHaveLength, 0)

	ctx, ctxCancel = context.WithCancel(context.Background())
	ch, cancel = SlowGoroutineWatcherAfterContext(ctx, 2*time.Second, "hello", logger)
	ctxCancel()
	<-ch
	cancel()
	test.That(t, len(observedLogs.All()), test.ShouldBeGreaterThan, 0)
	test.That(t, observedLogs.All()[0].Message, test.ShouldContainSubstring, "hello")
	test.That(t, observedLogs.All()[0].Message, test.ShouldContainSubstring, "[chan receive]")
	test.That(t,
		observedLogs.All()[0].Message,
		test.ShouldContainSubstring,
		"github.com/edaniels/goutils.TestSlowGoroutineWatcherAfterContext(")
}
