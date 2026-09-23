package metadata

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/media"
)

// Invariants tested:
// 1. Source identity at save time: If a local input changes after Record.Prepared and Record.Begin,
//    Record.Save must retain the submitted bytes in a separate file and reference that file in the
//    request.
// 2. Record persistence failures: Given non-JSON, truncated, and malformed-base64 responses plus a
//    failed request, Record.Save must retain all requests and responses. If retained media exceeds
//    the filename limit but the record name fits, Record.Save must return ENAMETOOLONG and save a
//    record with failed status and no artifacts.
// 3. Record reading: Read must return os.ErrNotExist for a missing file, reject malformed JSON,
//    null, unsupported schema versions, and incomplete version-one records, and return a nonnil
//    record for a file produced by Record.Save.
// 4. Complete transactions: For JSON containing arbitrary text and integers, Record.Begin,
//    Record.Receive, and Record.Save must preserve the request and response values, including
//    nested fields, nulls, exact numbers, and array order.

// TestChangedInputRetention verifies invariant #1: Source identity at save time.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// If a local input changes after Record.Prepared and Record.Begin, Record.Save must retain the
// submitted bytes in a separate file and reference that file in the request. It must leave the
// changed input file untouched.
//
// Kind: permanent.
func TestChangedInputRetention(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "source.bin")
	submitted := []byte("submitted input bytes")
	changed := []byte("changed input bytes")

	if err := os.WriteFile(inputPath, submitted, 0o600); err != nil {
		t.Fatalf("💣 input fixture: %v", err)
	}

	record := New("example", "model", "test", "boat", time.Now())
	record.Prepared([]media.Input{{Filepath: inputPath, Bytes: submitted}})

	fields := []BinaryField{{Path: []string{"image"}}}
	body := []byte(`{"image":"` + base64.StdEncoding.EncodeToString(submitted) + `"}`)

	record.SetFields(fields, nil)
	record.Begin("POST", "https://example.com/images", "application/json", body)

	if err := os.WriteFile(inputPath, changed, 0o600); err != nil {
		t.Fatalf("💣 input mutation: %v", err)
	}

	if _, err := record.Save(t.TempDir(), "boat", nil, nil); err != nil {
		t.Errorf("✗ retained request: %v", err)
	}

	payload := decodeObject(t, record.Request.Calls[0].Payload)

	var marker string
	if err := json.Unmarshal(payload["image"], &marker); err != nil {
		t.Errorf("✗ binary marker: %v", err)
	}

	retainedPath := strings.TrimSuffix(strings.TrimPrefix(marker, "[Bildomat: base64 data redacted for length; saved locally to "), "]")

	retained, err := os.ReadFile(retainedPath) //nolint:gosec // Inspect the retained file produced by this test.
	if err != nil || !bytes.Equal(retained, submitted) || retainedPath == inputPath {
		t.Errorf("✗ changed input replaced submitted bytes: %q, %q, %v", retainedPath, retained, err)
	}

	currentInput, err := os.ReadFile(inputPath) //nolint:gosec // Verify this test's original input remains untouched.
	if err != nil || !bytes.Equal(currentInput, changed) {
		t.Errorf("✗ persistence overwrote the changed input: %q, %v", currentInput, err)
	}

	if !t.Failed() {
		t.Log("✓ retained requests remain accurate when an original input changes")
	}
}

