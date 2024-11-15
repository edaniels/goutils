// Package utils contains all utility functions that currently have no better home than here.
package utils

// The code in this package originally comes from
// https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/runtime.go
// which is Apache 2.0 licensed. The following changes are:
// - Use pure zap

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// ContextualMain calls a main entry point function with a cancellable
// context via SIGTERM. This should be called once per process so as
// to not clobber the signals from Notify.
func ContextualMain(main func(ctx context.Context, args []string, logger *zap.SugaredLogger) error, logger *zap.SugaredLogger) {
	// This will only run on a successful exit due to the fatal error
	// logic in contextualMain.
	defer func() {
		if err := FindGoroutineLeaks(); err != nil {
			fmt.Fprintf(os.Stderr, "goroutine leak(s) detected: %v\n", err)
		}
	}()
	contextualMain(main, false, logger)
}

// ContextualMainQuit is the same as ContextualMain but catches quit signals into the provided
// context accessed via ContextMainQuitSignal.
func ContextualMainQuit(main func(ctx context.Context, args []string, logger *zap.SugaredLogger) error, logger *zap.SugaredLogger) {
	contextualMain(main, true, logger)
}

func contextualMain(main func(
	ctx context.Context,
	args []string,
	logger *zap.SugaredLogger,
) error, quitSignal bool, logger *zap.SugaredLogger,
) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if quitSignal {
		quitC := make(chan os.Signal, 1)
		signal.Notify(quitC, syscall.SIGQUIT)
		ctx = ContextWithQuitSignal(ctx, quitC)
	}
	usr1C := make(chan os.Signal, 1)
	notifySignals(usr1C)

	var signalWatcher sync.WaitGroup
	signalWatcher.Add(1)
	defer signalWatcher.Wait()
	defer stop()
	ManagedGo(func() {
		for {
			if !SelectContextOrWaitChan(ctx, usr1C) {
				return
			}
			//nolint:mnd
			buf := make([]byte, 1024)
			for {
				n := runtime.Stack(buf, true)
				if n < len(buf) {
					buf = buf[:n]
					break
				}
				//nolint:mnd
				buf = make([]byte, 2*len(buf))
			}
			logger.Warn(string(buf))
		}
	}, signalWatcher.Done)

	readyC := make(chan struct{})
	readyCtx := ContextWithReadyFunc(ctx, readyC)
	if err := FilterOutError(main(readyCtx, os.Args, logger), context.Canceled); err != nil {
		fatalf(logger, "%+v", err)
	}
}

var fatalf = func(logger *zap.SugaredLogger, fmtstr string, args ...interface{}) {
	logger.Fatalf(fmtstr, args...)
}

type ctxKey int

const (
	ctxKeyQuitSignaler = ctxKey(iota)
	ctxKeyReadyFunc
	ctxKeyIterFunc
)

// ContextWithQuitSignal attaches a quit signaler to the given context.
func ContextWithQuitSignal(ctx context.Context, c <-chan os.Signal) context.Context {
	return context.WithValue(ctx, ctxKeyQuitSignaler, c)
}

// ContextMainQuitSignal returns a signal channel for quits. It may
// be nil if the value was never set.
func ContextMainQuitSignal(ctx context.Context) <-chan os.Signal {
	signaler := ctx.Value(ctxKeyQuitSignaler)
	if signaler == nil {
		return nil
	}
	return signaler.(<-chan os.Signal)
}

// ContextWithReadyFunc attaches a ready signaler to the given context.
func ContextWithReadyFunc(ctx context.Context, c chan<- struct{}) context.Context {
	closeOnce := sync.Once{}
	return context.WithValue(ctx, ctxKeyReadyFunc, func() {
		closeOnce.Do(func() {
			close(c)
		})
	})
}

// ContextMainReadyFunc returns a function for indicating readiness. This
// is intended for main functions that block forever (e.g. daemons).
func ContextMainReadyFunc(ctx context.Context) func() {
	signaler := ctx.Value(ctxKeyReadyFunc)
	if signaler == nil {
		return func() {}
	}
	return signaler.(func())
}

// ContextWithIterFunc attaches an iteration func to the given context.
func ContextWithIterFunc(ctx context.Context, f func()) context.Context {
	return context.WithValue(ctx, ctxKeyIterFunc, f)
}

// ContextMainIterFunc returns a function for indicating an iteration of the
// program has completed.
func ContextMainIterFunc(ctx context.Context) func() {
	iterFunc := ctx.Value(ctxKeyIterFunc)
	if iterFunc == nil {
		return func() {}
	}
	return iterFunc.(func())
}

