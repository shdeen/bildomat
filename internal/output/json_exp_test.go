package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. JSON failure causes: Given a document containing a channel, PrintJSON must return an error
//     matching ErrJSONEncode and json.UnsupportedTypeError. A writable destination must receive one
//     failed-status document with one error. If writing that fallback fails, the returned error
//     must also match io.ErrClosedPipe and ErrOutputFileWrite.
//  2. Complete JSON delivery: When the destination accepts fewer bytes than offered but returns no
//     error, PrintJSON must return an error matching io.ErrShortWrite and ErrOutputFileWrite.
//  3. JSON delivery under arbitrary text: Given arbitrary string content, PrintJSON must write
//     exactly one decodable JSON document and return no error when the destination accepts writes.
//     When the writer refuses delivery, it must return an error matching io.ErrClosedPipe and
//     ErrOutputFileWrite.
//  4. JSON output fallback: Given a document containing a channel, PrintJSON must return
//     ErrJSONEncode and write one decodable fallback document with failed status and one error
//     message naming the encoding failure.
//  5. Unknown parameter change type: Given an unknown adjustment type, NoticeTexts must return the
//     exact FlagChangedGuard message containing the flag's display name, unknown type, input value,
//     and adjusted value.

// TestJSONFailureCauses verifies invariant #1: JSON failure causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a document containing a channel, PrintJSON must return an error matching ErrJSONEncode and
// json.UnsupportedTypeError. A writable destination must receive one failed-status document with
// one error. If writing that fallback fails, the returned error must also match io.ErrClosedPipe
// and ErrOutputFileWrite.
// Kind: permanent.
func TestJSONFailureCauses(t *testing.T) {
	for _, refuseWrite := range []bool{false, true} {
		var (
			content     bytes.Buffer
			destination io.Writer = &content
		)
		if refuseWrite {
			destination = failingJSONWriter{test: t}
		}

		err := PrintJSON(destination, map[string]any{"invalid": make(chan int)})

		var unsupported *json.UnsupportedTypeError
		if !errors.Is(err, errs.ErrJSONEncode) || !errors.As(err, &unsupported) {
			t.Errorf("✗ encoding failure disappeared: %v", err)
		}

		if refuseWrite {
			if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, errs.ErrOutputFileWrite) {
				t.Errorf("✗ write failure disappeared: %v", err)
			}
		} else {
			var report FailureOutcome
			if decodeErr := json.Unmarshal(content.Bytes(), &report); decodeErr != nil || report.Status != statusFailed || len(report.Errors) != 1 {
				t.Errorf("✗ fallback is not one failure document: %q, %v", content.String(), decodeErr)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ JSON retains encoding and fallback-write failures")
	}
}

// TestJSONShortWrite verifies invariant #2: Complete JSON delivery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When the destination accepts fewer bytes than offered but returns no error, PrintJSON must return
// an error matching io.ErrShortWrite and ErrOutputFileWrite.
// Kind: permanent.
func TestJSONShortWrite(t *testing.T) {
	destination := shortResultWriter{test: t}

	err := PrintJSON(destination, map[string]string{"status": "completed"})
	if !errors.Is(err, io.ErrShortWrite) || !errors.Is(err, errs.ErrOutputFileWrite) {
		t.Errorf("✗ incomplete JSON delivery lost its cause: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ incomplete JSON delivery returns its classified short-write cause")
	}
}

// FuzzJSONDelivery verifies invariant #3: JSON delivery under arbitrary text.
// Test class: Expanded.
// Test layer: Fuzzing.
// What is being tested:
// Given arbitrary string content, PrintJSON must write exactly one decodable JSON document and
// return no error when the destination accepts writes. When the writer refuses delivery, it must
// return an error matching io.ErrClosedPipe and ErrOutputFileWrite.
// Kind: permanent.
func FuzzJSONDelivery(f *testing.F) {
	f.Add("a boat", false)
	f.Add("\x00\xff\n\"<x>&", true)
	f.Fuzz(func(t *testing.T, content string, refuse bool) {
		var (
			destination bytes.Buffer
			writer      io.Writer = &destination
		)
		if refuse {
			writer = failingJSONWriter{test: t}
		}

		err := PrintJSON(writer, map[string]string{"prompt": content})
		if refuse {
			if !errors.Is(err, io.ErrClosedPipe) || !errors.Is(err, errs.ErrOutputFileWrite) {
				t.Errorf("✗ refused delivery loses its cause: %v", err)
			}
		} else {
			decoder := json.NewDecoder(&destination)

			var document map[string]string
			if err != nil || decoder.Decode(&document) != nil {
				t.Errorf("✗ writable JSON failed: %v", err)
			}

			if trailingErr := decoder.Decode(&document); !errors.Is(trailingErr, io.EOF) {
				t.Errorf("✗ result contains more than one JSON document: %v", trailingErr)
			}
		}

		if !t.Failed() {
			t.Log("✓ arbitrary JSON content retains delivery guarantees")
		}
	})

	if !f.Failed() {
		f.Log("✓ JSON delivery fuzz seeds registered")
	}
}

// TestPrintJSONUnencodableDocument verifies invariant #4: JSON output fallback.
//
// What is being tested:
// Given a document containing a channel, PrintJSON must return ErrJSONEncode and write one
// decodable fallback document with failed status and one error message naming the encoding failure.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPrintJSONUnencodableDocument(t *testing.T) {
	var printErr error

	stdout := captureStdout(t, func() {
		printErr = PrintJSON(os.Stdout, map[string]any{"blocked": make(chan int)})
	})
	if !errors.Is(printErr, errs.ErrJSONEncode) {
		t.Errorf("✗ the fallback document lost its encoding failure: %v", printErr)
	}

	var document FailureOutcome
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("💣 the fallback output did not decode: %v\n%s", err, stdout)
	}

	if document.Status != statusFailed || len(document.Errors) != 1 || !strings.Contains(document.Errors[0], errs.ErrJSONEncode.Error()) {
		t.Errorf("✗ fallback document = %#v, want status %q and one error naming the encode failure", document, statusFailed)
	}

	if !t.Failed() {
		t.Log("✓ an unencodable document becomes one valid error document naming the encode failure")
	}
}

