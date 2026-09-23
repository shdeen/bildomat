package main

// Invariants tested:
//  1. Results write failure reporting: When plain, styled, or JSON output fails, invocation.finish
//     must retain errs.ErrOutputFileWrite. PrintJSON must return that classification directly.
//  2. Result finalization causes: When writing and closing both fail, invocation.finish must retain
//     both causes and classifications. A second finish call must return nil.
//  3. Generation report causes: invocation.reportGeneration must preserve the supplied generation
//     error together with JSON encoding and output-write failures.
//  4. Failed result diagnostics: When generation and output both fail, invocation.reportGeneration
//     must preserve both causes and report every completed media or sidecar path on diagnostics in
//     all tested modes.
//  5. Late close diagnostics: If the results file is already closed, invocation.finish must return
//     the close cause and write a diagnostic without changing the delivered JSON document.
//     Repeating finish must return nil without another diagnostic.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/output"
)

// TestResultsWriteFailureSurfaces verifies invariant #1: Results write failure reporting.
//
// What is being tested:
// When the results writer rejects a plain report, styled report, or JSON document,
// invocation.finish must return an error matching errs.ErrOutputFileWrite. output.PrintJSON must
// also return that classification directly. A later invocation.finish(nil) must return nil.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestResultsWriteFailureSurfaces(t *testing.T) {
	invocation := newInvocation(os.Stdin, io.Discard, io.Discard)

	var writeErr error

	printPlain := 0
	printStyled := 1
	printJSON := 2

	for printForm := printPlain; printForm <= printJSON; printForm++ {
		resultsPath := filepath.Join(t.TempDir(), "results.txt")
		if err := invocation.openResults(resultsPath); err != nil {
			t.Fatalf("💣 openResults failed: %v", err)
		}

		invocation.results = failingWriter{test: t}

		switch printForm {
		case printPlain:
			writeErr = output.PrintSavedFile(invocation.results, output.SavedFile{Path: "/tmp/pic.png", Bytes: 12}, false)
		case printStyled:
			writeErr = output.PrintSavedFile(invocation.results, output.SavedFile{Path: "/tmp/pic.png", Bytes: 12}, true)
		case printJSON:
			writeErr = output.PrintJSON(invocation.results, map[string]string{"status": "failed"})
			if err := writeErr; !errors.Is(err, errs.ErrOutputFileWrite) {
				t.Errorf("✗ form %d: PrintJSON returned %v for a refused write, want the classified write failure", printForm, err)
			}
		}

		restoreErr := invocation.finish(writeErr)
		if !errors.Is(restoreErr, errs.ErrOutputFileWrite) {
			t.Errorf("✗ form %d: finalization = %v, want the classified write failure", printForm, restoreErr)
		}
	}

	if invocation.finish(nil) != nil {
		t.Errorf("✗ a repeated restore still carries a failure")
	}

	if !t.Failed() {
		t.Logf("✓ refused results-file writes surface at the restore for all three forms")
	}
}

// TestResultWriteAndCloseCauses verifies invariant #2: Result finalization causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a writer that returns io.ErrClosedPipe and a results file that is already closed,
// invocation.finish must return an error matching both original causes and errs.ErrOutputFileWrite
// and errs.ErrOutputFileClose. A second invocation.finish(nil) must return nil.
func TestResultWriteAndCloseCauses(t *testing.T) {
	invocation := newInvocation(os.Stdin, io.Discard, io.Discard)

	var writeErr error

	resultsPath := filepath.Join(t.TempDir(), "results.txt")
	if err := invocation.openResults(resultsPath); err != nil {
		t.Fatalf("💣 open result fixture: %v", err)
	}

	invocation.results = failingWriter{test: t}

	writeErr = output.PrintSavedFile(invocation.results, output.SavedFile{Path: "/tmp/boat.png", Bytes: 12}, false)
	if err := invocation.file.Close(); err != nil {
		t.Fatalf("💣 close result fixture: %v", err)
	}

	err := invocation.finish(writeErr)
	for _, cause := range []error{io.ErrClosedPipe, os.ErrClosed, errs.ErrOutputFileWrite, errs.ErrOutputFileClose} {
		if !errors.Is(err, cause) {
			t.Errorf("✗ missing cause %v in %v", cause, err)
		}
	}

	if repeatErr := invocation.finish(nil); repeatErr != nil {
		t.Errorf("✗ repeated closure returns %v", repeatErr)
	}

	if !t.Failed() {
		t.Log("✓ write and close failures survive finalization")
	}
}

