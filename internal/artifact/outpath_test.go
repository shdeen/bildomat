package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

// Invariants tested:
// 1. Output path components: ParseOutPath must split the supplied paths into the expected
//    directory, stem, extension, and format.
// 2. Existing output directories: ParseOutPath must treat an existing extensionless directory as a
//    directory, a missing extensionless name as a stem, and a path ending in .png as a filename
//    even when that path names an existing directory.
// 3. Image format extensions: FormatExt must map png and PNG to .png, jpeg and jpg to .jpg, and
//    webp to .webp.
// 4. Output directory creation: Given an absolute path to a missing nested directory,
//    resolveOutputDir must create the directory and return its absolute path.
// 5. Relative output directories: Given a relative directory path, resolveOutputDir must create
//    that directory beneath the working directory and return its absolute path.
// 6. Tilde-prefixed output directories: Given ~/shots and a configured home directory,
//    resolveOutputDir must create shots beneath that home and return its absolute path.
// 7. Filename sanitization: sanitizeFilename must replace slashes, backslashes, ASCII control
//    characters, and DEL with hyphens, while preserving the other characters in each case.
// 8. Bare tilde output directory: Given ~ and a configured home directory, resolveOutputDir must
//    return that home directory.

// TestParseOutPath verifies invariant #1: Output path components.
//
// What is being tested:
// ParseOutPath must split the supplied paths into the expected directory, stem, extension, and
// format. It must discard unknown extensions, preserve recognized extension case, and treat a
// trailing separator as a directory.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParseOutPath(t *testing.T) {
	cases := map[string]Location{
		"":              {},
		"dir/":          {Dir: "dir/"},
		"a/b/":          {Dir: "a/b/"},
		".":             {Dir: "."},
		"..":            {Dir: ".."},
		"~":             {Dir: "~"},
		"/":             {Dir: "/"},
		"out/..":        {Dir: "out/.."},
		"foo.png":       {Stem: "foo", Ext: ".png", Format: "png"},
		"a/b/foo.jpeg":  {Dir: "a/b", Stem: "foo", Ext: ".jpeg", Format: "jpeg"},
		"pics/foo.jpg":  {Dir: "pics", Stem: "foo", Ext: ".jpg", Format: "jpeg"},
		"/abs/foo.webp": {Dir: "/abs", Stem: "foo", Ext: ".webp", Format: "webp"},
		"~/x/foo":       {Dir: "~/x", Stem: "foo"},
		"./foo.png":     {Dir: ".", Stem: "foo", Ext: ".png", Format: "png"},
		"foo.PNG":       {Stem: "foo", Ext: ".PNG", Format: "png"},
		"vid.mp4":       {Stem: "vid", Ext: ".mp4"},
		"vid.MP4":       {Stem: "vid", Ext: ".MP4"},
		"archive.v2":    {Stem: "archive"},
		"my.file.v2":    {Stem: "my.file"},
		"foo.tar.gz":    {Stem: "foo.tar"},
		"foo":           {Stem: "foo"},
		".png":          {Ext: ".png", Format: "png"},
		".config":       {},
	}
	for input, want := range cases {
		if got := ParseOutPath(input); got != want {
			t.Errorf("✗ ParseOutPath(%q) = %+v, want %+v", input, got, want)
		}
	}

	if !t.Failed() {
		t.Log("✓ the output path parses into its directory, stem, extension, and format token")
	}
}

// TestParseOutPathExistingDirectory verifies invariant #2: Existing output directories.
//
// What is being tested:
// ParseOutPath must treat an existing extensionless directory as a directory, a missing
// extensionless name as a stem, and a path ending in .png as a filename even when that path names
// an existing directory.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestParseOutPathExistingDirectory(t *testing.T) {
	parentDir := t.TempDir()

	existingDir := filepath.Join(parentDir, "existing")
	if err := os.Mkdir(existingDir, 0o750); err != nil {
		t.Fatalf("💣 create extensionless directory: %v", err)
	}

	if got := ParseOutPath(existingDir); got != (Location{Dir: existingDir}) {
		t.Errorf("✗ existing extensionless directory parsed as %+v", got)
	}

	extensionDir := filepath.Join(parentDir, "named.png")
	if err := os.Mkdir(extensionDir, 0o750); err != nil {
		t.Fatalf("💣 create extension-bearing directory: %v", err)
	}

	wantExtensionFile := Location{Dir: parentDir, Stem: "named", Ext: ".png", Format: "png"}
	if got := ParseOutPath(extensionDir); got != wantExtensionFile {
		t.Errorf("✗ extension-bearing path parsed as %+v, want file parts %+v", got, wantExtensionFile)
	}

	missingPath := filepath.Join(parentDir, "missing")

	wantMissingFile := Location{Dir: parentDir, Stem: "missing"}
	if got := ParseOutPath(missingPath); got != wantMissingFile {
		t.Errorf("✗ missing extensionless path parsed as %+v, want file parts %+v", got, wantMissingFile)
	}

	if !t.Failed() {
		t.Log("✓ an existing extensionless directory is recognized after the explicit file and directory forms")
	}
}

