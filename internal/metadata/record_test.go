package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
// 1. Submitted multipart data: After recording and saving multipart input, Record must preserve the
//    order, names, and values of repeated text and file parts.
// 2. Complete transactions: After recording three request/response pairs and a generation failure,
//    Record.Save must preserve all three pairs with their response types and request indexes.
// 3. Binary retention: Record.Save must replace two identical encoded response values and a request
//    data URI with references to the existing artifact. When an encoded SVG response matches a
//    saved artifact, Record.Save must record image/svg+xml, reference that artifact in the
//    response, and create only the JSON record alongside the SVG.
// 4. Record filenames: Saving two records with the stem boat-02 must produce boat-02.bild.json and
//    boat-02.bild-02.json.

// TestMultipartRetention verifies invariant #1: Submitted multipart data.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// After recording and saving multipart input, Record must preserve the order, names, and values of
// repeated text and file parts. It must reference the unchanged input at its original path and save
// transformed bytes separately, producing one retained binary and one record.
//
// Kind: permanent.
func TestMultipartRetention(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "source.bin")
	original := []byte("original input bytes")
	prepared := []byte("transformed input bytes")

	if err := os.WriteFile(inputPath, original, 0o600); err != nil {
		t.Fatalf("💣 input fixture: %v", err)
	}

	var body bytes.Buffer

	form := multipart.NewWriter(&body)
	for _, value := range []string{"first", "second"} {
		if err := form.WriteField("label", value); err != nil {
			t.Fatalf("💣 multipart field: %v", err)
		}
	}

	for _, content := range [][]byte{original, prepared} {
		part, err := form.CreateFormFile("image[]", "source.bin")
		if err != nil {
			t.Fatalf("💣 multipart file: %v", err)
		}

		if _, err := io.Copy(part, bytes.NewReader(content)); err != nil {
			t.Fatalf("💣 multipart bytes: %v", err)
		}
	}

	if err := form.Close(); err != nil {
		t.Fatalf("💣 multipart close: %v", err)
	}

	record := New("example", "model", "test", "boat", time.Now())
	record.Prepared([]media.Input{
		{Filepath: inputPath, Bytes: original},
		{Filepath: inputPath, Bytes: prepared},
	})
	record.Begin("POST", "https://example.com/images", form.FormDataContentType(), body.Bytes())

	directory := t.TempDir()
	if _, err := record.Save(directory, "boat", nil, nil); err != nil {
		t.Errorf("✗ record persistence: %v", err)
	}

	parts := decodeObjects(t, record.Request.Calls[0].Payload)
	if len(parts) != 4 {
		t.Fatalf("💣 cannot inspect %d retained parts", len(parts))
	}

	for index, value := range []string{"first", "second"} {
		if string(parts[index]["name"]) != `"label"` || string(parts[index]["value"]) != `"`+value+`"` {
			t.Errorf("✗ repeated field %d: %s", index, record.Request.Calls[0].Payload)
		}
	}

	for index, content := range [][]byte{original, prepared} {
		part := parts[index+2]
		if string(part["name"]) != `"image[]"` || string(part["filename"]) != `"source.bin"` {
			t.Errorf("✗ file part %d lost identity: %v", index, part)
		}

		var retainedPath string
		if err := json.Unmarshal(part["file"], &retainedPath); err != nil {
			t.Errorf("✗ retained file path: %v", err)
		}

		data, err := os.ReadFile(retainedPath) //nolint:gosec // Inspect files produced by this test in temporary directories.
		if err != nil || !bytes.Equal(data, content) {
			t.Errorf("✗ retained bytes differ: %q, %v", data, err)
		}

		if index == 0 && retainedPath != inputPath || index == 1 && retainedPath == inputPath {
			t.Errorf("✗ input byte identity not respected: %d, %q", index, retainedPath)
		}
	}

	files, err := os.ReadDir(directory)
	if err != nil || len(files) != 2 {
		t.Errorf("✗ expected one retained binary and one record, found %d: %v", len(files), err)
	}

	if !t.Failed() {
		t.Log("✓ multipart records preserve repeated parts and actual submitted bytes")
	}
}

