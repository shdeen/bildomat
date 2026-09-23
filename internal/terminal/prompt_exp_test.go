package terminal

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
// 1. Failed prompt delivery: When the output writer refuses a prompt and the input file is closed,
//    ConfirmOneWordPrompt and RepromptModel must return ErrOutputFileWrite and io.ErrClosedPipe
//    without ErrProcessReadInput.
// 2. Failed reply reads: With a closed input file and a writable destination, ConfirmOneWordPrompt
//    and RepromptModel must both return ErrProcessReadInput and os.ErrClosed, and their combined
//    diagnostic output must be nonempty.
// 3. Total prompt confirmation behavior: For arbitrary prompts and replies, ConfirmOneWordPrompt
//    must return no error and write a prompt exactly when the input is a terminal and the prompt
//    has one word.

// TestPromptDeliveryFailure verifies invariant #1: Failed prompt delivery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the output writer refuses a prompt and the input file is closed, ConfirmOneWordPrompt and
// RepromptModel must return ErrOutputFileWrite and io.ErrClosedPipe without ErrProcessReadInput.
// They must report no cancellation or reply; RepromptModel must report that it attempted a prompt.
//
// Kind: permanent.
func TestPromptDeliveryFailure(t *testing.T) {
	closedInput, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("💣 input fixture: %v", err)
	}

	if err := closedInput.Close(); err != nil {
		t.Fatalf("💣 close input fixture: %v", err)
	}

	canceled, confirmationErr := ConfirmOneWordPrompt(closedInput, refusedWriter{test: t}, "boat", true, false)

	reply, asked, repromptErr := RepromptModel(closedInput, refusedWriter{test: t}, true, false)
	for _, promptErr := range []error{confirmationErr, repromptErr} {
		if !errors.Is(promptErr, io.ErrClosedPipe) || !errors.Is(promptErr, errs.ErrOutputFileWrite) || errors.Is(promptErr, errs.ErrProcessReadInput) {
			t.Errorf("✗ prompt did not stop at failed delivery: %v", promptErr)
		}
	}

	if canceled || reply != "" || !asked {
		t.Errorf("✗ failed prompt invented a response: canceled=%t, reply=%q, asked=%t", canceled, reply, asked)
	}

	if !t.Failed() {
		t.Log("✓ refused prompts fail before reading input")
	}
}

// TestReplyReadFailure verifies invariant #2: Failed reply reads.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// With a closed input file and a writable destination, ConfirmOneWordPrompt and RepromptModel must
// both return ErrProcessReadInput and os.ErrClosed, and their combined diagnostic output must be
// nonempty. They must report no cancellation or reply; RepromptModel must report that it attempted
// a prompt.
//
// Kind: permanent.
func TestReplyReadFailure(t *testing.T) {
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("💣 reply input: %v", err)
	}

	if err := input.Close(); err != nil {
		t.Fatalf("💣 close reply input: %v", err)
	}

	var diagnostics bytes.Buffer

	canceled, confirmationErr := ConfirmOneWordPrompt(input, &diagnostics, "boat", true, false)

	reply, asked, repromptErr := RepromptModel(input, &diagnostics, true, false)
	for _, readErr := range []error{confirmationErr, repromptErr} {
		if !errors.Is(readErr, errs.ErrProcessReadInput) || !errors.Is(readErr, os.ErrClosed) {
			t.Errorf("✗ reply read lost its operation or cause: %v", readErr)
		}
	}

	if canceled || reply != "" || !asked || diagnostics.Len() == 0 {
		t.Errorf("✗ failed reply read invented an answer or skipped the prompt: %t, %q, %t", canceled, reply, asked)
	}

	if !t.Failed() {
		t.Log("✓ unreadable replies retain their classification and cause")
	}
}