// TestFormatExt verifies invariant #3: Image format extensions.
//
// What is being tested:
// FormatExt must map png and PNG to .png, jpeg and jpg to .jpg, and webp to .webp. Empty input,
// gif, and mp4 must return an empty string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFormatExt(t *testing.T) {
	cases := map[string]string{
		"png":  ".png",
		"jpeg": ".jpg",
		"jpg":  ".jpg",
		"webp": ".webp",
		"PNG":  ".png",
		"":     "",
		"gif":  "",
		"mp4":  "",
	}
	for token, want := range cases {
		if got := FormatExt(token); got != want {
			t.Errorf("✗ FormatExt(%q) = %q, want %q", token, got, want)
		}
	}

	if !t.Failed() {
		t.Log("✓ accepted image-format tokens map to their canonical extensions; anything else maps to none")
	}
}

// TestResolveOutDir verifies invariant #4: Output directory creation.
//
// What is being tested:
// Given an absolute path to a missing nested directory, resolveOutputDir must create the directory
// and return its absolute path.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveOutDir(t *testing.T) {
	nested := filepath.Join(t.TempDir(), "auto", "deep", "nest")

	got, err := resolveOutputDir(nested)
	if err != nil {
		t.Errorf("✗ resolveOutputDir error: %v", err)
	}

	if got != nested {
		t.Errorf("✗ returned %q, want %q", got, nested)
	}

	if fi, err := os.Stat(nested); err != nil || !fi.IsDir() {
		t.Errorf("✗ nested path not created: %v", err)
	}

	if !t.Failed() {
		t.Logf("✓ missing nested path created (mkdir -p)")
	}
}

// TestResolveOutDirRelative verifies invariant #5: Relative output directories.
//
// What is being tested:
// Given a relative directory path, resolveOutputDir must create that directory beneath the working
// directory and return its absolute path.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveOutDirRelative(t *testing.T) {
	cases := []struct {
		name string
		raw  string
	}{
		{"default", "images"},
		{"explicit relative", "shots"},
		{"explicit nested relative", filepath.Join("a", "b")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assertResolvedUnderWorkingDir(t, c.raw)
		})
	}

	if !t.Failed() {
		t.Log("✓ relative output directories resolve to absolute paths under the working directory and are created there")
	}
}

// TestResolveOutDirTilde verifies invariant #6: Tilde-prefixed output directories.
//
// What is being tested:
// Given ~/shots and a configured home directory, resolveOutputDir must create shots beneath that
// home and return its absolute path.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveOutDirTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := resolveOutputDir("~/shots")
	if err != nil {
		t.Errorf("✗ resolveOutputDir(~/shots) error: %v", err)
	}

	want := filepath.Join(home, "shots")
	if got != want {
		t.Errorf("✗ resolveOutputDir(~/shots) = %q, want %q", got, want)
	}

	if fi, err := os.Stat(want); err != nil || !fi.IsDir() {
		t.Errorf("✗ expanded ~ path not created: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ resolveOutputDir expands a leading ~ to the home directory and creates it")
	}
}

// TestSanitize verifies invariant #7: Filename sanitization.
//
// What is being tested:
// sanitizeFilename must replace slashes, backslashes, ASCII control characters, and DEL with
// hyphens, while preserving the other characters in each case.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSanitize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"a/b", "a-b"},
		{`a\b`, "a-b"},
		{"a\tb", "a-b"},   // control char (tab)
		{"a\x00b", "a-b"}, // NUL
		{"a\x7fb", "a-b"}, // DEL
		{"hello world", "hello world"},
		{"clean-name_1.2", "clean-name_1.2"},
	}
	for _, c := range cases {
		if got := sanitizeFilename(c.in); got != c.want {
			t.Errorf("✗ sanitizeFilename(%q) = %q, want %q", c.in, got, c.want)
		}
	}

	if !t.Failed() {
		t.Log("✓ sanitizeFilename replaces /, \\, and control chars with -, preserves spaces and other chars")
	}
}

// TestResolveOutDirBareTilde verifies invariant #8: Bare tilde output directory.
//
// What is being tested:
// Given ~ and a configured home directory, resolveOutputDir must return that home directory.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveOutDirBareTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := resolveOutputDir("~")
	if err != nil || got != home {
		t.Errorf("✗ resolveOutputDir(~) = (%q,%v), want the home directory", got, err)
	}

	if !t.Failed() {
		t.Log("✓ a bare ~ resolves to the home directory")
	}
}

// assertResolvedUnderWorkingDir resolves a relative output directory from a fresh working directory
// and asserts the result is that directory's absolute path, created on disk.
func assertResolvedUnderWorkingDir(t *testing.T, relativeDir string) {
	t.Helper()

	workingDir := t.TempDir()
	t.Chdir(workingDir)

	resolvedDir, err := resolveOutputDir(relativeDir)
	if err != nil {
		t.Errorf("✗ resolveOutputDir(%q) error: %v", relativeDir, err)
	}

	expectedDir := filepath.Join(workingDir, relativeDir)
	if resolvedDir != expectedDir {
		t.Errorf("✗ resolveOutputDir(%q) = %q, want the absolute %q", relativeDir, resolvedDir, expectedDir)
	}

	if fi, err := os.Stat(expectedDir); err != nil || !fi.IsDir() {
		t.Errorf("✗ %q not created under the working directory: %v", expectedDir, err)
	}

	if !t.Failed() {
		t.Logf("✓ resolveOutputDir(%q) → %q (absolute under the working directory)", relativeDir, resolvedDir)
	}
}
