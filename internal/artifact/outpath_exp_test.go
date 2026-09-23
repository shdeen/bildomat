package artifact

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// Invariants tested:
// 1. Output directory creation failure: When a regular file blocks the requested directory path,
//    resolveOutputDir must return ErrOutputFile.
// 2. Filename sanitization under hostile input: Given path traversal, control characters, Unicode,
//    spaces, or empty input, sanitizeFilename must replace separators and control characters with
//    hyphens and preserve all other characters.
// 3. Home directory lookup failure: When HOME is empty, resolveOutputDir("~/x") must return
//    ErrOutputFileHome and include the supplied path in the error.
// 4. Output path parsing under arbitrary input: For arbitrary paths, ParseOutPath must return only
//    supported extensions and formats, omit a format when no extension is recognized, and exclude
//    path separators from the stem.
// 5. Filename sanitization under arbitrary input: For arbitrary text, sanitizeFilename must remove
//    slashes, backslashes, ASCII control characters, and DEL while preserving the input's rune
//    count.

// TestResolveOutDirMkdirError verifies invariant #1: Output directory creation failure.
//
// What is being tested:
// When a regular file blocks the requested directory path, resolveOutputDir must return
// ErrOutputFile.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestResolveOutDirMkdirError(t *testing.T) {
	dir := t.TempDir()

	file := filepath.Join(dir, "blocker")
	// #nosec G306 -- permissions are sufficient for this test-owned blocker file.
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("💣 write blocker file: %v", err)
	}

	_, err := resolveOutputDir(filepath.Join(file, "sub"))
	if err == nil || !errors.Is(err, errs.ErrOutputFile) {
		t.Errorf("✗ resolveOutputDir under a file = %v, want a wrapped ErrOutputFile", err)
	}

	if !t.Failed() {
		t.Log("✓ resolveOutputDir wraps a mkdir failure when a path component is a file")
	}
}

// TestSanitizeHostile verifies invariant #2: Filename sanitization under hostile input.
//
// What is being tested:
// Given path traversal, control characters, Unicode, spaces, or empty input, sanitizeFilename must
// replace separators and control characters with hyphens and preserve all other characters.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSanitizeHostile(t *testing.T) {
	cases := []struct{ in, want string }{
		{"../../etc/passwd", "..-..-etc-passwd"},
		{"a\nb", "a-b"},
		{"a\rb\tc", "a-b-c"},
		{"foo\x00bar", "foo-bar"},
		{`C:\Windows\System32`, "C:-Windows-System32"},
		{"emoji\U0001F600ok", "emoji\U0001F600ok"},
		{"  spaced  ", "  spaced  "},
		{"clean-name_1.2", "clean-name_1.2"},
		{"", ""},
	}
	for _, c := range cases {
		got := sanitizeFilename(c.in)
		if got != c.want {
			t.Errorf("✗ sanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
		}

		if strings.ContainsAny(got, "/\\") {
			t.Errorf("✗ sanitizeFilename(%q) = %q still contains a path separator", c.in, got)
		}
	}

	if !t.Failed() {
		t.Log("✓ sanitizeFilename strips separators and control chars from hostile titles, preserving spaces/dots/unicode")
	}
}

// TestResolveOutDirHomeError verifies invariant #3: Home directory lookup failure.
//
// What is being tested:
// When HOME is empty, resolveOutputDir("~/x") must return ErrOutputFileHome and include the
// supplied path in the error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestResolveOutDirHomeError(t *testing.T) {
	t.Setenv("HOME", "")

	_, err := resolveOutputDir("~/x")
	if err == nil || !errors.Is(err, errs.ErrOutputFileHome) {
		t.Errorf("✗ a failed home lookup = %v, want ErrOutputFileHome", err)
	}

	if err != nil && !strings.Contains(err.Error(), "~/x") {
		t.Errorf("✗ the home-lookup failure does not name the triggering path: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a failed home lookup wraps its sentinel and names the triggering path")
	}
}

// FuzzParseOutPath verifies invariant #4: Output path parsing under arbitrary input.
//
// What is being tested:
// For arbitrary paths, ParseOutPath must return only supported extensions and formats, omit a
// format when no extension is recognized, and exclude path separators from the stem. A trailing
// separator must produce no stem or extension.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzParseOutPath(f *testing.F) {
	for _, seed := range []string{
		"", "dir/", "a/b/", ".", "..", "~", "/", "foo.png", "a/b/foo.jpeg",
		"/abs/foo.webp", "~/x/foo", "archive.v2", ".config", "vid.MP4",
		"my.file.v2", "out/..", "x//y.png", "a\\b", "\x00.png", "….jpg",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, outPath string) {
		parts := ParseOutPath(outPath)
		switch strings.ToLower(parts.Ext) {
		case "", ".png", ".jpg", ".jpeg", ".webp", ".mp4":
		default:
			t.Errorf("✗ %q: extension %q outside the supported vocabulary", outPath, parts.Ext)
		}

		switch parts.Format {
		case "", "png", "jpeg", "webp":
		default:
			t.Errorf("✗ %q: format token %q outside the vocabulary", outPath, parts.Format)
		}

		if parts.Format != "" && parts.Ext == "" {
			t.Errorf("✗ %q: a format token with no extension", outPath)
		}

		if strings.ContainsRune(parts.Stem, os.PathSeparator) {
			t.Errorf("✗ %q: stem %q carries a path separator", outPath, parts.Stem)
		}

		if outPath != "" && os.IsPathSeparator(outPath[len(outPath)-1]) &&
			(parts.Stem != "" || parts.Ext != "") {
			t.Errorf("✗ %q: a separator-terminated value parsed a filename: %+v", outPath, parts)
		}

		if !t.Failed() {
			t.Logf("✓ structural invariants hold")
		}
	})
}

// FuzzSanitize verifies invariant #5: Filename sanitization under arbitrary input.
//
// What is being tested:
// For arbitrary text, sanitizeFilename must remove slashes, backslashes, ASCII control characters,
// and DEL while preserving the input's rune count.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSanitize(f *testing.F) {
	for _, s := range []string{"../../etc", "a/b\\c", "x\x00y", "plain name", "", "tab\tnl\n"} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		got := sanitizeFilename(s)
		if strings.ContainsAny(got, "/\\") {
			t.Errorf("✗ sanitizeFilename(%q) = %q retains a path separator", s, got)
		}

		for _, r := range got {
			if r == '/' || r == '\\' || r < 0x20 || r == 0x7f {
				t.Errorf("✗ sanitizeFilename(%q) = %q retains a forbidden rune %q", s, got, r)
			}
		}

		if got, want := len([]rune(got)), len([]rune(s)); got != want {
			t.Errorf("✗ sanitizeFilename changed the rune count: got %d, want %d", got, want)
		}

		if !t.Failed() {
			t.Logf("✓ sanitization stripped forbidden runes rune-for-rune")
		}
	})
}
