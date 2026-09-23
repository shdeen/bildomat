package media

// Invariants tested:
//  1. Input media read bound: Given a zero-filled 64 MiB file, readSource must return
//     ErrInputMediaMIME without ErrInputMediaSize.
//  2. Invalid input media: Given a missing file, ReadInputs must return ErrInputMediaNotFound under
//     ErrInputMedia and include "not found" in the error.
//  3. Input media failure classification: Given empty, GIF, or PDF files, readSource must return
//     ErrInputMediaMIME and name the detected type.
//  4. Input media directory: Given a directory path, readSource must return an error that wraps
//     both ErrInputMediaRead and ErrInputMedia.
//  5. Overflowed frame times: Given positive or negative overflowing numeric frame prefixes,
//     readSource must return ErrInputMediaTime and include the original source in the error.
//  6. Local read causes: Given a missing file, readSource must preserve os.ErrNotExist,
//     ErrInputMediaNotFound, and the path.
//  7. Input media data URIs under arbitrary input: For arbitrary bytes and MIME strings,
//     Input.DataURI must return an empty string when MIME is empty.
//  8. Source prefix round trips: For arbitrary source strings, splitFramePrefix must either return
//     a classified input-media error naming the source or accept at most one finite nonnegative
//     time or recognized frame anchor.

import (
	"bytes"
	"encoding/base64"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestReadInputBound verifies invariant #1: Input media read bound.
//
// What is being tested:
// Given a zero-filled 64 MiB file, readSource must return ErrInputMediaMIME without
// ErrInputMediaSize. At one byte above that limit, it must return both ErrInputMediaRead and
// ErrInputMediaSize and name the source path.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestReadInputBound(t *testing.T) {
	dir := t.TempDir()
	mk := func(name string, size int64) string {
		t.Helper()

		path := filepath.Join(dir, name)

		// #nosec G304 -- path is inside the test-owned temporary directory.
		f, err := os.Create(path)
		if err != nil {
			t.Fatalf("💣 create %s: %v", name, err)
		}

		if err := f.Truncate(size); err != nil {
			t.Fatalf("💣 truncate %s: %v", name, err)
		}

		if err := f.Close(); err != nil {
			t.Fatalf("💣 close %s: %v", name, err)
		}

		return path
	}

	// The bound is the contract value 64 MiB, written literally so a silent constant change
	// breaks this test. At the bound the read completes: the zero-filled bytes then fail the
	// MIME check (octet-stream), never the size limit.
	at := mk("at-bound.bin", 64<<20)
	if _, err := readSource(at); err == nil || !errors.Is(err, errs.ErrInputMediaMIME) || errors.Is(err, errs.ErrInputMediaSize) {
		t.Errorf("✗ at-bound input err = %v, want a MIME rejection past a completed read", err)
	}

	over := mk("over-bound.bin", 64<<20+1)

	_, err := readSource(over)
	if err == nil || !errors.Is(err, errs.ErrInputMediaRead) || !errors.Is(err, errs.ErrInputMediaSize) {
		t.Errorf("✗ over-bound input err = %v, want errs.ErrInputMediaRead wrapping errs.ErrInputMediaSize", err)
	}

	if err != nil && !strings.Contains(err.Error(), over) {
		t.Errorf("✗ over-bound input err %q does not name the path %q", err, over)
	}

	if !t.Failed() {
		t.Log("✓ an input at the 64 MiB bound reads; one over it fails with the read and size sentinels naming the path")
	}
}

// TestInputMediaInvalid verifies invariant #2: Invalid input media.
//
// What is being tested:
// Given a missing file, ReadInputs must return ErrInputMediaNotFound under ErrInputMedia and
// include "not found" in the error. Given a text file, it must return ErrInputMediaMIME and
// identify text/plain.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestInputMediaInvalid(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.png")
	if _, err := ReadInputs([]string{missing}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("✗ missing input image err = %v, want not found", err)
	} else if !errors.Is(err, errs.ErrInputMediaNotFound) || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ missing input image err = %v, want errs.ErrInputMediaNotFound under errs.ErrInputMedia", err)
	}

	txt := filepath.Join(t.TempDir(), "note.txt")
	// #nosec G306 -- permissions are sufficient for this test-owned non-image file.
	if err := os.WriteFile(txt, []byte("plain text is not an image"), 0o644); err != nil {
		t.Fatalf("💣 write text fixture: %v", err)
	}

	if _, err := ReadInputs([]string{txt}); err == nil || !strings.Contains(err.Error(), "text/plain") {
		t.Errorf("✗ text input err = %v, want detected MIME text/plain", err)
	} else if !errors.Is(err, errs.ErrInputMediaMIME) {
		t.Errorf("✗ text input err = %v, want errs.ErrInputMediaMIME", err)
	}

	if !t.Failed() {
		t.Log("✓ input-media rejects missing files and non-image MIME with their sentinels")
	}
}

