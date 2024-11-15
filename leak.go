package utils

// This code originally comes from https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/leak.go
// which is Apache 2.0 licensed. The following changes are:
// - Removed an ignore
// - edaniels goleak fork

import "github.com/edaniels/goleak"

// FindGoroutineLeaks finds any goroutine leaks after a program is done running. This
// should be used at the end of a main test run or a top-level process run.
func FindGoroutineLeaks(options ...goleak.Option) error {
	optsCopy := make([]goleak.Option, len(options))
	copy(optsCopy, options)
	optsCopy = append(optsCopy,
		goleak.IgnoreTopFunction("go.opencensus.io/stats/view.(*worker).start"),
		goleak.IgnoreTopFunction("github.com/desertbit/timer.timerRoutine"), // gRPC uses this
	)
	return goleak.Find(optsCopy...)
}
