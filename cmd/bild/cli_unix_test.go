//go:build darwin || linux

package main

// Invariants tested:
// This file contains test support only and no test or fuzz function.

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
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

// terminalFixture is a pseudo-terminal the program sees on its streams. The slave is the program's
// side; the master is the test's side, drained into the capture from the moment the terminal opens.
//
// Test class: Core: Helper.
type terminalFixture struct {
	master, slave *os.File
	captured      bytes.Buffer
	drained       chan struct{}
	typing        chan struct{}
	stopTyping    chan struct{}
	endOnce       sync.Once
	closeOnce     sync.Once
}

// openTerminal opens a raw pseudo-terminal and starts draining its master. The terminal closes when
// the test ends.
//
// Test class: Core: Helper.
func openTerminal(test testing.TB) *terminalFixture {
	test.Helper()

	master, slave := openPTY(test)
	fixture := &terminalFixture{master: master, slave: slave, drained: make(chan struct{}), stopTyping: make(chan struct{})}

	go fixture.drain()

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

// ReadOutput ends the program's side of the terminal and returns everything the program wrote,
// without styling sequences.
func (fixture *terminalFixture) ReadOutput(test testing.TB) string {
	test.Helper()
	fixture.endOutput()

	return stripStyling(fixture.captured.String())
}

// Styled reports whether the program wrote text inside a styling sequence: the last sequence
// before the text on its line selects a rendition other than the default.
func (fixture *terminalFixture) Styled(test testing.TB, text string) bool {
	test.Helper()
	fixture.endOutput()

	raw := fixture.captured.String()

	index := strings.Index(raw, text)
	if index < 0 {
		return false
	}

	lineStart := strings.LastIndex(raw[:index], "\n") + 1

	sequences := regexp.MustCompile(`\x1b\[([0-9;]*)m`).FindAllStringSubmatch(raw[lineStart:index], -1)
	if len(sequences) == 0 {
		return false
	}

	return sequences[len(sequences)-1][1] != "0"
}

// Close ends any typing, the program's side, and the drain, then releases the master.
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

	fixture.endOutput()

	_ = fixture.master.Close()
}

// drain copies the master into the capture until the program's side closes.
func (fixture *terminalFixture) drain() {
	defer close(fixture.drained)

	_, _ = io.Copy(&fixture.captured, fixture.master)
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

// endOutput closes the program's side and waits for the drain, once.
func (fixture *terminalFixture) endOutput() {
	fixture.endOnce.Do(fixture.closeSlaveAndWait)
}

// closeSlaveAndWait is endOutput's body, run once.
func (fixture *terminalFixture) closeSlaveAndWait() {
	_ = fixture.slave.Close()
	<-fixture.drained
}