// TestInputMediaAdversarial verifies invariant #3: Input media failure classification.
//
// What is being tested:
// Given empty, GIF, or PDF files, readSource must return ErrInputMediaMIME and name the detected
// type. Given a file with no read permission, it must return ErrInputMediaRead, exclude
// ErrInputMediaNotFound, and name the path.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestInputMediaAdversarial(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name string
		data []byte
		mime string // the detected MIME the error must name
	}{
		{"empty.png", nil, "text/plain"},
		{"anim.gif", []byte("GIF89a\x01\x00\x01\x00\x00\x00\x00;"), "image/gif"},
		{"doc.pdf", []byte("%PDF-1.4\n%fake"), "application/pdf"},
	}
	for _, c := range cases {
		path := filepath.Join(dir, c.name)
		// #nosec G306 -- permissions are sufficient for these test-owned image fixtures.
		if err := os.WriteFile(path, c.data, 0o644); err != nil {
			t.Fatalf("💣 write %s fixture: %v", c.name, err)
		}

		_, err := readSource(path)
		if err == nil || !errors.Is(err, errs.ErrInputMediaMIME) {
			t.Errorf("✗ %s err = %v, want errs.ErrInputMediaMIME", c.name, err)

			continue
		}

		if !strings.Contains(err.Error(), c.mime) {
			t.Errorf("✗ %s err %q does not name the detected MIME %q", c.name, err, c.mime)
		}
	}
	// An unreadable existing file (permission denied) is a read failure, not a missing-path
	// failure.
	locked := filepath.Join(dir, "locked.png")
	if err := os.WriteFile(locked, []byte("x"), 0o000); err != nil {
		t.Fatalf("💣 write locked fixture: %v", err)
	}

	_, err := readSource(locked)
	if err == nil || !errors.Is(err, errs.ErrInputMediaRead) || errors.Is(err, errs.ErrInputMediaNotFound) {
		t.Errorf("✗ unreadable input err = %v, want errs.ErrInputMediaRead (not a missing-path error)", err)
	}

	if err != nil && !strings.Contains(err.Error(), locked) {
		t.Errorf("✗ unreadable input err %q does not name the path", err)
	}

	if !t.Failed() {
		t.Log("✓ zero-byte, wrong-MIME, and unreadable inputs are rejected with their sentinels")
	}
}

// TestInputMediaDirectory verifies invariant #4: Input media directory.
//
// What is being tested:
// Given a directory path, readSource must return an error that wraps both ErrInputMediaRead and
// ErrInputMedia.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestInputMediaDirectory(t *testing.T) {
	_, err := readSource(t.TempDir())
	if err == nil || !errors.Is(err, errs.ErrInputMediaRead) || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ ReadInputs(directory) = %v, want errs.ErrInputMediaRead under errs.ErrInputMedia", err)
	}

	if !t.Failed() {
		t.Log("✓ a directory path is rejected as a read failure with its precise sentinel")
	}
}

