package output

import (
	"regexp"
	"testing"
	"time"
)

// Invariants tested:
//  1. Negative elapsed duration: Given a negative duration, ElapsedText must return 0.0s.
//  2. Negative file size: Given a negative byte count, fileSizeText must return 0 B.
//  3. Elapsed duration formatting under arbitrary input: For durations constructed from arbitrary
//     integer millisecond counts, ElapsedText must return nonnegative seconds with one decimal
//     place and optional minute or hour fields. The seconds field must stay below 60, and minutes
//     following hours must stay below 60.
//  4. File size formatting under arbitrary input: For any integer byte count, fileSizeText must
//     return a nonnegative whole number followed by B, or a nonnegative number with one decimal
//     place followed by KB, MB, or GB.

// TestElapsedTextNegative verifies invariant #1: Negative elapsed duration.
//
// What is being tested:
// Given a negative duration, ElapsedText must return 0.0s.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestElapsedTextNegative(t *testing.T) {
	if got := ElapsedText(-5 * time.Second); got != "0.0s" {
		t.Errorf("✗ ElapsedText(-5s) = %q, want %q", got, "0.0s")
	}

	if !t.Failed() {
		t.Logf("✓ a negative elapsed duration renders as zero")
	}
}

// TestFileSizeTextNegative verifies invariant #2: Negative file size.
//
// What is being tested:
// Given a negative byte count, fileSizeText must return 0 B.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFileSizeTextNegative(t *testing.T) {
	if got := fileSizeText(-42); got != "0 B" {
		t.Errorf("✗ fileSizeText(-42) = %q, want %q", got, "0 B")
	}

	if !t.Failed() {
		t.Logf("✓ a negative byte count renders as zero bytes")
	}
}

// FuzzElapsedText verifies invariant #3: Elapsed duration formatting under arbitrary input.
//
// What is being tested:
// For durations constructed from arbitrary integer millisecond counts, ElapsedText must return
// nonnegative seconds with one decimal place and optional minute or hour fields. The seconds field
// must stay below 60, and minutes following hours must stay below 60.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzElapsedText(f *testing.F) {
	for _, seed := range []int64{0, -1, 100, 59999, 60000, 61200, 3600000, 3661000, 1 << 40} {
		f.Add(seed)
	}

	counterShape := regexp.MustCompile(`^([0-9]+h [0-5]?[0-9]m |[0-9]+m )?[0-5]?[0-9]\.[0-9]s$`)

	f.Fuzz(func(t *testing.T, milliseconds int64) {
		rendered := ElapsedText(time.Duration(milliseconds) * time.Millisecond)
		if !counterShape.MatchString(rendered) {
			t.Errorf("✗ ElapsedText(%dms) = %q, outside both counter shapes", milliseconds, rendered)
		}

		if !t.Failed() {
			t.Logf("✓ the counter shape holds")
		}
	})
}

// FuzzFileSizeText verifies invariant #4: File size formatting under arbitrary input.
//
// What is being tested:
// For any integer byte count, fileSizeText must return a nonnegative whole number followed by B, or
// a nonnegative number with one decimal place followed by KB, MB, or GB.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzFileSizeText(f *testing.F) {
	for _, seed := range []int64{0, -1, 999, 1000, 999999, 1000000, 999999999, 1000000000, 1 << 62} {
		f.Add(seed)
	}

	sizeShape := regexp.MustCompile(`^([0-9]+ B|[0-9]+\.[0-9] (KB|MB|GB))$`)

	f.Fuzz(func(t *testing.T, byteCount int64) {
		rendered := fileSizeText(byteCount)
		if !sizeShape.MatchString(rendered) {
			t.Errorf("✗ fileSizeText(%d) = %q, outside the size shapes", byteCount, rendered)
		}

		if !t.Failed() {
			t.Logf("✓ the size shape holds")
		}
	})
}
