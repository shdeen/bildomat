package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. JSON HTML characters: Given a document containing angle brackets and an ampersand, PrintJSON
//     must return no error and write those characters literally, without Unicode escape sequences
//     for them.
//  2. JSON output: Given a document containing a quote and newline, PrintJSON must return no error,
//     write exactly one decodable JSON document to stdout with the original values, and write
//     nothing to stderr.
//  3. Parameter change notice forms: Given each supported parameter adjustment type, NoticeTexts
//     must return the exact configured message with the flag's display name and relevant values or
//     reason. It must quote string values, leave integer values bare, use adjusted bounds for
//     capped and raised notices, and honor the supplied output-path display name.
//  4. Parameter change batches: Given no adjustments, NoticeTexts must return no messages. Given a
//     snapped duration followed by a rejected quality, it must return the exact adjustment and
//     rejection messages in that order.

// TestPrintJSONKeepsAngleBrackets verifies invariant #1: JSON HTML characters.
//
// What is being tested:
// Given a document containing angle brackets and an ampersand, PrintJSON must return no error and
// write those characters literally, without Unicode escape sequences for them.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintJSONKeepsAngleBrackets(t *testing.T) {
	stdout := captureStdout(t, func() {
		if err := PrintJSON(os.Stdout, map[string]any{"description": "A value in <x:y> format & more"}); err != nil {
			t.Errorf("✗ JSON write failed: %v", err)
		}
	})

	if !strings.Contains(stdout, "<x:y>") || !strings.Contains(stdout, "& more") {
		t.Errorf("✗ the document escapes angle brackets or the ampersand: %s", stdout)
	}

	if strings.Contains(stdout, `\u003c`) || strings.Contains(stdout, `\u0026`) {
		t.Errorf("✗ the document carries escape sequences: %s", stdout)
	}

	if !t.Failed() {
		t.Log("✓ JSON documents write angle brackets and ampersands as they are")
	}
}

// TestPrintJSON verifies invariant #2: JSON output.
//
// What is being tested:
// Given a document containing a quote and newline, PrintJSON must return no error, write exactly
// one decodable JSON document to stdout with the original values, and write nothing to stderr.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintJSON(t *testing.T) {
	var (
		stdout   string
		printErr error
	)

	stderr := captureStderr(t, func() {
		stdout = captureStdout(t, func() {
			printErr = PrintJSON(os.Stdout, map[string]any{"prompt": "quote \" and newline\n", "status": "completed"})
		})
	})
	if printErr != nil {
		t.Fatalf("💣 JSON write failed: %v", printErr)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty", stderr)
	}

	decoder := json.NewDecoder(strings.NewReader(stdout))

	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("💣 JSON output did not decode: %v\n%s", err, stdout)
	}

	if document["prompt"] != "quote \" and newline\n" || document["status"] != "completed" {
		t.Errorf("✗ decoded document = %#v", document)
	}

	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		t.Errorf("✗ output contains a second JSON value: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ JSON output writes one valid, correctly escaped document to stdout and nothing to stderr")
	}
}

