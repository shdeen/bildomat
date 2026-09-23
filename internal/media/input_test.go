package media

// Invariants tested:
//  1. Data URI without input media: Given an empty Input, DataURI must return an empty string.
//  2. Valid local input media: Given local PNG, JPEG, and WebP files, ReadInputs must return one
//     record per file in input order with the exact source bytes, path, and detected MIME type.
//  3. Local and remote input media sources: Given a local MP4 and HTTP or HTTPS URLs, ReadInputs
//     must retain the MP4 bytes, path, and video MIME type.
//  4. Read input media times: Given prefixed URLs, ReadInputs must preserve zero and 8.5-second
//     times, recognize first and last, and leave plain URLs without frame markers.
//  5. Data URI with an empty payload: Given MIME image/png and no payload bytes, Input.DataURI must
//     return data:image/png;base64,.

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
)

// TestDataURINil verifies invariant #1: Data URI without input media.
//
// What is being tested:
// Given an empty Input, DataURI must return an empty string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDataURINil(t *testing.T) {
	if got := (Input{}).DataURI(); got != "" {
		t.Errorf("✗ DataURI(empty) = %q, want empty", got)
	}

	if !t.Failed() {
		t.Log("✓ DataURI of an empty image returns the empty string")
	}
}

// TestInputMediaValid verifies invariant #2: Valid local input media.
//
// What is being tested:
// Given local PNG, JPEG, and WebP files, ReadInputs must return one record per file in input order
// with the exact source bytes, path, and detected MIME type.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestInputMediaValid(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		mime string
		b64  string
	}{
		{"one.png", "image/png", "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="},
		{"one.jpg", "image/jpeg", "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////2wBDAf//////////////////////////////////////////////////////////////////////////////////////wAARCAABAAEDASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAX/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oADAMBAAIQAxAAAAF/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAEFAqf/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oACAEDAQE/ASP/xAAUEQEAAAAAAAAAAAAAAAAAAAAA/9oACAECAQE/ASP/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAY/Al//xAAUEAEAAAAAAAAAAAAAAAAAAAAA/9oACAEBAAE/IV//2gAMAwEAAgADAAAAEP/EABQRAQAAAAAAAAAAAAAAAAAAABD/2gAIAQMBAT8QH//EABQRAQAAAAAAAAAAAAAAAAAAABD/2gAIAQIBAT8QH//EABQQAQAAAAAAAAAAAAAAAAAAABD/2gAIAQEAAT8QH//Z"},
		{"one.webp", "image/webp", "UklGRiIAAABXRUJQVlA4IBYAAAAwAQCdASoBAAEAAQAcJaQAA3AA/vuUAAA="},
	}

	paths := make([]string, len(cases))
	for i, c := range cases {
		data, err := base64.StdEncoding.DecodeString(c.b64)
		if err != nil {
			t.Fatalf("💣 decode %s fixture: %v", c.name, err)
		}

		paths[i] = filepath.Join(dir, c.name)
		// #nosec G306 -- permissions are sufficient for these test-owned image fixtures.
		if err := os.WriteFile(paths[i], data, 0o644); err != nil {
			t.Fatalf("💣 write %s fixture: %v", c.name, err)
		}
	}

	imgs, err := ReadInputs(paths)
	if err != nil {
		t.Errorf("✗ ReadInputs valid images: %v", err)

		return
	}

	if len(imgs) != len(cases) {
		t.Errorf("✗ got %d images, want %d", len(imgs), len(cases))
	}

	for i, c := range cases {
		if i >= len(imgs) {
			continue
		}

		want, err := base64.StdEncoding.DecodeString(c.b64)
		if err != nil {
			t.Fatalf("💣 decode %s fixture: %v", c.name, err)
		}

		path := filepath.Join(dir, c.name)
		if imgs[i].MIME != c.mime {
			t.Errorf("✗ %s MIME = %q, want %q", c.name, imgs[i].MIME, c.mime)
		}

		if !bytes.Equal(imgs[i].Bytes, want) {
			t.Errorf("✗ %s bytes differ from the source file (%d bytes read, %d written)", c.name, len(imgs[i].Bytes), len(want))
		}

		if imgs[i].Filepath != path {
			t.Errorf("✗ %s path = %q, want the source path %q", c.name, imgs[i].Filepath, path)
		}
	}

	if !t.Failed() {
		t.Log("✓ input-media accepts PNG/JPEG/WebP, carrying exact bytes, path, and order")
	}
}