// PanicCapturingGo spawns a goroutine to run the given function and captures
// any panic that occurs and logs it.
func PanicCapturingGo(f func()) {
	PanicCapturingGoWithCallback(f, nil)
}

const waitDur = 3 * time.Second

// PanicCapturingGoWithCallback spawns a goroutine to run the given function and captures
// any panic that occurs, logs it, and calls the given callback. The callback can be
// used for restart functionality.
func PanicCapturingGoWithCallback(f func(), callback func(err interface{})) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				debug.PrintStack()
				zap.L().Sugar().Errorw("panic while running function", "error", err)
				if callback == nil {
					return
				}
				zap.L().Sugar().Infow("waiting a bit to call callback", "wait", waitDur.String())
				time.Sleep(waitDur)
				callback(err)
			}
		}()
		f()
	}()
}

// ManagedGo keeps the given function alive in the background until
// it terminates normally.
func ManagedGo(f, onComplete func()) {
	PanicCapturingGoWithCallback(func() {
		defer func() {
			if err := recover(); err == nil && onComplete != nil {
				onComplete()
			} else if err != nil {
				// re-panic
				panic(err)
			}
		}()
		f()
	}, func(_ interface{}) {
		ManagedGo(f, onComplete)
	})
}

// SelectContextOrWait either terminates because the given context is done
// or the given duration elapses. It returns true if the duration elapsed.
func SelectContextOrWait(ctx context.Context, dur time.Duration) bool {
	timer := time.NewTimer(dur)
	defer timer.Stop()
	return SelectContextOrWaitChan(ctx, timer.C)
}

// SelectContextOrWaitChan either terminates because the given context is done
// or the given channel is received on. It returns true if the channel
// was received on.
func SelectContextOrWaitChan[T any](ctx context.Context, c <-chan T) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}
	select {
	case <-ctx.Done():
		return false
	case <-c:
	}
	return true
}

// SelectContextOrWaitChanVal either terminates because the given context is done
// or the given channel has a value on it. It returns true if the channel
// was received on.
func SelectContextOrWaitChanVal[T any](ctx context.Context, c <-chan T) (T, bool) {
	var zero T
	select {
	case <-ctx.Done():
		return zero, false
	default:
	}
	select {
	case <-ctx.Done():
		return zero, false
	case v := <-c:
		return v, true
	}
}

// SlowGoroutineWatcherAfterContext is used to monitor if a goroutine is going "slow".
// It will first wait for the given context to be done and then kick off
// a wait based on 'dur'. If 'dur' elapses before the cancel func is called, then
// currently running goroutines will be dumped. The returned channel can be used to wait for
// this background goroutine to finish.
func SlowGoroutineWatcherAfterContext(
	ctx context.Context,
	dur time.Duration,
	slowMsg string,
	logger *zap.SugaredLogger,
) (<-chan struct{}, func()) {
	return slowGoroutineWatcher(ctx, dur, slowMsg, logger)
}

// SlowGoroutineWatcher is used to monitor if a goroutine is going "slow".
// It will first kick off a wait based on 'dur'. If 'dur' elapses before the cancel func is called,
// then currently running goroutines will be dumped. The returned channel can be used to wait for
// this background goroutine to finish.
func SlowGoroutineWatcher(
	dur time.Duration,
	slowMsg string,
	logger *zap.SugaredLogger,
) (<-chan struct{}, func()) {
	//nolint:staticcheck
	return slowGoroutineWatcher(nil, dur, slowMsg, logger)
}

func slowGoroutineWatcher(
	ctx context.Context,
	dur time.Duration,
	slowMsg string,
	logger *zap.SugaredLogger,
) (<-chan struct{}, func()) {
	slowWatcher := make(chan struct{})
	slowWatcherCtx, slowWatcherCancel := context.WithCancel(context.Background())
	PanicCapturingGo(func() {
		defer close(slowWatcher)
		if ctx != nil {
			select {
			case <-ctx.Done():
			case <-slowWatcherCtx.Done():
				return
			}
		}
		if !SelectContextOrWait(slowWatcherCtx, dur) {
			return
		}

		//nolint:mnd
		buf := make([]byte, 1024)
		for {
			n := runtime.Stack(buf, true)
			if n < len(buf) {
				buf = buf[:n]
				break
			}
			//nolint:mnd
			buf = make([]byte, 2*len(buf))
		}
		logger.Warn(slowMsg, "\n", string(buf))
	})
	return slowWatcher, slowWatcherCancel
}
