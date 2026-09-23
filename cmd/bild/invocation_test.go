package main

// Invariants tested:
//  1. Actual stream capabilities: newInvocation must derive interaction and styling capabilities
//     from the supplied streams. Interaction requires terminal input and diagnostics; each output's
//     styling follows its own terminal status.
//  2. Results file: invocation.openResults must create missing parents and receive run-detail
//     output through invocation.results. Reopening the file must truncate its contents.
//  3. JSON results file: After invocation.openResults opens the file, output.PrintJSON must write
//     the supplied completed status into it, and invocation.finish must succeed.
//  4. Final adjustment notices: invocation.printAdjustments followed by invocation.reportGeneration
//     must emit each initial adjustment, later adjustment, and file notice once, in order, on
//     diagnostics alone.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/output"
)

// TestInvocationTerminalStreams verifies invariant #1: Actual stream capabilities.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Given terminal, file, pipe, or /dev/null streams, newInvocation must enable interaction only when
// both input and diagnostics are terminals. It must set styled and diagnosticStyled independently
// from the corresponding output stream.
func TestInvocationTerminalStreams(t *testing.T) {
	master, slave := openPTY(t)
	t.Cleanup(func() { _ = master.Close(); _ = slave.Close() })

	nullStream, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("💣 null stream: %v", err)
	}

	t.Cleanup(func() { _ = nullStream.Close() })

	regular, err := os.CreateTemp(t.TempDir(), "stream")
	if err != nil {
		t.Fatalf("💣 regular stream: %v", err)
	}

	t.Cleanup(func() { _ = regular.Close() })

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 pipe streams: %v", err)
	}

	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })

	for _, streams := range []struct {
		label                                 string
		input                                 *os.File
		stdout, stderr                        io.Writer
		interactive, styled, diagnosticStyled bool
	}{
		{label: "null", input: nullStream, stdout: nullStream, stderr: nullStream},
		{label: "file", input: regular, stdout: regular, stderr: regular},
		{label: "pipe", input: reader, stdout: writer, stderr: writer},
		{label: "input terminal, redirected diagnostics", input: slave, stdout: writer, stderr: regular},
		{label: "input and diagnostic terminals", input: slave, stdout: writer, stderr: slave, interactive: true, diagnosticStyled: true},
		{label: "output terminal", input: nullStream, stdout: slave, stderr: writer, styled: true},
		{label: "diagnostic terminal", input: regular, stdout: writer, stderr: slave, diagnosticStyled: true},
	} {
		invocation := newInvocation(streams.input, streams.stdout, streams.stderr)
		if invocation.interactive != streams.interactive || invocation.styled != streams.styled || invocation.diagnosticStyled != streams.diagnosticStyled {
			t.Errorf("✗ %s: input capability %t, output capability %t, diagnostic capability %t", streams.label, invocation.interactive, invocation.styled, invocation.diagnosticStyled)
		}
	}

	if !t.Failed() {
		t.Log("✓ each terminal choice follows the actual selected stream")
	}
}

// TestSaveResults verifies invariant #2: Results file.
//
// What is being tested:
// Given a results path with missing parent directories, invocation.openResults must open it
// successfully. After output.PrintRunDetails writes a provider and model and invocation.finish
// closes the file, the file must contain both headers. Reopening and closing the same path without
// writing must leave an empty file.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSaveResults(t *testing.T) {
	invocation := newInvocation(os.Stdin, io.Discard, io.Discard)

	var writeErr error

	nestedPath := filepath.Join(t.TempDir(), "missing", "parents", "results.txt")

	if err := invocation.openResults(nestedPath); err != nil {
		t.Fatalf("💣 the nested results path failed to open: %v", err)
	}

	writeErr = output.PrintRunDetails(invocation.results, "Fixture", "Fixture Image Model")

	if err := invocation.finish(writeErr); err != nil {
		t.Fatalf("💣 restoring the results destination failed: %v", err)
	}

	// #nosec G304 -- nestedPath is built from this test's own temporary directory.
	content, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("💣 reading the results file failed: %v", err)
	}

	for _, requiredText := range []string{fmt.Sprintf(output.ModelHeader, "Fixture Image Model"), fmt.Sprintf(output.ProviderHeader, "Fixture")} {
		if !strings.Contains(string(content), requiredText) {
			t.Errorf("✗ the results file lacks %q:\n%s", requiredText, content)
		}
	}

	if err := invocation.openResults(nestedPath); err != nil {
		t.Fatalf("💣 reopening the existing results path failed: %v", err)
	}

	if err := invocation.finish(writeErr); err != nil {
		t.Fatalf("💣 restoring the reopened destination failed: %v", err)
	}

	// #nosec G304 -- nestedPath is built from this test's own temporary directory.
	content, err = os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("💣 rereading the results file failed: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("✗ the reopened results file holds %q, want truncation to empty", content)
	}

	if !t.Failed() {
		t.Logf("✓ the results file opens through missing parents, receives the output, and truncates")
	}
}