// TestGenerationReportCauses verifies invariant #3: Generation report causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given errs.ErrResponseNoData, a channel value that JSON cannot encode, and a writer that returns
// io.ErrClosedPipe, invocation.reportGeneration must return an error matching the generation cause,
// errs.ErrJSONEncode, errs.ErrOutputFileWrite, and io.ErrClosedPipe.
func TestGenerationReportCauses(t *testing.T) {
	invocation := newInvocation(os.Stdin, io.Discard, io.Discard)
	invocation.results = failingWriter{test: t}
	invocation.jsonOutput = true
	outcome := output.NewGenerationOutcome(time.Now(), "a boat", map[string]any{"invalid": make(chan int)})
	invocation.outcome = outcome

	err := invocation.reportGeneration(time.Now(), "provider", "model", errs.ErrResponseNoData)
	for _, cause := range []error{errs.ErrResponseNoData, errs.ErrJSONEncode, errs.ErrOutputFileWrite, io.ErrClosedPipe} {
		if !errors.Is(err, cause) {
			t.Errorf("✗ missing cause %v in %v", cause, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ generation and reporting failures retain every cause")
	}
}

// TestFailedResultDiagnostics verifies invariant #4: Failed result diagnostics.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When generation returns errs.ErrResponseNoData and the results writer returns io.ErrClosedPipe,
// invocation.reportGeneration must preserve both causes and errs.ErrOutputFileWrite. In text, JSON,
// filename, and combined filename/JSON modes, diagnostics must identify the write failure and both
// completed artifact paths, including the Markdown sidecar.
func TestFailedResultDiagnostics(t *testing.T) {
	for _, mode := range []string{"text", "json", "filename", "filename-json"} {
		t.Run(mode, func(t *testing.T) {
			var diagnostics bytes.Buffer

			invocation := newInvocation(os.Stdin, failingWriter{test: t}, &diagnostics)
			invocation.jsonOutput = strings.Contains(mode, "json")
			invocation.printFilename = strings.Contains(mode, "filename")
			invocation.outcome = output.NewGenerationOutcome(time.Now(), "a boat", nil)
			invocation.outcome.Artifacts = []output.SavedFile{{Path: "/tmp/boat.png", Bytes: 12}, {Path: "/tmp/boat.md", Bytes: 24}}

			err := invocation.reportGeneration(time.Now(), "provider", "model", errs.ErrResponseNoData)
			for _, cause := range []error{errs.ErrResponseNoData, errs.ErrOutputFileWrite, io.ErrClosedPipe} {
				if !errors.Is(err, cause) {
					t.Errorf("✗ missing returned cause %v: %v", cause, err)
				}
			}

			deliveryText := fmt.Sprintf(output.OutputWriteFailed, "")
			if !strings.Contains(diagnostics.String(), deliveryText) {
				t.Errorf("✗ failed delivery has no diagnostic: %q", diagnostics.String())
			}

			for _, savedFile := range invocation.outcome.Artifacts {
				if !strings.Contains(diagnostics.String(), savedFile.Path) {
					t.Errorf("✗ saved path %q missing from diagnostics: %q", savedFile.Path, diagnostics.String())
				}
			}

			if !t.Failed() {
				t.Log("✓ primary and delivery failures preserve causes and saved paths")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every output mode identifies a failed destination alongside partial success")
	}
}

// TestLateResultClose verifies invariant #5: Late close diagnostics.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// After invocation.reportGeneration writes a completed JSON document, closing its file before
// invocation.finish must make finish return os.ErrClosed and errs.ErrOutputFileClose and write a
// diagnostic. The file must retain exactly the same bytes and decode to a completed outcome. A
// second finish call must return nil without adding diagnostics.
func TestLateResultClose(t *testing.T) {
	var diagnostics bytes.Buffer

	invocation := newInvocation(os.Stdin, io.Discard, &diagnostics)
	invocation.jsonOutput = true

	destination := filepath.Join(t.TempDir(), "results.json")
	if err := invocation.openResults(destination); err != nil {
		t.Fatalf("💣 results fixture: %v", err)
	}

	invocation.outcome = output.NewGenerationOutcome(time.Now(), "a boat", nil)
	if err := invocation.reportGeneration(time.Now(), "provider", "model", nil); err != nil {
		t.Fatalf("💣 initial document: %v", err)
	}

	if err := invocation.file.Close(); err != nil {
		t.Fatalf("💣 closed-file fixture: %v", err)
	}
	// #nosec G304 -- destination is this test's result file.
	before, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("💣 read initial document: %v", err)
	}

	closeErr := invocation.finish(nil)
	if !errors.Is(closeErr, os.ErrClosed) || !errors.Is(closeErr, errs.ErrOutputFileClose) {
		t.Errorf("✗ close causes lost: %v", closeErr)
	}
	// #nosec G304 -- destination is this test's result file.
	after, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("💣 reread completed document: %v", err)
	}

	var document output.GenerationOutcome
	if !bytes.Equal(before, after) || json.Unmarshal(after, &document) != nil || document.Status != "completed" {
		t.Errorf("✗ late close changed the delivered document: %s", after)
	}

	if diagnostics.Len() == 0 {
		t.Error("✗ close failure has no diagnostic")
	}

	reportedLength := diagnostics.Len()
	if err := invocation.finish(nil); err != nil || diagnostics.Len() != reportedLength {
		t.Errorf("✗ repeated close changed failure or diagnostics: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a late close failure is reported without rewriting JSON")
	}
}

// failingWriter rejects writes with io.ErrClosedPipe.
//   - test: the owning test context.
type failingWriter struct{ test testing.TB }

// Write returns zero bytes written and io.ErrClosedPipe.
func (writer failingWriter) Write([]byte) (int, error) {
	writer.test.Helper()

	return 0, io.ErrClosedPipe
}
