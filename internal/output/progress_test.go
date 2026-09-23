package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

// Invariants tested:
//  1. Elapsed duration text: For durations spanning seconds, minutes, and hours, ElapsedText must
//     return the specified unit labels and truncate fractional seconds to one decimal place,
//     including at minute boundaries.
//  2. File size text: For the supplied byte counts, fileSizeText must return whole bytes below 1000
//     and decimal KB, MB, or GB with one decimal place above that threshold.
//  3. Generation completion report: Given an elapsed time of 12.1s, PrintGenerationCompleted must
//     include that value in the configured generation-completed message.

// TestElapsedText verifies invariant #1: Elapsed duration text.
//
// What is being tested:
// For durations spanning seconds, minutes, and hours, ElapsedText must return the specified unit
// labels and truncate fractional seconds to one decimal place, including at minute boundaries.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestElapsedText(t *testing.T) {
	cases := []struct {
		elapsed time.Duration
		want    string
	}{
		{0, "0.0s"},
		{100 * time.Millisecond, "0.1s"},
		{12340 * time.Millisecond, "12.3s"},
		{12399 * time.Millisecond, "12.3s"},
		{59900 * time.Millisecond, "59.9s"},
		{59999 * time.Millisecond, "59.9s"},
		{60 * time.Second, "1m 0.0s"},
		{61200 * time.Millisecond, "1m 1.2s"},
		{83600 * time.Millisecond, "1m 23.6s"},
		{90 * time.Second, "1m 30.0s"},
		{600 * time.Second, "10m 0.0s"},
		{3600 * time.Second, "1h 0m 0.0s"},
		{3661 * time.Second, "1h 1m 1.0s"},
		{7323500 * time.Millisecond, "2h 2m 3.5s"},
	}
	for _, c := range cases {
		if got := ElapsedText(c.elapsed); got != c.want {
			t.Errorf("✗ ElapsedText(%v) = %q, want %q", c.elapsed, got, c.want)
		}
	}

	if !t.Failed() {
		t.Logf("✓ the counter renders truncated decimal seconds with minutes and hours prefixes")
	}
}

// TestFileSizeText verifies invariant #2: File size text.
//
// What is being tested:
// For the supplied byte counts, fileSizeText must return whole bytes below 1000 and decimal KB, MB,
// or GB with one decimal place above that threshold.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFileSizeText(t *testing.T) {
	cases := []struct {
		byteCount int64
		want      string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{999, "999 B"},
		{1000, "1.0 KB"},
		{1500, "1.5 KB"},
		{999949, "999.9 KB"},
		{1000000, "1.0 MB"},
		{1234567, "1.2 MB"},
		{1000000000, "1.0 GB"},
		{2500000000, "2.5 GB"},
	}
	for _, c := range cases {
		if got := fileSizeText(c.byteCount); got != c.want {
			t.Errorf("✗ fileSizeText(%d) = %q, want %q", c.byteCount, got, c.want)
		}
	}

	if !t.Failed() {
		t.Logf("✓ sizes render as bytes, then one-decimal KB, MB, and GB")
	}
}

// TestPrintGenerationCompleted verifies invariant #3: Generation completion report.
//
// What is being tested:
// Given an elapsed time of 12.1s, PrintGenerationCompleted must include that value in the
// configured generation-completed message.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintGenerationCompleted(t *testing.T) {
	var buf bytes.Buffer

	destination := &buf
	_ = PrintGenerationCompleted(destination, "12.1s")

	if !strings.Contains(buf.String(), fmt.Sprintf(GenerationCompleted, "12.1s")) {
		t.Errorf("✗ the completion report is wrong: %q", buf.String())
	}

	if !t.Failed() {
		t.Logf("✓ the completion report carries the final counter value")
	}
}