// TestChangesUnknownKind verifies invariant #5: Unknown parameter change type.
//
// What is being tested:
// Given an unknown adjustment type, NoticeTexts must return the exact FlagChangedGuard message
// containing the flag's display name, unknown type, input value, and adjusted value.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestChangesUnknownKind(t *testing.T) {
	paramFlags := params.Flags()

	notices := NoticeTexts(paramFlags, []params.Adjustment{{FlagID: params.FlagTypeQuality, Type: "mutant", InputVal: "a", WireVal: "b"}}, map[params.FlagType]string{"output-path": OutPathDisplayName})

	want := fmt.Sprintf(FlagChangedGuard, flagNameByID(t, paramFlags, params.FlagTypeQuality), "mutant", "a", "b")
	if !reflect.DeepEqual(notices, []string{want}) {
		t.Errorf("✗ an unknown classification rendered %q, want the guard form %q", notices, want)
	}

	if !t.Failed() {
		t.Log("✓ an unknown change classification still renders the guard warning")
	}
}

// shortResultWriter accepts all but the last byte of every nonempty document.
type shortResultWriter struct{ test testing.TB }

// Write returns an incomplete count without an explicit failure.
func (writer shortResultWriter) Write(content []byte) (int, error) {
	writer.test.Helper()

	return len(content) - 1, nil
}

// failingJSONWriter refuses delivery with an observable original cause.
type failingJSONWriter struct{ test testing.TB }

// Write returns the refused-delivery cause.
func (writer failingJSONWriter) Write([]byte) (int, error) {
	writer.test.Helper()

	return 0, io.ErrClosedPipe
}
