//go:build !darwin && !linux && !windows

package terminal

// Invariants tested:
// This file contains test support only and no test or fuzz function.

import (
	"os"
	"runtime"
	"testing"
)

// terminalFixture has no implementation on this platform; the methods exist so the terminal tests
// compile, and openTerminal fails before any of them runs.
type terminalFixture struct{}

// openTerminal fails the test because this platform has no terminal fixture: the fixtures are
// written for darwin, linux, and windows, the release targets, and a test that needs a terminal
// cannot run without one.
func openTerminal(test testing.TB) *terminalFixture {
	test.Helper()
	test.Fatalf("💣 no terminal fixture for %s", runtime.GOOS)

	return nil
}

// Input is never reached: openTerminal fails the test first.
func (fixture *terminalFixture) Input() *os.File { return nil }

// Output is never reached: openTerminal fails the test first.
func (fixture *terminalFixture) Output() *os.File { return nil }

// TypeReply is never reached: openTerminal fails the test first.
func (fixture *terminalFixture) TypeReply(_ testing.TB, _ string) {}