// TestRecordTransaction verifies invariant #2: Complete transactions.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// After recording three request/response pairs and a generation failure, Record.Save must preserve
// all three pairs with their response types and request indexes. The saved record must retain the
// exact integer 9007199254740993, a null value, one return-value entry, and an error entry.
//
// Kind: permanent.
func TestRecordTransaction(t *testing.T) {
	record := New("google", "veo-3.1-generate-preview", "test", "a paper boat", time.Now())
	submission := record.Begin("POST", "https://example.com/jobs", "application/json", []byte(`{"prompt":"a paper boat","seed":9007199254740993}`))
	record.Receive(submission, "synchronous", 200, "application/json", []byte(`{"name":"operations/boat","unknown":{"number":9007199254740993,"nullable":null}}`), nil)

	for _, body := range []string{`{"done":false}`, `{"done":true,"video":{"uri":"https://example.com/boat?token=a=b"}}`} {
		poll := record.Begin("GET", "https://example.com/operations/boat", "", nil)
		record.Receive(poll, "asynchronous", 200, "application/json", []byte(body), nil)
	}

	record.Retain("google", "veo-3.1-generate-preview", json.RawMessage(`{"operation":"operations/boat","videos":[{"uri":"https://example.com/boat?token=a=b"}]}`))

	path, err := record.Save(t.TempDir(), "boat", nil, os.ErrPermission)
	if err != nil {
		t.Errorf("✗ save returned %v", err)
	}

	document := readRecordJSON(t, path)
	responses := decodeObjects(t, document["provider-responses"])

	requests := decodeObjects(t, decodeObject(t, document["request"])["provider-requests"])
	if len(responses) != 3 || len(requests) != 3 {
		t.Errorf("✗ incomplete transaction: %d responses, %d requests", len(responses), len(requests))
	}

	for index, response := range responses {
		var requestIndex int
		if err := json.Unmarshal(response["request-index"], &requestIndex); err != nil || requestIndex != index {
			t.Errorf("✗ response %d has request %s", index, response["request-index"])
		}

		responseType := "asynchronous"
		if index == 0 {
			responseType = "synchronous"
		}

		if string(response["type"]) != `"`+responseType+`"` {
			t.Errorf("✗ response type: %s", response["type"])
		}
	}

	if len(responses) > 0 {
		unknown := decodeObject(t, decodeObject(t, responses[0]["full-response"])["unknown"])
		if string(unknown["number"]) != "9007199254740993" || string(unknown["nullable"]) != "null" {
			t.Errorf("✗ changed unknown values: %s", responses[0]["full-response"])
		}
	}

	if len(decodeObjects(t, document["return-values"])) != 1 || len(document["errors"]) == 0 {
		t.Error("✗ lost retained values or generation error")
	}

	if !t.Failed() {
		t.Log("✓ complete transaction survives generation failure")
	}
}