// TestRecordFailures verifies invariant #2: Record persistence failures.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given non-JSON, truncated, and malformed-base64 responses plus a failed request, Record.Save must
// retain all requests and responses. It must preserve the text body, mark the truncated response
// incomplete, mark the binary unavailable, retain the unrelated JSON value, and record persistence
// and transport errors.
//
// Kind: permanent.
func TestRecordFailures(t *testing.T) {
	record := New("example", "model", "test", "test prompt", time.Now())
	record.SetFields(nil, []BinaryField{{Path: []string{"b64_json"}}})
	requestIndex := record.Begin("POST", "https://example.com/job", "application/json", []byte(`{}`))
	record.Receive(requestIndex, "synchronous", 502, "text/plain", []byte("provider temporarily unavailable"), nil)
	pollIndex := record.Begin("GET", "https://example.com/job/1", "", nil)
	record.Receive(pollIndex, "asynchronous", 200, "application/json", []byte(`{"partial":`), os.ErrClosed)
	lastIndex := record.Begin("GET", "https://example.com/job/1", "", nil)
	record.Receive(lastIndex, "asynchronous", 200, "application/json", []byte(`{"b64_json":"!!!","other":42}`), nil)
	failedIndex := record.Begin("GET", "https://example.com/job/1", "", nil)
	record.RequestFailed(failedIndex, os.ErrDeadlineExceeded)
	path, _ := record.Save(t.TempDir(), "failed", nil, os.ErrDeadlineExceeded)
	document := readRecordJSON(t, path)
	responses := decodeObjects(t, document["provider-responses"])

	requests := decodeObjects(t, decodeObject(t, document["request"])["provider-requests"])
	if len(responses) != 3 || len(requests) != 4 {
		t.Fatalf("💣 cannot inspect %d responses/%d requests", len(responses), len(requests))
	}

	if string(responses[0]["full-response"]) != `"provider temporarily unavailable"` {
		t.Error("✗ non-JSON body lost")
	}

	if string(responses[1]["incomplete"]) != "true" {
		t.Error("✗ truncated response claims complete capture")
	}

	if !bytes.Contains(responses[2]["full-response"], []byte("[Bildomat: base64 data unavailable; see persistence-errors]")) || !bytes.Contains(responses[2]["full-response"], []byte("42")) {
		t.Error("✗ binary failure loses truthful response content")
	}

	if len(document["persistence-errors"]) == 0 || len(requests[3]["error"]) == 0 {
		t.Error("✗ missing persistence or transport error")
	}

	if !t.Failed() {
		t.Log("✓ failures remain inspectable without false saved-file claims")
	}
}

// TestRetainedFileFailure verifies invariant #2: Record persistence failures.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// If retained media exceeds the filename limit but the record name fits, Record.Save must return
// ENAMETOOLONG and save a record with failed status and no artifacts. Both binary fields must use
// the unavailable marker and appear in persistence-errors, while unrelated response data remains.
//
// Kind: permanent.
func TestRetainedFileFailure(t *testing.T) {
	record := New("example", "model", "test", "boat", time.Now())
	encoded := base64.StdEncoding.EncodeToString([]byte("retained bytes"))

	record.SetFields([]BinaryField{{Path: []string{"input", "image"}}}, []BinaryField{{Path: []string{"output", "image"}}})
	requestIndex := record.Begin("POST", "https://example.com/images", "application/json", []byte(`{"input":{"image":"`+encoded+`"}}`))
	record.Receive(requestIndex, "synchronous", 200, "application/json", []byte(`{"output":{"image":"`+encoded+`"},"unknown":42}`), nil)

	// The 242-character stem leaves room for .bild.json but not .bild-data.bin under the
	// filesystem's 255-byte filename limit.
	path, err := record.Save(t.TempDir(), strings.Repeat("a", 242), nil, nil)
	if !errors.Is(err, syscall.ENAMETOOLONG) {
		t.Errorf("✗ retained write lost its filename-limit cause: %v", err)
	}

	document := readRecordJSON(t, path)
	if string(document["status"]) != `"failed"` || string(document["artifacts"]) != `[]` {
		t.Error("✗ failed retention claims successful completion or a saved artifact")
	}

	for _, fieldPath := range []string{"input/image", "output/image"} {
		if !bytes.Contains(document["persistence-errors"], []byte(fieldPath)) {
			t.Errorf("✗ failed field %q is absent from persistence errors: %s", fieldPath, document["persistence-errors"])
		}
	}

	for _, recordedBody := range []json.RawMessage{document["request"], document["provider-responses"]} {
		if !bytes.Contains(recordedBody, []byte(BinaryUnavailable)) || bytes.Contains(recordedBody, []byte("saved locally to")) {
			t.Errorf("✗ failed data has an untruthful reference: %s", recordedBody)
		}
	}

	if !bytes.Contains(document["provider-responses"], []byte("42")) {
		t.Error("✗ binary save failure lost unrelated response data")
	}

	if !t.Failed() {
		t.Log("✓ retained-file failures preserve the record and identify every affected field")
	}
}

