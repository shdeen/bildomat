package artifact

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
// 1. Mixed artifact carriage: Given one artifact in memory and one in a temporary file, WriteMedia
//    must save their exact bytes under the returned stem with -0.png and -1.mp4 suffixes and remove
//    the temporary source.
// 2. Requested filename collision: WriteMedia must use the requested stem for the first write, then
//    choose a different stem on a repeated write and preserve the original file's content.
// 3. Unrelated destination lookalikes: When thumb.jpg is free but similarly named files exist,
//    WriteMedia must use thumb.jpg without a numbered suffix and leave every existing file
//    unchanged.
// 4. Sequential filenames: uniqueStem must return the default or requested stem when its
//    destination is free.
// 5. Stem in a missing directory: Given a missing directory, uniqueStem must return the requested
//    stem without a numbered suffix.
// 6. Default stem per medium: Without a requested stem, WriteMedia must return bild-image and its
//    PNG path for an image, or bild-video and its MP4 path for a video.
// 7. Structured artifact write facts: WriteMedia and Write must return the saved paths and byte
//    counts for the artifact and sidecar, and write nothing to stdout.

// TestMixedBatch verifies invariant #1: Mixed artifact carriage.
//
// What is being tested:
// Given one artifact in memory and one in a temporary file, WriteMedia must save their exact bytes
// under the returned stem with -0.png and -1.mp4 suffixes and remove the temporary source.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestMixedBatch(t *testing.T) {
	dir := t.TempDir()
	src := srcFile(t, "FILE-HALF")

	stem, _, err := WriteMedia(dir, "", media.Image, []Media{
		{Data: []byte("BYTES-HALF"), FileExt: ".png"},
		{TmpPath: src, FileExt: ".mp4"},
	}, false)
	if err != nil {
		t.Fatalf("💣 WriteMedia(mixed): %v", err)
	}

	// #nosec G304 -- dir is test-owned and stem is the returned output name.
	if b, _ := os.ReadFile(filepath.Join(dir, stem+"-0.png")); string(b) != "BYTES-HALF" {
		t.Errorf("✗ bytes-backed artifact = %q, want BYTES-HALF", b)
	}

	// #nosec G304 -- dir is test-owned and stem is the returned output name.
	if b, _ := os.ReadFile(filepath.Join(dir, stem+"-1.mp4")); string(b) != "FILE-HALF" {
		t.Errorf("✗ file-backed artifact = %q, want FILE-HALF", b)
	}

	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ source temp file still present: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ a mixed batch writes bytes-backed and file-backed artifacts under one stem")
	}
}

// TestWriteAllTitleConflict verifies invariant #2: Requested filename collision.
//
// What is being tested:
// WriteMedia must use the requested stem for the first write, then choose a different stem on a
// repeated write and preserve the original file's content.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteAllTitleConflict(t *testing.T) {
	dir := t.TempDir()

	s1, _, err := WriteMedia(dir, "sunset", media.Image, []Media{{Data: []byte("1"), FileExt: ".png"}}, false)
	if err != nil {
		t.Errorf("✗ first write: %v", err)

		return
	}

	if s1 != "sunset" {
		t.Errorf("✗ first title stem = %q, want sunset", s1)
	}

	s2, _, err := WriteMedia(dir, "sunset", media.Image, []Media{{Data: []byte("2"), FileExt: ".png"}}, false)
	if err != nil {
		t.Errorf("✗ second write: %v", err)

		return
	}

	if s2 == s1 {
		t.Errorf("✗ conflicting title reused stem %q (would clobber)", s2)
	}

	// #nosec G304 -- dir is a test-owned temporary directory.
	if b, _ := os.ReadFile(filepath.Join(dir, "sunset.png")); string(b) != "1" {
		t.Errorf("✗ original sunset.png clobbered: %q", b)
	}

	if !t.Failed() {
		t.Logf("✓ title honored, conflict avoided (%s, %s)", s1, s2)
	}
}

