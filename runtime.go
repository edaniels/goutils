// Package utils contains all utility functions that currently have no better home than here.
package utils

// This code originally comes from https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/runtime.go
// which is Apache 2.0 licensed. The following changes are:
// - Remove golog.Logger
// - Use Fatalf from edaniels/golog instead of Fatal

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

	"github.com/edaniels/golog"
	"go.viam.com/utils"
)

// ContextualMain calls a main entry point function with a cancellable
// context via SIGTERM. This should be called once per process so as
// to not clobber the signals from Notify.
func ContextualMain(main func(ctx context.Context, args []string, logger golog.Logger) error, logger golog.Logger) {
	// This will only run on a successful exit due to the fatal error
	// logic in contextualMain.
	defer func() {
		if err := utils.FindGoroutineLeaks(); err != nil {
			fmt.Fprintf(os.Stderr, "goroutine leak(s) detected: %v\n", err)
		}
	}()
	contextualMain(main, false, logger)
}

// ContextualMainQuit is the same as ContextualMain but catches quit signals into the provided
// context accessed via ContextMainQuitSignal.
func ContextualMainQuit(main func(ctx context.Context, args []string, logger golog.Logger) error, logger golog.Logger) {
	contextualMain(main, true, logger)
}

func contextualMain(main func(ctx context.Context, args []string, logger golog.Logger) error, quitSignal bool, logger golog.Logger) {
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
	utils.ManagedGo(func() {
		for {
			if !utils.SelectContextOrWaitChan(ctx, usr1C) {
				return
			}
			buf := make([]byte, 1024)
			for {
				n := runtime.Stack(buf, true)
				if n < len(buf) {
					buf = buf[:n]
					break
				}
				buf = make([]byte, 2*len(buf))
			}
			logger.Warn(string(buf))
		}
	}, signalWatcher.Done)

	readyC := make(chan struct{})
	readyCtx := ContextWithReadyFunc(ctx, readyC)
	if err := utils.FilterOutError(main(readyCtx, os.Args, logger), context.Canceled); err != nil {
		fatalf(logger, "%+v", err)
	}
}

var fatalf = func(logger golog.Logger, fmtstr string, args ...interface{}) {
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
				golog.Global().Errorw("panic while running function", "error", err)
				if callback == nil {
					return
				}
				golog.Global().Infow("waiting a bit to call callback", "wait", waitDur.String())
				time.Sleep(waitDur)
				callback(err)
			}
		}()
		f()
	}()
}
