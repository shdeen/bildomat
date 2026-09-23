package errs

import (
	"errors"
	"fmt"
	"testing"
)

// Invariants tested:
// 1. Sentinel categories: Given a wrapped error from each listed category, errors.Is must return
//    true for its category root.
// 2. Sentinel chains: For each listed sentinel, errors.Is must match the sentinel itself and its
//    own category root, and reject every other listed root.

// TestSentinelCategories verifies invariant #1: Sentinel categories.
//
// What is being tested:
// Given a wrapped error from each listed category, errors.Is must return true for its category
// root.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSentinelCategories(t *testing.T) {
	transportE := fmt.Errorf("%q, %w, %w", "http://127.0.0.1:1/x", ErrTransportRequest, errors.New("dial refused"))
	parseE := fmt.Errorf("%q, %w, %w", "body", ErrResponseDecode, errors.New("invalid character"))
	inputE := fmt.Errorf("%q, %w, %w", "/nonexistent/bild-test-input.png", ErrInputMediaNotFound, errors.New("no such file or directory"))
	outputE := fmt.Errorf("%q, %w", "dir", ErrOutputFileEmpty)
	resolveE := fmt.Errorf("%q, %w", "definitely-not-a-model", ErrModelResolveUnknown)
	provConfigE := fmt.Errorf("%q, %w, %w", "fixture.json", ErrProvConfigDecode, errors.New("unexpected EOF"))

	cases := []struct {
		name      string
		err, want error
	}{
		{"transport", transportE, ErrTransport},
		{"provider response", parseE, ErrResponse},
		{"input image", inputE, ErrInputMedia},
		{"output file", outputE, ErrOutputFile},
		{"model resolution", resolveE, ErrModelResolve},
		{"provider config", provConfigE, ErrProvConfig},
	}
	for _, c := range cases {
		if !errors.Is(c.err, c.want) {
			t.Errorf("✗ %s error (%v) does not match its root via errors.Is", c.name, c.err)
		}
	}

	if !t.Failed() {
		t.Log("✓ a representative error from each category matches its root via errors.Is")
	}
}

// TestSentinelChains verifies invariant #2: Sentinel chains.
//
// What is being tested:
// For each listed sentinel, errors.Is must match the sentinel itself and its own category root, and
// reject every other listed root. ErrKeyMissing and ErrCanceled must match none of those roots.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSentinelChains(t *testing.T) {
	cases := []struct {
		name     string
		sentinel error
		root     error
	}{
		{"ErrTransportMarshal", ErrTransportMarshal, ErrTransport},
		{"ErrTransportCreate", ErrTransportCreate, ErrTransport},
		{"ErrTransportRequest", ErrTransportRequest, ErrTransport},
		{"ErrTransportRead", ErrTransportRead, ErrTransport},
		{"ErrTransportSize", ErrTransportSize, ErrTransport},
		{"ErrTransportMultipart", ErrTransportMultipart, ErrTransport},
		{"ErrTransportStatus", ErrTransportStatus, ErrTransport},
		{"ErrTransportDownload", ErrTransportDownload, ErrTransport},
		{"ErrTransportTimeout", ErrTransportTimeout, ErrTransport},
		{"ErrTransportRedirectLimit", ErrTransportRedirectLimit, ErrTransport},
		{"ErrResponseDecode", ErrResponseDecode, ErrResponse},
		{"ErrResponseNoData", ErrResponseNoData, ErrResponse},
		{"ErrResponseStatus", ErrResponseStatus, ErrResponse},
		{"ErrResponseGen", ErrResponseGen, ErrResponse},
		{"ErrResponseUnknown", ErrResponseUnknown, ErrResponse},
		{"ErrInputMediaNotFound", ErrInputMediaNotFound, ErrInputMedia},
		{"ErrInputMediaRead", ErrInputMediaRead, ErrInputMedia},
		{"ErrInputMediaMIME", ErrInputMediaMIME, ErrInputMedia},
		{"ErrInputMediaUnsendable", ErrInputMediaUnsendable, ErrInputMedia},
		{"ErrInputMediaEmpty", ErrInputMediaEmpty, ErrInputMedia},
		{"ErrInputMediaSize", ErrInputMediaSize, ErrInputMedia},
		{"ErrInputMediaDecode", ErrInputMediaDecode, ErrInputMedia},
		{"ErrInputMediaEncode", ErrInputMediaEncode, ErrInputMedia},
		{"ErrModelResolveUnknown", ErrModelResolveUnknown, ErrModelResolve},
		{"ErrModelResolveConflict", ErrModelResolveConflict, ErrModelResolve},
		{"ErrOutputFileHome", ErrOutputFileHome, ErrOutputFile},
		{"ErrOutputFileMkdir", ErrOutputFileMkdir, ErrOutputFile},
		{"ErrOutputFileEmpty", ErrOutputFileEmpty, ErrOutputFile},
		{"ErrOutputFileWrite", ErrOutputFileWrite, ErrOutputFile},
		{"ErrOutputFileClose", ErrOutputFileClose, ErrOutputFile},
		{"ErrOutputFileCreate", ErrOutputFileCreate, ErrOutputFile},
		{"ErrProvConfigDecode", ErrProvConfigDecode, ErrProvConfig},
		{"ErrProvConfigInvalid", ErrProvConfigInvalid, ErrProvConfig},
		{"ErrParamValueNumber", ErrParamValueNumber, ErrParamValue},
		{"ErrParamValueInteger", ErrParamValueInteger, ErrParamValue},
		{"ErrParamValueBoolean", ErrParamValueBoolean, ErrParamValue},
		{"ErrParamValueDataType", ErrParamValueDataType, ErrParamValue},
		{"ErrParamValueTypeMismatch", ErrParamValueTypeMismatch, ErrParamValue},
		{"ErrOutputPageParse", ErrOutputPageParse, ErrOutputPage},
		{"ErrOutputPageUnknown", ErrOutputPageUnknown, ErrOutputPage},
		{"ErrOutputPageExecute", ErrOutputPageExecute, ErrOutputPage},
		{"ErrJSONEncode", ErrJSONEncode, ErrJSON},
		{"ErrJSONDecode", ErrJSONDecode, ErrJSON},
		{"ErrKeyMissing", ErrKeyMissing, nil},
		{"ErrCanceled", ErrCanceled, nil},
	}
	// Isolation: a sentinel matches ONLY its own root — a chain accidentally carrying a second
	// root would dispatch the wrong user-message category.
	roots := []error{ErrTransport, ErrResponse, ErrInputMedia, ErrModelResolve, ErrOutputFile, ErrProvConfig, ErrParamValue, ErrOutputPage, ErrJSON}

	for _, c := range cases {
		if !errors.Is(c.sentinel, c.sentinel) {
			t.Errorf("✗ %s does not match itself via errors.Is", c.name)
		}

		if c.root != nil && !errors.Is(c.sentinel, c.root) {
			t.Errorf("✗ %s does not match its category root via errors.Is", c.name)
		}

		for _, r := range roots {
			if !errors.Is(r, c.root) && errors.Is(c.sentinel, r) {
				t.Errorf("✗ %s matches the foreign root %v", c.name, r)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ every precise sentinel matches itself and exactly its own category root; the rootless markers match no root")
	}
}