// TestReadInputsSources verifies invariant #3: Local and remote input media sources.
//
// What is being tested:
// Given a local MP4 and HTTP or HTTPS URLs, ReadInputs must retain the MP4 bytes, path, and video
// MIME type. It must preserve each URL in order without assigning local bytes, a local path, a MIME
// type, or a media kind.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestReadInputsSources(t *testing.T) {
	dir := t.TempDir()
	mp4Path := filepath.Join(dir, "clip.mp4")

	mp4Data := []byte("\x00\x00\x00\x18ftypmp42\x00\x00\x00\x00mp42isom")
	// #nosec G306 -- permissions are sufficient for this test-owned media fixture.
	if err := os.WriteFile(mp4Path, mp4Data, 0o644); err != nil {
		t.Fatalf("💣 write MP4 fixture: %v", err)
	}

	sources := []string{
		mp4Path,
		"HTTPS://media.example/reference.png?download=1#preview",
		"http://media.example/continuation.mp4",
		"https://res.cloudinary.com/zenbusiness/q_auto,w_1050/v1670445040/logaster/logaster-2013-06-jpg.avif",
		"https://media.example/no-extension",
	}

	mediaInputs, err := ReadInputs(sources)
	if err != nil {
		t.Fatalf("💣 ReadInputs: %v", err)
	}

	if len(mediaInputs) != 5 {
		t.Fatalf("💣 input count = %d, want 5", len(mediaInputs))
	}

	if mediaInputs[0].Filepath != mp4Path || mediaInputs[0].URL != "" || mediaInputs[0].MIME != mimeMP4 || mediaInputs[0].Kind() != Video || !bytes.Equal(mediaInputs[0].Bytes, mp4Data) {
		t.Errorf("✗ local MP4 = %+v, want file-only video metadata and exact bytes", mediaInputs[0])
	}

	for mediaIndex, mediaInput := range mediaInputs[1:] {
		if mediaInput.URL != sources[mediaIndex+1] || mediaInput.Filepath != "" || len(mediaInput.Bytes) != 0 || mediaInput.MIME != "" || mediaInput.Kind() != "" {
			t.Errorf("✗ remote source acquired unverified metadata: %+v", mediaInput)
		}
	}

	if !t.Failed() {
		t.Log("✓ URLs remain unresolved and local MP4 retains its bytes")
	}
}

// TestReadInputsTimes verifies invariant #4: Read input media times.
//
// What is being tested:
// Given prefixed URLs, ReadInputs must preserve zero and 8.5-second times, recognize first and
// last, and leave plain URLs without frame markers. It must reject negative times and FTP sources
// with ErrInputMedia, and reject an unknown prefix with ErrInputMediaSource.
//
// Test class: Expanded.
// Test layer: Coverage.
// Kind: permanent.
func TestReadInputsTimes(t *testing.T) {
	mediaInputs, err := ReadInputs([]string{
		"0:https://media.example/open.webp",
		"8.5:https://media.example/close.jpg",
		"first:https://media.example/lead.png",
		"last:https://media.example/tail.png",
		"https://media.example/plain.png",
	})
	if err != nil {
		t.Fatalf("💣 ReadInputs framed URLs: %v", err)
	}

	if seconds, set := mediaInputs[0].FrameTime(); !set || seconds != 0 {
		t.Errorf("✗ opening time = (%v, %v), want (0, true)", seconds, set)
	}

	if seconds, set := mediaInputs[1].FrameTime(); !set || seconds != 8.5 {
		t.Errorf("✗ closing time = (%v, %v), want (8.5, true)", seconds, set)
	}

	if mediaInputs[2].FrameAnchor != FrameFirst || mediaInputs[3].FrameAnchor != FrameLast {
		t.Errorf("✗ anchors = (%q, %q), want (first, last)", mediaInputs[2].FrameAnchor, mediaInputs[3].FrameAnchor)
	}

	for i, mediaInput := range mediaInputs[2:5] {
		if _, set := mediaInput.FrameTime(); set {
			t.Errorf("✗ input %d carries a numeric time, want none", i+2)
		}
	}

	if mediaInputs[4].HasFrame() {
		t.Errorf("✗ a plain URL reports a frame request")
	}

	// The URL scheme's own colon is not a frame prefix. Remote extensions do not decide
	// provider compatibility; only an invalid frame or URL source is rejected here.
	invalidSources := []string{
		"-1:https://media.example/open.png",
		"ftp://media.example/reference.png",
	}
	for _, source := range invalidSources {
		if _, err := ReadInputs([]string{source}); err == nil || !errors.Is(err, errs.ErrInputMedia) {
			t.Errorf("✗ ReadInputs(%q) error = %v, want input-media rejection", source, err)
		}
	}

	// A non-numeric, non-keyword prefix is no prefix at all: the whole text is the source,
	// which fails here only because it is not a supported source.
	if _, err := ReadInputs([]string{"oops:https://media.example/open.png"}); err == nil || !errors.Is(err, errs.ErrInputMediaSource) {
		t.Errorf("✗ an unrecognized prefix = %v, want the whole text treated as a source", err)
	}
}

// TestDataURIEmptyPayload verifies invariant #5: Data URI with an empty payload.
//
// What is being tested:
// Given MIME image/png and no payload bytes, Input.DataURI must return data:image/png;base64,.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestDataURIEmptyPayload(t *testing.T) {
	if got := (Input{MIME: mimePNG}).DataURI(); got != "data:image/png;base64," {
		t.Errorf("✗ DataURI(empty payload, png) = %q, want the empty-payload URI", got)
	}

	if !t.Failed() {
		t.Log("✓ an empty payload with a declared MIME yields the empty-payload URI")
	}
}