// TestWriteAllIgnoresNonDestinationFiles verifies invariant #3: Unrelated destination lookalikes.
//
// What is being tested:
// When thumb.jpg is free but similarly named files exist, WriteMedia must use thumb.jpg without a
// numbered suffix and leave every existing file unchanged.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteAllIgnoresNonDestinationFiles(t *testing.T) {
	dir := t.TempDir()
	existingFiles := map[string]string{
		"thumb-02.jpg":  "numbered two",
		"thumb-03.jpg":  "numbered three",
		"thumb-tmp.jpg": "temporary name",
		"thumb.png":     "different extension",
		"thumb.md":      "sidecar not requested",
	}

	for filename, content := range existingFiles {
		// #nosec G306 -- permissions are sufficient for test-owned files.
		if err := os.WriteFile(filepath.Join(dir, filename), []byte(content), 0o644); err != nil {
			t.Fatalf("💣 seed %s: %v", filename, err)
		}
	}

	stem, _, err := WriteMedia(dir, "thumb", media.Image, []Media{{Data: []byte("new image"), FileExt: ".jpg"}}, false)
	if err != nil {
		t.Errorf("✗ write with an available exact destination: %v", err)

		return
	}

	if stem != "thumb" {
		t.Errorf("✗ resolved stem = %q, want the requested stem thumb", stem)
	}

	// #nosec G304 -- the destination belongs to this test's temporary directory.
	writtenImage, err := os.ReadFile(filepath.Join(dir, "thumb.jpg"))
	if err != nil {
		t.Errorf("✗ exact destination thumb.jpg was not written: %v", err)
	} else if string(writtenImage) != "new image" {
		t.Errorf("✗ exact destination content = %q, want new image", writtenImage)
	}

	for filename, content := range existingFiles {
		// #nosec G304 -- the seeded file belongs to this test's temporary directory.
		unchangedContent, readErr := os.ReadFile(filepath.Join(dir, filename))
		if readErr != nil {
			t.Errorf("✗ read seeded file %s: %v", filename, readErr)
		} else if string(unchangedContent) != content {
			t.Errorf("✗ seeded file %s changed to %q, want %q", filename, unchangedContent, content)
		}
	}

	if !t.Failed() {
		t.Log("✓ only an existing exact destination forces a numbered suffix")
	}
}

// TestSequenceNaming verifies invariant #4: Sequential filenames.
//
// What is being tested:
// uniqueStem must return the default or requested stem when its destination is free. If the default
// and -02 paths are occupied, it must choose -03; if only the requested path is occupied, it must
// choose -02.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestSequenceNaming(t *testing.T) {
	dir := t.TempDir()
	artifacts := []Media{{FileExt: ".png"}}

	for _, similarFilename := range []string{"bild-image-tmp.png", "bild-image.jpg", "bild-image-02.png"} {
		// #nosec G306 -- permissions are sufficient for this test-owned seed file.
		if err := os.WriteFile(filepath.Join(dir, similarFilename), []byte("x"), 0o644); err != nil {
			t.Fatalf("💣 seed %s: %v", similarFilename, err)
		}
	}

	if s := uniqueStem(dir, DefaultStem(media.Image), artifacts, false); s != "bild-image" {
		t.Errorf("✗ uniqueStem(with only similar filenames) = %q, want the bare bild-image", s)
	}

	// #nosec G306 -- permissions are sufficient for this test-owned seed file.
	if err := os.WriteFile(filepath.Join(dir, "bild-image.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("💣 seed bild-image: %v", err)
	}

	if s := uniqueStem(dir, DefaultStem(media.Image), artifacts, false); s != "bild-image-03" {
		t.Errorf("✗ uniqueStem with the bare name and -02 taken = %q, want bild-image-03", s)
	}

	if s := uniqueStem(dir, "sunset", artifacts, false); s != "sunset" {
		t.Errorf("✗ uniqueStem(no sibling) = %q, want sunset", s)
	}

	// #nosec G306 -- permissions are sufficient for this test-owned seed file.
	if err := os.WriteFile(filepath.Join(dir, "sunset.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("💣 seed sunset: %v", err)
	}

	if s := uniqueStem(dir, "sunset", artifacts, false); s != "sunset-02" {
		t.Errorf("✗ uniqueStem(conflict) = %q, want sunset-02", s)
	}

	if !t.Failed() {
		t.Log("✓ bild-image-NN uses the smallest available exact destination from 01; a titled exact collision suffixes -02")
	}
}

// TestBareStemMissingDir verifies invariant #5: Stem in a missing directory.
//
// What is being tested:
// Given a missing directory, uniqueStem must return the requested stem without a numbered suffix.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestBareStemMissingDir(t *testing.T) {
	if s := uniqueStem(filepath.Join(t.TempDir(), "does-not-exist"), "seq", []Media{{FileExt: ".png"}}, false); s != "seq" {
		t.Errorf("✗ uniqueStem(missing dir) = %q, want the bare seq", s)
	}

	if !t.Failed() {
		t.Log("✓ a missing directory scans as conflict-free; the stem is taken bare")
	}
}

