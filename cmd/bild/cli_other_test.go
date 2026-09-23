//go:build !darwin && !linux

package main

// Invariants tested:
// This file contains test support only and no test or fuzz function.

import (
	"os"
	"runtime"
	"testing"
)

// openPTY fails the test because this platform has no pseudo-terminal helper.
//
// Test class: Core: Helper.
func openPTY(test testing.TB) (master, slave *os.File) {
	test.Helper()
	test.Fatalf("💣 no pseudo-terminal helper for %s", runtime.GOOS)

	return nil, nil
}

// inputPending returns zero and false on platforms without a pseudo-terminal helper.
//
// Test class: Core: Helper.
func inputPending(_ testing.TB, _ *os.File) (int32, bool) {
	return 0, false
}