// TestReadRecord verifies invariant #3: Record reading.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Read must return os.ErrNotExist for a missing file, reject malformed JSON, null, unsupported
// schema versions, and incomplete version-one records, and return a nonnil record for a file
// produced by Record.Save.
//
// Kind: permanent.
func TestReadRecord(t *testing.T) {
	directory := t.TempDir()
	if _, err := Read(filepath.Join(directory, "missing.json")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("✗ missing record cause: %v", err)
	}

	for index, content := range []string{`{`, `null`, `{"schema-version":987}`, `{"schema-version":1}`} {
		path := filepath.Join(directory, string(rune('a'+index))+".json")
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("💣 fixture: %v", err)
		}

		if _, err := Read(path); err == nil {
			t.Errorf("✗ accepted invalid record %s", content)
		}
	}

	record := New("example", "model", "test", "a prompt", time.Now())

	path, err := record.Save(directory, "valid", nil, nil)
	if err != nil {
		t.Errorf("✗ save: %v", err)
	}

	if loaded, err := Read(path); err != nil || loaded == nil {
		t.Errorf("✗ load saved record: %v", err)
	}

	if !t.Failed() {
		t.Log("✓ record reader distinguishes valid records from unusable input")
	}
}

// FuzzRecordJSON verifies invariant #4: Complete transactions.
// Test class: Expanded.
// Test layer: Fuzzing.
// What is being tested:
// For JSON containing arbitrary text and integers, Record.Begin, Record.Receive, and Record.Save
// must preserve the request and response values, including nested fields, nulls, exact numbers, and
// array order. They must leave the original body bytes unchanged.
//
// Kind: permanent.
func FuzzRecordJSON(f *testing.F) {
	f.Add("provider data", int64(9007199254740993))
	f.Add("\"escaped\"\ntext", int64(-9007199254740993))
	f.Fuzz(func(t *testing.T, providerText string, providerNumber int64) {
		originalBody, err := json.Marshal(map[string]any{
			"unknown": []any{nil, json.Number(strconv.FormatInt(providerNumber, 10)), map[string]string{"data": "text: " + providerText}},
		})
		if err != nil {
			t.Fatalf("💣 JSON fixture: %v", err)
		}

		bodyCopy := bytes.Clone(originalBody)
		record := New("example", "model", "test", "boat", time.Now())
		requestIndex := record.Begin("POST", "https://example.com/jobs", "application/json", originalBody)
		record.Receive(requestIndex, "synchronous", 200, "application/json", originalBody, nil)

		path, err := record.Save(t.TempDir(), "boat", nil, nil)
		if err != nil {
			t.Errorf("✗ record persistence: %v", err)
		}

		document := readRecordJSON(t, path)
		responses := decodeObjects(t, document["provider-responses"])

		requests := decodeObjects(t, decodeObject(t, document["request"])["provider-requests"])
		if len(responses) != 1 || len(requests) != 1 {
			t.Fatalf("💣 cannot inspect %d responses and %d requests", len(responses), len(requests))
		}

		var originalValue any

		originalDecoder := json.NewDecoder(bytes.NewReader(originalBody))
		originalDecoder.UseNumber()

		if err := originalDecoder.Decode(&originalValue); err != nil {
			t.Fatalf("💣 original JSON: %v", err)
		}

		for _, retainedBody := range []json.RawMessage{responses[0]["full-response"], requests[0]["payload"]} {
			var retainedValue any

			retainedDecoder := json.NewDecoder(bytes.NewReader(retainedBody))
			retainedDecoder.UseNumber()

			if err := retainedDecoder.Decode(&retainedValue); err != nil || !reflect.DeepEqual(originalValue, retainedValue) {
				t.Errorf("✗ JSON values changed during retention: %s, %v", retainedBody, err)
			}
		}

		if !bytes.Equal(originalBody, bodyCopy) {
			t.Error("✗ persistence changed the provider body")
		}

		if !t.Failed() {
			t.Log("✓ complete JSON values survive request and response retention")
		}
	})
}