// TestChangeNoticeForms verifies invariant #3: Parameter change notice forms.
//
// What is being tested:
// Given each supported parameter adjustment type, NoticeTexts must return the exact configured
// message with the flag's display name and relevant values or reason. It must quote string values,
// leave integer values bare, use adjusted bounds for capped and raised notices, and honor the
// supplied output-path display name.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestChangeNoticeForms(t *testing.T) {
	paramFlags := params.Flags()

	cases := []struct {
		name   string
		change params.Adjustment
		form   string
		args   []any
	}{
		{
			"a snapped string value renders the adjusted form quoted",
			params.Adjustment{FlagID: params.FlagTypeAspect, Type: params.ChangeSnapped, InputVal: "3:2", WireVal: "1536x1024"},
			FlagAdjusted,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeAspect), "'1536x1024'"},
		},
		{
			"a snapped integer value renders the adjusted form bare",
			params.Adjustment{FlagID: params.FlagTypeDuration, Type: params.ChangeSnapped, InputVal: "5", WireVal: "4"},
			FlagAdjusted,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeDuration), "4"},
		},
		{
			"a derived value renders the adjusted form",
			params.Adjustment{FlagID: params.FlagTypeSize, Type: params.ChangeDerived, InputVal: "512x512", WireVal: "816x816", Comment: "custom-size-image limits"},
			FlagAdjusted,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeSize), "'816x816'"},
		},
		{
			"a conformed value renders the adjusted form",
			params.Adjustment{FlagID: params.FlagTypeInputMedia, Type: params.ChangeConformed, InputVal: "3:ref.png", WireVal: "ref.png", Comment: generation.ReasonNoFramePrefix},
			FlagAdjusted,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeInputMedia), "'ref.png'"},
		},
		{
			"a forced value renders the adjusted form with its reason",
			params.Adjustment{FlagID: params.FlagTypeDuration, Type: params.ChangeForced, WireVal: "8", Comment: "1080p"},
			FlagAdjustedReason,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeDuration), "8", "1080p"},
		},
		{
			"a dropped value renders the not-supported warning",
			params.Adjustment{FlagID: params.FlagTypeResolution, Type: params.ChangeDropped, InputVal: "bogus"},
			FlagNotSupported,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeResolution), "bogus"},
		},
		{
			"a rejected value renders the not-allowed warning",
			params.Adjustment{FlagID: params.FlagTypeQuality, Type: params.ChangeRejected, InputVal: "bogus", Comment: "low|medium|high|auto"},
			FlagNotAllowed,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeQuality), "bogus"},
		},
		{
			"a capped value renders the exceeds warning with the bound as the limit",
			params.Adjustment{FlagID: params.FlagTypeImageN, Type: params.ChangeCapped, InputVal: "12", WireVal: "10", Comment: "max 10"},
			FlagExceedsLimit,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeImageN), "12", "10", "10"},
		},
		{
			"a raised value renders the below warning with the bound as the limit",
			params.Adjustment{FlagID: params.FlagTypeImageN, Type: params.ChangeRaised, InputVal: "0", WireVal: "1", Comment: "min 1"},
			FlagBelowLimit,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeImageN), "0", "1", "1"},
		},
		{
			"an ignored value renders the ignored warning with its reason",
			params.Adjustment{FlagID: params.FlagTypeDuration, Type: params.ChangeIgnored, Comment: fmt.Sprintf(generation.ReasonNotConsumed, "fixture-model")},
			FlagIgnored,
			[]any{flagNameByID(t, paramFlags, params.FlagTypeDuration), fmt.Sprintf(generation.ReasonNotConsumed, "fixture-model")},
		},
		{
			"an output-path record renders under its run-surface name",
			params.Adjustment{FlagID: "output-path", Type: params.ChangeDerived, InputVal: ".jpg", WireVal: ".png"},
			FlagAdjusted,
			[]any{OutPathDisplayName, singleQuote + ".png" + singleQuote},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want := fmt.Sprintf(c.form, c.args...)

			notices := NoticeTexts(paramFlags, []params.Adjustment{c.change}, map[params.FlagType]string{"output-path": OutPathDisplayName})
			if !reflect.DeepEqual(notices, []string{want}) {
				t.Errorf("✗ notices = %q, want %q", notices, want)
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every change classification produces its dictated catalog form")
	}
}

// TestChangesBatch verifies invariant #4: Parameter change batches.
//
// What is being tested:
// Given no adjustments, NoticeTexts must return no messages. Given a snapped duration followed by a
// rejected quality, it must return the exact adjustment and rejection messages in that order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestChangesBatch(t *testing.T) {
	paramFlags := params.Flags()

	notices := NoticeTexts(paramFlags, nil, map[params.FlagType]string{"output-path": OutPathDisplayName})
	if len(notices) != 0 {
		t.Errorf("✗ empty records produced notices: %q", notices)
	}

	notices = NoticeTexts(paramFlags, []params.Adjustment{
		{FlagID: params.FlagTypeDuration, Type: params.ChangeSnapped, InputVal: "5", WireVal: "4"},
		{FlagID: params.FlagTypeQuality, Type: params.ChangeRejected, InputVal: "x", Comment: "low|high"},
	}, map[params.FlagType]string{"output-path": OutPathDisplayName})

	wantFirst := fmt.Sprintf(FlagAdjusted, flagNameByID(t, paramFlags, params.FlagTypeDuration), "4")
	wantSecond := fmt.Sprintf(FlagNotAllowed, flagNameByID(t, paramFlags, params.FlagTypeQuality), "x")

	if !reflect.DeepEqual(notices, []string{wantFirst, wantSecond}) {
		t.Errorf("✗ batch notices = %q, want the two records in order", notices)
	}

	if !t.Failed() {
		t.Log("✓ records render one notice line each in order; an empty set renders nothing")
	}
}

// flagNameByID returns the user-facing FlagName the definitions declare for a flag ID.
func flagNameByID(t *testing.T, paramFlags []params.Flag, flagID params.FlagType) string {
	t.Helper()

	for i := range paramFlags {
		if paramFlags[i].FlagID == flagID {
			return paramFlags[i].FlagName
		}
	}

	t.Fatalf("💣 no parameter-flag definition declares the ID %q", flagID)

	return ""
}
