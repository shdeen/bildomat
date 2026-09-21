//go:build !darwin && !linux

package main

import (
	"os"
	"runtime"
	"testing"
)

// openPTY fails the test: the pseudo-terminal helper is written for darwin
// and linux, and a test that needs a terminal cannot run without one.
//
// Test class: Core: Helper.
func openPTY(test testing.TB) (master, slave *os.File) {
	test.Helper()
	test.Fatalf("💣 no pseudo-terminal helper for %s", runtime.GOOS)

	return nil, nil
}

// inputPending is never reached: openPTY fails the test first.
func inputPending(_ testing.TB, _ *os.File) (int32, bool) {
	return 0, false
}
