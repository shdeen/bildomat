//go:build darwin || linux

package terminal

// Invariants tested:
// This file contains test support only and no test or fuzz function.

import (
	"io"
	"os"
	"sync"
	"testing"
	"time"
)

// inputDrainPollInterval is the pause between checks that the program has read the typed reply,
// and inputSettleWindow bounds the wait for a written reply to become visible on the slave: the
// kernel delivers master writes to the slave's queue after the write returns, so a check right
// after the write can find nothing pending although nothing has been read yet.
const (
	inputDrainPollInterval = time.Millisecond
	inputSettleWindow      = 200 * time.Millisecond
)

// terminalFixture is a pseudo-terminal the code under test sees on its streams. The slave is the
// program's side; the master is the test's side.
type terminalFixture struct {
	master, slave *os.File
	typing        chan struct{}
	stopTyping    chan struct{}
	closeOnce     sync.Once
}

// openTerminal opens a raw pseudo-terminal. The terminal closes when the test ends.
func openTerminal(test testing.TB) *terminalFixture {
	test.Helper()

	master, slave := openPTY(test)
	fixture := &terminalFixture{master: master, slave: slave, stopTyping: make(chan struct{})}

	test.Cleanup(fixture.Close)

	return fixture
}

// Input returns the file the program reads as a terminal.
func (fixture *terminalFixture) Input() *os.File { return fixture.slave }

// Output returns the file the program writes as a terminal.
func (fixture *terminalFixture) Output() *os.File { return fixture.slave }

// TypeReply supplies text as typed input and ends the input once the program has read it, or once
// the test ends.
func (fixture *terminalFixture) TypeReply(test testing.TB, text string) {
	test.Helper()

	fixture.typing = make(chan struct{})

	go fixture.typeThenEnd(test, text)
}

// Close ends any typing, then releases both sides of the terminal.
func (fixture *terminalFixture) Close() {
	fixture.closeOnce.Do(fixture.closeAll)
}

// closeAll is Close's body, run once.
func (fixture *terminalFixture) closeAll() {
	if fixture.typing != nil {
		// A write the program never reads blocks once the terminal's input queue is full, so the
		// deadline ends it before the wait.
		_ = fixture.master.SetWriteDeadline(time.Now())

		close(fixture.stopTyping)
		<-fixture.typing
	}

	_ = fixture.slave.Close()
	_ = fixture.master.Close()
}

// typeThenEnd writes the reply, waits until the program has drained it, and closes the master so
// the input ends with EOF. Closing earlier would discard whatever the program has not read yet.
func (fixture *terminalFixture) typeThenEnd(test testing.TB, text string) {
	test.Helper()

	defer close(fixture.typing)
	defer fixture.closeMaster()

	_, _ = io.WriteString(fixture.master, text)

	poll := time.NewTicker(inputDrainPollInterval)
	defer poll.Stop()

	settled := time.Now().Add(inputSettleWindow)
	replySeen := false

	for {
		pending, open := inputPending(test, fixture.slave)
		if !open {
			return
		}

		if pending > 0 {
			replySeen = true
		} else if replySeen || time.Now().After(settled) {
			return
		}

		select {
		case <-fixture.stopTyping:
			return
		case <-poll.C:
		}
	}
}

// closeMaster releases the master; a second close is harmless.
func (fixture *terminalFixture) closeMaster() {
	_ = fixture.master.Close()
}