// TestSaveResultsJSONDocument verifies invariant #3: JSON results file.
//
// What is being tested:
// After invocation.openResults opens results.json, output.PrintJSON must successfully write the
// supplied status map through invocation.results. invocation.finish must succeed, and results.json
// must contain the JSON field "status": "completed".
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSaveResultsJSONDocument(t *testing.T) {
	invocation := newInvocation(os.Stdin, io.Discard, io.Discard)

	var writeErr error

	resultsPath := filepath.Join(t.TempDir(), "results.json")

	if err := invocation.openResults(resultsPath); err != nil {
		t.Fatalf("💣 openResults failed: %v", err)
	}

	if err := output.PrintJSON(invocation.results, map[string]string{"status": "completed"}); err != nil {
		t.Errorf("✗ PrintJSON to the results file failed: %v", err)
	}

	if err := invocation.finish(writeErr); err != nil {
		t.Fatalf("💣 restoring the results destination failed: %v", err)
	}

	// #nosec G304 -- resultsPath is built from this test's own temporary directory.
	content, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("💣 reading the results file failed: %v", err)
	}

	if !strings.Contains(string(content), `"status": "completed"`) {
		t.Errorf("✗ the JSON document did not reach the results file: %q", content)
	}

	if !t.Failed() {
		t.Logf("✓ the JSON document lands in the results file")
	}
}

// TestFinalAdjustmentNotices verifies invariant #4: Final adjustment notices.
//
// What is being tested:
// After invocation.printAdjustments reports the initial aspect adjustment,
// invocation.reportGeneration must write only the later duration adjustment and output-path notice
// to diagnostics. The combined diagnostics must contain each notice exactly once in that order, and
// the results stream must remain empty.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFinalAdjustmentNotices(t *testing.T) {
	var results, diagnostics bytes.Buffer

	invocation := newInvocation(os.Stdin, &results, &diagnostics)
	initialNotice := fmt.Sprintf(output.FlagAdjusted, "Aspect", "'16:9'")
	lateNotice := fmt.Sprintf(output.FlagAdjusted, "Duration", "8")
	fileNotice := fmt.Sprintf(output.FlagAdjusted, output.OutPathDisplayName, "'.png'")

	invocation.outcome = &output.GenerationOutcome{Adjustments: []output.Adjustment{{Notice: initialNotice}}}
	if err := invocation.printAdjustments(); err != nil {
		t.Errorf("✗ initial notice delivery: %v", err)
	}

	invocation.outcome.Adjustments = append(invocation.outcome.Adjustments, output.Adjustment{Notice: lateNotice})

	invocation.outcome.Notices = []string{fileNotice}
	if err := invocation.reportGeneration(time.Now(), "provider", "model", nil); err != nil {
		t.Errorf("✗ final report: %v", err)
	}

	expectedNotices := strings.Join([]string{initialNotice, lateNotice, fileNotice}, "\n") + "\n"
	if diagnostics.String() != expectedNotices || results.Len() != 0 {
		t.Errorf("✗ notice order or destination: stderr %q, stdout %q", diagnostics.String(), results.String())
	}

	if !t.Failed() {
		t.Log("✓ final reporting delivers only remaining adjustments and file notices")
	}
}
