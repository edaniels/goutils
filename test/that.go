// Derived from Apache 2.0 Eric Daniels https://github.com/viamrobotics/test/tree/f61b7c01c33ed4e8d01ae2f2263e0c34559384d3
package test

import "testing"

func That(tb testing.TB, actual interface{}, assert func(actual interface{}, expected ...interface{}) string, expected ...interface{}) {
	tb.Helper()
	if result := assert(actual, expected...); result != "" {
		tb.Fatal(result)
	}
}
