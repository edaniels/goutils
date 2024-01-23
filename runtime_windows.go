package utils

import "os"

// This code originally comes from https://github.com/viamrobotics/goutils/blob/fadaa66af715d712feea4e3637cecd12ed4b742b/runtime.go
// which is Apache 2.0 licensed. The following changes are:

func notifySignals(channel chan os.Signal) {
	println("skipping notifySignals on windows platform")
}