// FuzzSingleWordPromptConfirmation verifies invariant #3: Total prompt confirmation behavior.
//
// What is being tested:
// For arbitrary prompts and replies, ConfirmOneWordPrompt must return no error and write a prompt
// exactly when the input is a terminal and the prompt has one word. It must cancel exactly when it
// asks for confirmation and the trimmed first reply line equals n or no, ignoring case.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSingleWordPromptConfirmation(f *testing.F) {
	for _, seed := range []struct {
		prompt string
		reply  string
		tty    bool
	}{
		{prompt: "hellp", reply: "n\n", tty: true},
		{prompt: "word", reply: "No", tty: true},
		{prompt: "two words", reply: "n\n", tty: true},
		{prompt: "word", reply: "nope\n", tty: true},
		{prompt: "word", reply: "n\nnext input", tty: true},
		{prompt: "word", reply: "no\n", tty: false},
		{prompt: "word", reply: "n\rx\n", tty: true},
		{prompt: "two words", reply: strings.Repeat("x", longTerminalReply), tty: true},
		{prompt: "word", reply: "n\n" + strings.Repeat("x", longTerminalReply), tty: true},
	} {
		f.Add(seed.prompt, seed.reply, seed.tty)
	}

	f.Fuzz(func(t *testing.T, prompt, reply string, tty bool) {
		useStdin(t, reply, tty)

		var (
			canceled   bool
			confirmErr error
		)

		rendered := captureBoth(t, func() {
			canceled, confirmErr = ConfirmOneWordPrompt(os.Stdin, os.Stderr, prompt, IsTerminal(os.Stdin), IsTerminal(os.Stderr))
		})
		if confirmErr != nil {
			t.Fatalf("💣 confirmation failed over an in-memory reply: %v", confirmErr)
		}

		confirmationExpected := tty && len(strings.Fields(prompt)) == 1
		if (rendered != "") != confirmationExpected {
			t.Errorf("✗ prompt %q, tty=%v: rendered=%q, confirmation expected=%v", prompt, tty, rendered, confirmationExpected)
		}

		replyLine, _, _ := strings.Cut(reply, "\n")
		trimmedReply := strings.TrimSpace(replyLine)

		cancelExpected := confirmationExpected && (strings.EqualFold(trimmedReply, "n") || strings.EqualFold(trimmedReply, "no"))
		if canceled != cancelExpected {
			t.Errorf("✗ prompt %q, reply %q, tty=%v: canceled=%v, want %v", prompt, reply, tty, canceled, cancelExpected)
		}

		if !t.Failed() {
			t.Logf("✓ confirmation and cancellation stayed within the terminal one-word contract")
		}
	})
}

// refusedWriter refuses diagnostic delivery while retaining the test context.
type refusedWriter struct{ test testing.TB }

// Write returns the failure used to verify delivery-before-read ordering.
func (writer refusedWriter) Write(_ []byte) (int, error) {
	writer.test.Helper()

	return 0, io.ErrClosedPipe
}

// useStdin supplies test input through a raw pseudo-terminal when tty is true, or through a pipe
// otherwise. It closes the input after sending the text.
func useStdin(test testing.TB, text string, tty bool) {
	test.Helper()

	originalStdin := os.Stdin

	test.Cleanup(func() { os.Stdin = originalStdin })

	if !tty {
		readEnd, writeEnd, err := os.Pipe()
		if err != nil {
			test.Fatalf("💣 stdin pipe: %v", err)
		}

		os.Stdin = readEnd
		written := make(chan struct{})

		// The write runs beside the test so text beyond the pipe's capacity cannot block
		// the setup; closing the read end first unblocks a writer nothing has read, and the
		// wait leaves no writer behind.
		go func() {
			defer close(written)

			_, _ = io.WriteString(writeEnd, text)
			_ = writeEnd.Close()
		}()

		test.Cleanup(func() {
			_ = readEnd.Close()

			<-written
		})

		return
	}

	master, slave := openPTY(test)
	os.Stdin = slave

	// The terminal is in raw mode, so the bytes arrive unchanged. Closing the master ends the
	// input with EOF, but it also discards whatever the program has not read yet, so the writer
	// closes it only once the input is drained, or once the test ends. The writer alone touches
	// the master; the test closes the slave after the writer is done. A write the program never
	// reads blocks once the terminal's input queue is full, so the test ends it with a write
	// deadline before waiting for the writer.
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() { _ = master.Close() }()

		_, _ = io.WriteString(master, text)

		for {
			pending, open := inputPending(test, slave)
			if !open || pending == 0 {
				return
			}

			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			}
		}
	}()

	test.Cleanup(func() {
		_ = master.SetWriteDeadline(time.Now())

		close(stop)
		<-done

		_ = slave.Close()
	})
}

// captureBoth redirects stdout and stderr into one pipe and returns their combined output.
func captureBoth(t *testing.T, fn func()) string {
	t.Helper()

	origOut, origErr := os.Stdout, os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	os.Stdout, os.Stderr = w, w
	done := make(chan string, 1)

	go func() { b, _ := io.ReadAll(r); done <- string(b) }()

	fn()

	os.Stdout, os.Stderr = origOut, origErr
	_ = w.Close()

	return <-done
}

// longTerminalReply is longer than a pseudo-terminal's input queue, so a reply of that length that
// the program never reads blocks its writer.
const longTerminalReply = 1 << 17