// TestDefaultStemFollowsMedium verifies invariant #6: Default stem per medium.
//
// What is being tested:
// Without a requested stem, WriteMedia must return bild-image and its PNG path for an image, or
// bild-video and its MP4 path for a video.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDefaultStemFollowsMedium(t *testing.T) {
	for _, c := range []struct {
		media    media.Kind
		ext      string
		wantStem string
	}{
		{media.Video, ".mp4", "bild-video"},
		{media.Image, ".png", "bild-image"},
	} {
		dir := t.TempDir()

		stem, savedFiles, err := WriteMedia(dir, "", c.media, []Media{{Data: []byte("DATA"), FileExt: c.ext}}, false)
		if err != nil {
			t.Fatalf("💣 WriteMedia(%s): %v", c.media, err)
		}

		if stem != c.wantStem {
			t.Errorf("✗ %s stem = %q, want %q", c.media, stem, c.wantStem)
		}

		if len(savedFiles) != 1 || savedFiles[0].Path != filepath.Join(dir, c.wantStem+c.ext) {
			t.Errorf("✗ %s saved files = %+v, want one at %s", c.media, savedFiles, filepath.Join(dir, c.wantStem+c.ext))
		}
	}

	if !t.Failed() {
		t.Log("✓ the default stem names the medium of the run")
	}
}

// TestWriteArtifactsReturnsFacts verifies invariant #7: Structured artifact write facts.
//
// What is being tested:
// WriteMedia and Write must return the saved paths and byte counts for the artifact and sidecar,
// and write nothing to stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestWriteArtifactsReturnsFacts(t *testing.T) {
	dir := t.TempDir()

	var (
		stem      string
		artifacts []SavedFile
		writeErr  error
	)

	stdout := captureStdout(t, func() {
		stem, artifacts, writeErr = WriteMedia(dir, "json-result", media.Image, []Media{{
			Data:    []byte("IMAGE"),
			FileExt: ".png",
		}}, false)
	})
	if writeErr != nil {
		t.Fatalf("💣 artifact write failed: %v", writeErr)
	}

	if stdout != "" {
		t.Errorf("✗ quiet artifact write printed %q", stdout)
	}

	wantArtifactPath := filepath.Join(dir, "json-result.png")
	if stem != "json-result" || len(artifacts) != 1 || artifacts[0].Path != wantArtifactPath || artifacts[0].Bytes != 5 {
		t.Errorf("✗ artifact result = stem %q, files %#v", stem, artifacts)
	}

	var sidecar SavedFile

	stdout = captureStdout(t, func() {
		sidecar, writeErr = Write(dir, stem, SidecarExt, []byte("notes"))
	})
	if writeErr != nil {
		t.Fatalf("💣 sidecar write failed: %v", writeErr)
	}

	if stdout != "" {
		t.Errorf("✗ quiet sidecar write printed %q", stdout)
	}

	if sidecar.Path != filepath.Join(dir, "json-result.md") || sidecar.Bytes != 5 {
		t.Errorf("✗ sidecar result = %#v", sidecar)
	}

	if !t.Failed() {
		t.Log("✓ artifact and sidecar writes return saved-file facts and print nothing themselves")
	}
}

// srcFile writes a bild-dl-* temp source holding data and returns its path.
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

// captureStdout runs fn with os.Stdout redirected and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("💣 close pipe: %v", err)
	}

	os.Stdout = old

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("💣 read captured stdout: %v", err)
	}

	return string(out)
}
