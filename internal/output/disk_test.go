package output

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/term"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
//  1. Single artifact naming: Given one in-memory image and no title, artifact.WriteMedia must save
//     its bytes as bild-image.png without an indexed file. PrintSavedFile must report that path and
//     byte count exactly. A second untitled write in the same directory must return the stem
//     bild-image-02.
//  2. Multiple artifact naming: Given three in-memory artifacts, artifact.WriteMedia must save
//     their bytes under one stem with suffixes -0, -1, and -2 and each artifact's extension,
//     without creating the bare stem file. PrintSavedFile must report every path and byte count in
//     artifact order.
//  3. File-backed artifact placement: Given a temporary video file, artifact.WriteMedia must save
//     its exact bytes at the returned destination and remove the source. PrintSavedFile must report
//     the saved path and 11-byte size exactly.
//  4. Styled saved-file report: With styling enabled, PrintSavedFile must include the Saved label,
//     the supplied path in clay-colored ANSI text, and the dimmed size 1.2 MB. It must omit the
//     Saved ( prefix used for an elapsed duration.

// TestWriteAllSingle verifies invariant #1: Single artifact naming.
//
// What is being tested:
// Given one in-memory image and no title, artifact.WriteMedia must save its bytes as bild-image.png
// without an indexed file. PrintSavedFile must report that path and byte count exactly. A second
// untitled write in the same directory must return the stem bild-image-02.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteAllSingle(t *testing.T) {
	dir := t.TempDir()

	stem, savedFiles, err := artifact.WriteMedia(dir, "", media.Image, []artifact.Media{{Data: []byte("AAA"), FileExt: ".png"}}, false)

	out := savedFileReports(t, savedFiles)
	if err != nil {
		t.Errorf("✗ artifact.WriteMedia error: %v", err)

		return
	}
	// No title → the bare default stem.
	if stem != "bild-image" {
		t.Errorf("✗ stem = %q, want the default bild-image", stem)
	}

	bare := filepath.Join(dir, stem+".png")

	// #nosec G304 -- bare is a test-owned output path assembled above.
	b, err := os.ReadFile(bare)
	if err != nil {
		t.Errorf("✗ expected bare file %s: %v", bare, err)
	} else if string(b) != "AAA" {
		t.Errorf("✗ content = %q, want AAA", b)
	}

	if _, err := os.Stat(filepath.Join(dir, stem+"-0.png")); err == nil {
		t.Errorf("✗ indexed file written for a single artifact")
	}
	// The save report: exactly one output line naming the written path and byte count.
	if want := fmt.Sprintf(SavedReport, bare, 3) + "\n"; out != want {
		t.Errorf("✗ save report = %q, want %q", out, want)
	}
	// A second no-title run in the same directory sequences to 02.
	stem2, _, err := artifact.WriteMedia(dir, "", media.Image, []artifact.Media{{Data: []byte("BBB"), FileExt: ".png"}}, false)
	if err != nil || stem2 != "bild-image-02" {
		t.Errorf("✗ second run stem = (%q, %v), want the sequenced bild-image-02", stem2, err)
	}

	if !t.Failed() {
		t.Logf("✓ single artifact → bare %s with its exact save report; the default stem sequences", filepath.Base(bare))
	}
}

// TestWriteAllMulti verifies invariant #2: Multiple artifact naming.
//
// What is being tested:
// Given three in-memory artifacts, artifact.WriteMedia must save their bytes under one stem with
// suffixes -0, -1, and -2 and each artifact's extension, without creating the bare stem file.
// PrintSavedFile must report every path and byte count in artifact order.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteAllMulti(t *testing.T) {
	dir := t.TempDir()
	artifacts := []artifact.Media{
		{Data: []byte("A"), FileExt: ".png"},
		{Data: []byte("B"), FileExt: ".jpg"},
		{Data: []byte("C"), FileExt: ".png"},
	}

	stem, savedFiles, err := artifact.WriteMedia(dir, "", media.Image, artifacts, false)

	out := savedFileReports(t, savedFiles)
	if err != nil {
		t.Errorf("✗ artifact.WriteMedia error: %v", err)

		return
	}
	// One save report per artifact, in write order, each with its exact path and byte count.
	var wantOut strings.Builder

	for i, a := range artifacts {
		fmt.Fprintf(&wantOut, SavedReport+"\n", filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, a.FileExt)), len(a.Data))
	}

	if out != wantOut.String() {
		t.Errorf("✗ save reports = %q, want %q", out, wantOut.String())
	}

	for i, a := range artifacts {
		p := filepath.Join(dir, fmt.Sprintf("%s-%d%s", stem, i, a.FileExt))

		// #nosec G304 -- p is a test-owned output path assembled above.
		b, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("✗ missing indexed file %s: %v", p, err)

			continue
		}

		if string(b) != string(a.Data) {
			t.Errorf("✗ %s = %q, want %q", p, b, a.Data)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, stem+".png")); err == nil {
		t.Errorf("✗ bare file written for a multi-artifact batch")
	}

	if !t.Failed() {
		t.Logf("✓ %d artifacts → indexed %s-0..", len(artifacts), stem)
	}
}