// TestRecordBinary verifies invariant #3: Binary retention.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Record.Save must replace two identical encoded response values and a request data URI with
// references to the existing artifact. It must save a different decoded value once as a WebP file,
// preserve an unrecognized data string, and leave the original response bytes unchanged.
//
// Kind: permanent.
func TestRecordBinary(t *testing.T) {
	directory := t.TempDir()
	artifactBytes := []byte("actual generated bytes")

	artifactPath := filepath.Join(directory, "boat-02.png")
	if err := os.WriteFile(artifactPath, artifactBytes, 0o600); err != nil {
		t.Fatalf("💣 fixture: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(artifactBytes)
	extraBytes := []byte("unselected media")
	extraEncoded := base64.StdEncoding.EncodeToString(extraBytes)
	body := []byte(`{"data":[{"b64_json":"` + encoded + `","mime_type":"image/png"},{"b64_json":"` + encoded + `"},{"b64_json":"` + extraEncoded + `","mime_type":"image/webp"}],"unknown":{"data":"` + encoded + `"}}`)
	bodyCopy := bytes.Clone(body)
	record := New("example", "image", "test", "test prompt", time.Now())
	record.SetFields(nil, []BinaryField{{Path: []string{"data", "*", "b64_json"}, MIMEField: "mime_type"}})
	requestIndex := record.Begin("POST", "https://example.com/images", "application/json", []byte(`{"image":"data:image/png;base64,`+encoded+`"}`))
	record.Receive(requestIndex, "synchronous", 200, "application/json", body, nil)

	path, err := record.Save(directory, "boat-02", []artifact.SavedFile{{Path: artifactPath, Bytes: int64(len(artifactBytes))}}, nil)
	if err != nil {
		t.Errorf("✗ save returned %v", err)
	}

	document := readRecordJSON(t, path)

	responses := decodeObjects(t, document["provider-responses"])
	if len(responses) != 1 {
		t.Fatalf("💣 cannot inspect %d responses", len(responses))
	}

	response := decodeObject(t, responses[0]["full-response"])

	images := decodeObjects(t, response["data"])
	if len(images) != 3 {
		t.Fatalf("💣 cannot inspect %d images", len(images))
	}

	marker := "[Bildomat: base64 data redacted for length; saved locally to " + artifactPath + "]"

	for index := range 2 {
		var reference string
		if err := json.Unmarshal(images[index]["b64_json"], &reference); err != nil || reference != marker {
			t.Errorf("✗ artifact reference %d: %s", index, images[index]["b64_json"])
		}
	}

	var extraMarker string
	if err := json.Unmarshal(images[2]["b64_json"], &extraMarker); err != nil {
		t.Errorf("✗ extra reference: %v", err)
	}

	extraPath := strings.TrimSuffix(strings.TrimPrefix(extraMarker, "[Bildomat: base64 data redacted for length; saved locally to "), "]")

	decoded, readErr := os.ReadFile(extraPath) //nolint:gosec // Inspect the retained file created in this test's temporary directory.
	if readErr != nil || !bytes.Equal(decoded, extraBytes) || filepath.Ext(extraPath) != ".webp" {
		t.Errorf("✗ retained media %q: %v", extraPath, readErr)
	}

	if !bytes.Equal(body, bodyCopy) {
		t.Error("✗ normalization mutated the response used for generation")
	}

	if string(decodeObject(t, response["unknown"])["data"]) != `"`+encoded+`"` {
		t.Error("✗ unknown data string was guessed to be binary")
	}

	files, readErr := os.ReadDir(directory)
	if readErr != nil || len(files) != 3 {
		t.Errorf("✗ duplicate or missing files: %d, %v", len(files), readErr)
	}

	requests := decodeObjects(t, decodeObject(t, document["request"])["provider-requests"])
	if len(requests) != 1 || !bytes.Contains(requests[0]["payload"], []byte(artifactPath)) {
		t.Error("✗ actual request data URI was not replaced by artifact reference")
	}

	if !t.Failed() {
		t.Log("✓ known binary content retains exact files without duplicates")
	}
}

// TestArtifactMediaType verifies invariant #3: Binary retention.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// When an encoded SVG response matches a saved artifact, Record.Save must record image/svg+xml,
// reference that artifact in the response, and create only the JSON record alongside the SVG.
//
// Kind: permanent.
func TestArtifactMediaType(t *testing.T) {
	directory := t.TempDir()
	content := []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"><rect width="10" height="10"/></svg>`)

	artifactPath := filepath.Join(directory, "boat.svg")
	if err := os.WriteFile(artifactPath, content, 0o600); err != nil {
		t.Fatalf("💣 SVG fixture: %v", err)
	}

	record := New("example", "model", "test", "boat", time.Now())
	record.SetFields(nil, []BinaryField{{Path: []string{"data"}, MIMEField: "mime"}})
	requestIndex := record.Begin("POST", "https://example.com/images", "application/json", []byte(`{}`))
	body := []byte(`{"data":"` + base64.StdEncoding.EncodeToString(content) + `","mime":"image/svg+xml"}`)
	record.Receive(requestIndex, "synchronous", 200, "application/json", body, nil)

	path, err := record.Save(directory, "boat", []artifact.SavedFile{{Path: artifactPath, Bytes: int64(len(content))}}, nil)
	if err != nil {
		t.Errorf("✗ SVG record persistence: %v", err)
	}

	document := readRecordJSON(t, path)

	files := decodeObjects(t, document["artifacts"])
	if len(files) != 1 {
		t.Fatalf("💣 cannot inspect %d artifact facts", len(files))
	}

	if string(files[0]["content-type"]) != `"image/svg+xml"` {
		t.Errorf("✗ SVG artifact media type changed: %s", files[0]["content-type"])
	}

	responses := decodeObjects(t, document["provider-responses"])
	if len(responses) != 1 || !bytes.Contains(responses[0]["full-response"], []byte(artifactPath)) {
		t.Error("✗ SVG response does not reference its actual artifact")
	}

	directoryFiles, err := os.ReadDir(directory)
	if err != nil || len(directoryFiles) != 2 {
		t.Errorf("✗ expected only SVG and record, found %d files: %v", len(directoryFiles), err)
	}

	if !t.Failed() {
		t.Log("✓ artifact metadata preserves the provider-declared media type")
	}
}

// TestRecordNames verifies invariant #4: Record filenames.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Saving two records with the stem boat-02 must produce boat-02.bild.json and boat-02.bild-02.json.
// The second Record.Save must leave the first record unchanged.
//
// Kind: permanent.
func TestRecordNames(t *testing.T) {
	directory := t.TempDir()
	record := New("example", "model", "test", "first prompt", time.Now())

	firstPath, err := record.Save(directory, "boat-02", nil, nil)
	if err != nil {
		t.Errorf("✗ first save: %v", err)
	}

	firstBytes, readErr := os.ReadFile(firstPath) //nolint:gosec // Inspect this test's generated record.
	if readErr != nil {
		t.Errorf("✗ first record: %v", readErr)
	}

	second := New("example", "model", "test", "second prompt", time.Now())

	secondPath, err := second.Save(directory, "boat-02", nil, nil)
	if err != nil {
		t.Errorf("✗ second save: %v", err)
	}

	if filepath.Base(firstPath) != "boat-02.bild.json" || filepath.Base(secondPath) != "boat-02.bild-02.json" {
		t.Errorf("✗ filenames %q, %q", firstPath, secondPath)
	}

	unchanged, readErr := os.ReadFile(firstPath) //nolint:gosec // Verify this test's earlier record was not overwritten.
	if readErr != nil || !bytes.Equal(firstBytes, unchanged) {
		t.Error("✗ existing record overwritten")
	}

	if !t.Failed() {
		t.Log("✓ record collisions preserve final artifact stems")
	}
}

// readRecordJSON reads one produced record as lossless JSON fields.
func readRecordJSON(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()

	content, err := os.ReadFile(path) //nolint:gosec // Read this test's generated record in its temporary directory.
	if err != nil {
		t.Errorf("✗ record was not saved: %v", err)

		return nil
	}

	return decodeObject(t, content)
}

// decodeObject decodes a JSON object for outcome assertions.
func decodeObject(t *testing.T, content []byte) map[string]json.RawMessage {
	t.Helper()

	var object map[string]json.RawMessage
	if err := json.Unmarshal(content, &object); err != nil {
		t.Errorf("✗ invalid JSON object: %v", err)
	}

	return object
}

// decodeObjects decodes an ordered JSON object array for outcome assertions.
func decodeObjects(t *testing.T, content []byte) []map[string]json.RawMessage {
	t.Helper()

	var objects []map[string]json.RawMessage
	if err := json.Unmarshal(content, &objects); err != nil {
		t.Errorf("✗ invalid JSON array: %v", err)
	}

	return objects
}