// TestOverflowFrameTime verifies invariant #5: Overflowed frame times.
//
// What is being tested:
// Given positive or negative overflowing numeric frame prefixes, readSource must return
// ErrInputMediaTime and include the original source in the error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestOverflowFrameTime(t *testing.T) {
	for _, source := range []string{"1e999:https://media.example/asset?a=1,2", "-1e999:https://media.example/asset"} {
		_, err := readSource(source)
		if !errors.Is(err, errs.ErrInputMediaTime) || !strings.Contains(err.Error(), source) {
			t.Errorf("✗ time failure for %q: %v", source, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ numeric overflow remains a classified frame-time failure")
	}
}

// TestLocalReadCauses verifies invariant #6: Local read causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a missing file, readSource must preserve os.ErrNotExist, ErrInputMediaNotFound, and the
// path. Given a directory, it must preserve an os.PathError and ErrInputMediaRead.
// Kind: permanent.
func TestLocalReadCauses(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.png")

	_, err := readSource(missing)
	if !errors.Is(err, os.ErrNotExist) || !errors.Is(err, errs.ErrInputMediaNotFound) || !strings.Contains(err.Error(), missing) {
		t.Errorf("✗ missing source lost cause or context: %v", err)
	}

	directory := t.TempDir()
	_, err = readSource(directory)

	var pathError *os.PathError
	if !errors.As(err, &pathError) || !errors.Is(err, errs.ErrInputMediaRead) {
		t.Errorf("✗ read failure lost filesystem cause or classification: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ local input failures preserve filesystem causes")
	}
}

// FuzzInputMediaPayload verifies invariant #7: Input media data URIs under arbitrary input.
//
// What is being tested:
// For arbitrary bytes and MIME strings, Input.DataURI must return an empty string when MIME is
// empty. Otherwise it must use the supplied MIME prefix and Base64 content that decodes to the
// exact original bytes.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzInputMediaPayload(f *testing.F) {
	f.Add([]byte{0, 1, 2, 3}, "image/png")
	f.Add([]byte("jpeg-ish"), "image/jpeg")
	f.Add([]byte("webp-ish"), "image/webp")
	f.Add([]byte("text"), "text/plain")
	f.Add([]byte{}, "image/png")
	f.Add([]byte("x"), "")
	f.Fuzz(func(t *testing.T, data []byte, mime string) {
		img := Input{Bytes: data, MIME: mime}

		uri := img.DataURI()
		if mime == "" {
			if uri != "" {
				t.Errorf("✗ DataURI with no MIME = %q, want empty", uri)
			}

			return
		}

		prefix := "data:" + mime + ";base64,"
		if !strings.HasPrefix(uri, prefix) {
			t.Errorf("✗ data URI prefix = %q, want %q", uri, prefix)

			return
		}

		got, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(uri, prefix))
		if err != nil || !bytes.Equal(got, data) {
			t.Errorf("✗ data URI round trip = (%d bytes,%v), want %d bytes", len(got), err, len(data))
		}

		if !t.Failed() {
			t.Logf("✓ the URI grammar and round trip held")
		}
	})
}

// FuzzSourcePrefixes verifies invariant #8: Source prefix round trips.
// Test class: Expanded.
// Test layer: Fuzzing.
// What is being tested:
// For arbitrary source strings, splitFramePrefix must either return a classified input-media error
// naming the source or accept at most one finite nonnegative time or recognized frame anchor.
// Rendering an accepted result through Input.Source and reparsing must preserve the bare source,
// anchor, and optional time.
// Kind: permanent.
func FuzzSourcePrefixes(f *testing.F) {
	for _, source := range []string{"image.png", "0:image.png", "1.25:https://media.example/a?a=1,2", "last:https://media.example/a", "1e999:https://media.example/a", "oops:https://media.example/a"} {
		f.Add(source)
	}

	f.Fuzz(func(t *testing.T, source string) {
		bare, timeValue, anchor, err := splitFramePrefix(source)
		if err != nil {
			quotedSource := strconv.Quote(source)
			if !errors.Is(err, errs.ErrInputMedia) || !strings.Contains(err.Error(), quotedSource[1:len(quotedSource)-1]) {
				t.Errorf("✗ unclassified failure or missing source: %q, %v", source, err)
			}

			return
		}

		if timeValue != nil && (math.IsInf(*timeValue, 0) || math.IsNaN(*timeValue) || *timeValue < 0) {
			t.Errorf("✗ invalid accepted time: %v", *timeValue)
		}

		if anchor != "" && anchor != FrameFirst && anchor != FrameLast {
			t.Errorf("✗ unrecognized accepted anchor: %q", anchor)
		}

		if timeValue != nil && anchor != "" {
			t.Error("✗ a source has both time and anchor")
		}

		input := Input{Filepath: bare, Time: timeValue, FrameAnchor: anchor}
		repeatedSource, repeatedTime, repeatedAnchor, repeatedErr := splitFramePrefix(input.Source())
		repeated := Input{Filepath: repeatedSource, Time: repeatedTime, FrameAnchor: repeatedAnchor}
		seconds, present := input.FrameTime()

		repeatedSeconds, repeatedPresent := repeated.FrameTime()
		if repeatedErr != nil || repeatedSource != bare || repeatedAnchor != anchor || seconds != repeatedSeconds || present != repeatedPresent {
			t.Errorf("✗ frame source changed when rendered: %q -> %q", source, input.Source())
		}

		if !t.Failed() {
			t.Log("✓ accepted source prefixes preserve source and optional frame request")
		}
	})
}