// TestFileBackedCopy verifies invariant #3: File-backed artifact placement.
//
// What is being tested:
// Given a temporary video file, artifact.WriteMedia must save its exact bytes at the returned
// destination and remove the source. PrintSavedFile must report the saved path and 11-byte size
// exactly.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFileBackedCopy(t *testing.T) {
	dir := t.TempDir()
	src := srcFile(t, "VIDEO-BYTES")

	stem, savedFiles, err := artifact.WriteMedia(dir, "clip", media.Image, []artifact.Media{{TmpPath: src, FileExt: ".mp4"}}, false)

	out := savedFileReports(t, savedFiles)
	if err != nil {
		t.Fatalf("💣 artifact.WriteMedia(file-backed): %v", err)
	}

	// #nosec G304 -- dir is test-owned and stem is the returned output name.
	b, err := os.ReadFile(filepath.Join(dir, stem+".mp4"))
	if err != nil || string(b) != "VIDEO-BYTES" {
		t.Errorf("✗ written file = (%q,%v), want the source bytes at the claimed path", b, err)
	}

	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ source temp file still present after the move: %v", err)
	}

	if want := fmt.Sprintf(SavedReport+"\n", filepath.Join(dir, stem+".mp4"), 11); out != want {
		t.Errorf("✗ save report = %q, want %q", out, want)
	}

	if !t.Failed() {
		t.Log("✓ a file-backed artifact is written byte-identical from its temporary file, with that file gone and its exact save report")
	}
}

// TestPrintStyledSavedFile verifies invariant #4: Styled saved-file report.
//
// What is being tested:
// With styling enabled, PrintSavedFile must include the Saved label, the supplied path in
// clay-colored ANSI text, and the dimmed size 1.2 MB. It must omit the Saved ( prefix used for an
// elapsed duration.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintStyledSavedFile(t *testing.T) {
	var buf bytes.Buffer

	destination := &buf
	_ = PrintSavedFile(destination, SavedFile{Path: "/tmp/pic.png", Bytes: 1200000}, true)

	report := buf.String()
	if savedPrefix, _, _ := strings.Cut(SavedReportStyled, "%"); !strings.Contains(report, savedPrefix) {
		t.Errorf("✗ the Saved label is missing: %q", report)
	}

	if !strings.Contains(report, "\x1b[38;5;173m/tmp/pic.png") {
		t.Errorf("✗ the clay-colored path is missing: %q", report)
	}

	if !strings.Contains(report, "\x1b[2m(1.2 MB)") {
		t.Errorf("✗ the dimmed human-friendly size is missing: %q", report)
	}

	if strings.Contains(report, "Saved (") {
		t.Errorf("✗ a duration is tied to the file report: %q", report)
	}

	if !t.Failed() {
		t.Logf("✓ the styled saved-file report renders the path and size without a duration")
	}
}

// srcFile writes data to a temporary file with a bild-dl- prefix and returns its path.
func srcFile(t *testing.T, data string) string {
	t.Helper()

	f, err := os.CreateTemp(t.TempDir(), "bild-dl-*")
	if err != nil {
		t.Fatalf("💣 create temp source: %v", err)
	}

	if _, err := f.WriteString(data); err != nil {
		t.Fatalf("💣 write temp source: %v", err)
	}

	if err := f.Close(); err != nil {
		t.Fatalf("💣 close temp source: %v", err)
	}

	return f.Name()
}

// savedFileReports calls PrintSavedFile for each saved file and returns the captured plain-text
// reports in input order.
func savedFileReports(t *testing.T, savedFiles []SavedFile) string {
	t.Helper()

	return captureStdout(t, func() {
		for _, savedFile := range savedFiles {
			_ = PrintSavedFile(os.Stdout, savedFile, term.IsTerminal(int(os.Stdout.Fd())))
		}
	})
}
