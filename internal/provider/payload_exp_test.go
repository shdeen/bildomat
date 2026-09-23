package provider

// Invariants tested:
//  1. String-style input media encoding: Given no string-style inputs, addInputMedia must leave the
//     request empty.
//  2. Video rejection in the single-or-array style: Given a video URL and the single-or-array
//     style, addInputMedia must return ErrInputMediaSource and leave the empty request body
//     unchanged.
//  3. Failed generation cleanup causes: Given a successful first download, a second HTTP 503, and
//     permission denied while removing the first file, createImageArtifacts must return no
//     artifacts and preserve both ErrTransportStatus and os.ErrPermission.
//  4. Explicit JSON media rejection: Given media with the multipart-parts style in a JSON body,
//     addInputMedia must return an error under ErrInputMedia or ErrProvConfig and leave the empty
//     body unchanged.
//  5. Optional image response fields: Given image responses with unknown fields or wrongly typed
//     optional fields, parseRespImages must retain the expected string-presence flags and MIME
//     value without an error.
//  6. Image response parsing under arbitrary bodies: For arbitrary response bytes with URL
//     downloads disabled, createImageArtifacts must return at least one artifact with an extension
//     and exactly one payload form, or an error under ErrResponse with no artifacts.
//  7. String input media payloads under arbitrary input: Given arbitrary local bytes and zero to
//     ten inputs, addInputMedia with string style must leave an empty body for zero inputs, return
//     a string for one, or return an equally sized string slice for several.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// TestStringMediaStyleHardening verifies invariant #1: String-style input media encoding.
//
// What is being tested:
// Given no string-style inputs, addInputMedia must leave the request empty. jsonBody must preserve
// one video URL as a string; addInputMedia must preserve ten image URLs as an ordered string array.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestStringMediaStyleHardening(t *testing.T) {
	const providerField = "source_image"

	t.Run("empty input leaves the request unchanged", func(t *testing.T) {
		checkEmptyStringMediaInput(t, providerField)

		if !t.Failed() {
			t.Log("✓ empty input leaves the request unchanged")
		}
	})

	t.Run("video input is represented as a bare string", func(t *testing.T) {
		checkStringMediaVideoInput(t, providerField)

		if !t.Failed() {
			t.Log("✓ video input is represented as a bare string")
		}
	})

	t.Run("ten inputs remain ordered strings", func(t *testing.T) {
		checkTenStringMediaInputs(t, providerField)

		if !t.Failed() {
			t.Log("✓ ten inputs remain ordered strings")
		}
	})

	if !t.Failed() {
		t.Log("✓ the string media layout preserves empty, video, and high-count inputs")
	}
}

// TestSingleMediaStyleRejectsVideo verifies invariant #2: Video rejection in the single-or-array
// style.
//
// What is being tested:
// Given a video URL and the single-or-array style, addInputMedia must return ErrInputMediaSource
// and leave the empty request body unchanged.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSingleMediaStyleRejectsVideo(t *testing.T) {
	body := map[string]any{}
	videoInput := []media.Input{{URL: "https://media.example/source.mp4", MIME: "video/mp4"}}

	err := addInputMedia(body, catalog.InputMediaSingle, "image", "images", videoInput)
	if err == nil || !errors.Is(err, errs.ErrInputMediaSource) {
		t.Errorf("✗ single-or-array video error = %v, want input-media source rejection", err)
	}

	if len(body) != 0 {
		t.Errorf("✗ rejected video changed request body: %#v", body)
	}
}

// TestPartialGenerationCleanupCause verifies invariant #3: Failed generation cleanup causes.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a successful first download, a second HTTP 503, and permission denied while removing the
// first file, createImageArtifacts must return no artifacts and preserve both ErrTransportStatus
// and os.ErrPermission. Saving and rereading the record must preserve both responses and the first
// download's exact bytes.
// Kind: permanent.
func TestPartialGenerationCleanupCause(t *testing.T) {
	temporaryDirectory := t.TempDir()
	t.Setenv("TMPDIR", temporaryDirectory)
	t.Cleanup(func() {
		// #nosec G302 -- owner directory permissions establish and restore the test failure.
		if err := os.Chmod(temporaryDirectory, 0o700); err != nil {
			t.Errorf("✗ restore download directory: %v", err)
		}
	})

	mediaBytes := []byte("generated media")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/first" {
			response.Header().Set("Content-Type", "image/png")

			if _, err := response.Write(mediaBytes); err != nil {
				t.Errorf("✗ download fixture write: %v", err)
			}

			return
		}

		// #nosec G302 -- owner directory permissions establish and restore the test failure.
		if err := os.Chmod(temporaryDirectory, 0o500); err != nil {
			t.Errorf("✗ cleanup-failure fixture: %v", err)
		}

		response.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	responseBody, encodeErr := json.Marshal(map[string]any{"data": []map[string]string{{"url": server.URL + "/first"}, {"url": server.URL + "/failed"}}})
	if encodeErr != nil {
		t.Fatalf("💣 generation response fixture: %v", encodeErr)
	}

	record := metadata.New("provider", "model", "test", "boat", time.Now())
	artifacts, generationErr := createImageArtifacts(context.Background(), "provider/model", &catalog.ImageAPI{RespImageURL: true, FallbackExt: ".png"}, http.StatusOK, responseBody, record)

	temporaryFiles, globErr := filepath.Glob(filepath.Join(temporaryDirectory, "bild-dl-*"))
	if globErr != nil || len(temporaryFiles) != 1 {
		t.Fatalf("💣 expected one source protected from removal: %v, %v", temporaryFiles, globErr)
	}

	if probeErr := os.Remove(temporaryFiles[0]); !errors.Is(probeErr, os.ErrPermission) {
		t.Fatalf("💣 fixture did not establish cleanup permission failure: %v", probeErr)
	}

	if len(artifacts) != 0 || !errors.Is(generationErr, errs.ErrTransportStatus) || !errors.Is(generationErr, os.ErrPermission) {
		t.Errorf("✗ failed generation transferred artifacts or lost a cause: %+v, %v", artifacts, generationErr)
	}

	recordPath, persistenceErr := record.Save(t.TempDir(), "boat", nil, generationErr)
	if persistenceErr != nil {
		t.Errorf("✗ failed-generation response could not be retained: %v", persistenceErr)
	} else {
		retained, readErr := metadata.Read(recordPath)
		if readErr != nil || len(retained.Responses) != 2 {
			t.Errorf("✗ received transaction lost: %+v, %v", retained, readErr)
		} else {
			var fileReference map[string]string
			if decodeErr := json.Unmarshal(retained.Responses[0].Body, &fileReference); decodeErr != nil {
				t.Errorf("✗ download file reference unreadable: %v", decodeErr)
			} else {
				// #nosec G304 -- this path belongs to the test temporary directory.
				content, fileErr := os.ReadFile(fileReference["file"])
				if fileErr != nil || !bytes.Equal(content, mediaBytes) {
					t.Errorf("✗ retained download bytes changed: %q, %v", content, fileErr)
				}
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ failed generation transfers no artifacts and retains operation and cleanup causes")
	}
}

// TestJSONRejectsFileParts verifies invariant #4: Explicit JSON media rejection.
//
// What is being tested:
// Given media with the multipart-parts style in a JSON body, addInputMedia must return an error
// under ErrInputMedia or ErrProvConfig and leave the empty body unchanged.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestJSONRejectsFileParts(t *testing.T) {
	body := map[string]any{}

	err := addInputMedia(body, catalog.InputMediaParts, "images", "images", []media.Input{{URL: "https://example.test/reference.png", MIME: "image/png"}})
	if !errors.Is(err, errs.ErrInputMedia) && !errors.Is(err, errs.ErrProvConfig) {
		t.Errorf("✗ JSON file parts were not rejected with context: %v", err)
	}

	if len(body) != 0 {
		t.Errorf("✗ unsupported media changed the request body: %v", body)
	}

	if !t.Failed() {
		t.Log("✓ JSON cannot silently discard file-part inputs")
	}
}

// TestOptionalResponseFields verifies invariant #5: Optional image response fields.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given image responses with unknown fields or wrongly typed optional fields, parseRespImages must
// retain the expected string-presence flags and MIME value without an error. An empty primary MIME
// must override the alternate; absent or null data must return no images, and a null image record
// must return ErrResponseDecode.
// Kind: permanent.
func TestOptionalResponseFields(t *testing.T) {
	for _, responseCase := range []struct {
		name, body, mime           string
		count                      int
		base64Set, urlSet, mimeSet bool
		classification             error
	}{
		{name: "unknown and wrong types", body: `{"extra":true,"data":[{"b64_json":17,"url":"https://media.example/image","mime_type":{},"media_type":"image/jpeg"}]}`, count: 1, urlSet: true, mimeSet: true, mime: "image/jpeg"},
		{name: "empty primary MIME", body: `{"data":[{"b64_json":"","mime_type":"","media_type":"image/jpeg"}]}`, count: 1, base64Set: true, mimeSet: true},
		{name: "all optional wrong", body: `{"data":[{"b64_json":{},"url":[],"mime_type":17,"media_type":false}]}`, count: 1},
		{name: "absent", body: `{}`},
		{name: "null data", body: `{"data":null}`},
		{name: "null record", body: `{"data":[null]}`, classification: errs.ErrResponseDecode},
	} {
		t.Run(responseCase.name, func(t *testing.T) {
			images, err := parseRespImages([]byte(responseCase.body))
			if !errors.Is(err, responseCase.classification) || len(images) != responseCase.count {
				t.Errorf("✗ response count=%d error=%v; want count=%d classification=%v", len(images), err, responseCase.count, responseCase.classification)
			}

			if len(images) == 1 {
				imageResponse := images[0]
				_, base64Set := imageResponse.base64.ValIf()
				_, urlSet := imageResponse.url.ValIf()

				mimeValue, mimeSet := imageResponse.mime.ValIf()
				if base64Set != responseCase.base64Set || urlSet != responseCase.urlSet || mimeSet != responseCase.mimeSet || mimeValue != responseCase.mime {
					t.Errorf("✗ optional string presence or MIME precedence changed: %+v", imageResponse)
				}
			}

			if !t.Failed() {
				t.Log("✓ optional image values retain their tolerated representations")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ image responses preserve optional-field and presence semantics")
	}
}

// FuzzParseImg verifies invariant #6: Image response parsing under arbitrary bodies.
//
// What is being tested:
// For arbitrary response bytes with URL downloads disabled, createImageArtifacts must return at
// least one artifact with an extension and exactly one payload form, or an error under ErrResponse
// with no artifacts.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzParseImg(f *testing.F) {
	for _, s := range []string{
		`{"data":[{"b64_json":"AAAA"}]}`, `{}`, `not json`,
		`{"data":[]}`, `{"error":{"message":"x"}}`, `{"data":[{"url":"u"}]}`,
		`{"data":[{"b64_json":"AAAA","media_type":"image/svg+xml"}]}`,
		`{"data":[{"b64_json":""},{"b64_json":"AAAA"}]}`,
		`{"data":[{"b64_json":"AAAA","mime_type":""}]}`,
	} {
		f.Add([]byte(s))
	}

	api := catalog.ImageAPI{FallbackExt: ".png"}

	f.Fuzz(func(t *testing.T, body []byte) {
		artifacts, err := createImageArtifacts(context.Background(), "fuzz (m)", &api, 200, body, nil)
		fuzzVerdict(t, artifacts, err)
	})
}

// FuzzStringMediaStylePayload verifies invariant #7: String input media payloads under arbitrary
// input.
//
// What is being tested:
// Given arbitrary local bytes and zero to ten inputs, addInputMedia with string style must leave an
// empty body for zero inputs, return a string for one, or return an equally sized string slice for
// several. If the first input has a frame marker, it must return ErrInputMediaTime.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzStringMediaStylePayload(f *testing.F) {
	f.Add([]byte("seed-image"), uint8(0), false)
	f.Add([]byte("seed-image"), uint8(1), false)
	f.Add([]byte("seed-image"), uint8(10), false)
	f.Add([]byte("seed-image"), uint8(3), true)

	f.Fuzz(func(t *testing.T, mediaBytes []byte, inputCountByte uint8, frameRequested bool) {
		const providerField = "source_image"

		inputCount := int(inputCountByte % 11)

		mediaInputs := make([]media.Input, inputCount)
		for mediaIndex := range mediaInputs {
			mediaInputs[mediaIndex] = media.Input{Bytes: mediaBytes, MIME: "image/png"}
		}

		if frameRequested && inputCount > 0 {
			mediaInputs[0].FrameAnchor = media.FrameFirst
		}

		requestBody := map[string]any{}
		err := addInputMedia(requestBody, catalog.InputMediaString, providerField, "images", mediaInputs)

		if frameRequested && inputCount > 0 {
			if !errors.Is(err, errs.ErrInputMediaTime) {
				t.Fatalf("frame-marked input error = %v, want input-media time classification", err)
			}
		} else {
			if err != nil {
				t.Fatalf("string media assembly returned an unclassified error: %v", err)
			}

			checkFuzzMediaBody(t, requestBody, providerField, mediaInputs)
		}

		if !t.Failed() {
			t.Log("✓ arbitrary media retains the string layout or the shared frame rejection")
		}
	})
}

// checkEmptyStringMediaInput verifies that an empty input list does not change the request.
func checkEmptyStringMediaInput(t *testing.T, providerField string) {
	t.Helper()

	requestBody := map[string]any{}

	if err := addInputMedia(requestBody, catalog.InputMediaString, providerField, "images", nil); err != nil {
		t.Fatalf("💣 empty string media assembly failed: %v", err)
	}

	if len(requestBody) != 0 {
		t.Errorf("✗ empty media input changed request body: %#v", requestBody)
	}
}

// checkStringMediaVideoInput verifies that video input retains its URL in the string layout.
func checkStringMediaVideoInput(t *testing.T, providerField string) {
	t.Helper()

	mediaURL := "https://media.example/reference.mp4"

	requestBody, err := jsonBody(
		&catalog.ImageAPI{InputMediaProvParam: providerField, InputMediaStyle: catalog.InputMediaString},
		&catalog.Model{ID: "fixture-model", Params: []params.Definition{{FlagID: params.FlagTypeInputMedia}}},
		"fixture prompt",
		nil,
		[]media.Input{{URL: mediaURL, MIME: "video/mp4"}},
	)
	if err != nil {
		t.Fatalf("💣 video string media assembly failed: %v", err)
	}

	if requestBody[providerField] != mediaURL {
		t.Errorf("✗ video field = %#v, want the unchanged URL", requestBody[providerField])
	}
}

// checkTenStringMediaInputs verifies that ten inputs retain their order in the string layout.
func checkTenStringMediaInputs(t *testing.T, providerField string) {
	t.Helper()

	mediaInputs := make([]media.Input, 10)
	mediaStrings := make([]string, 10)

	for mediaIndex := range mediaInputs {
		mediaInputs[mediaIndex] = media.Input{URL: fmt.Sprintf("https://media.example/reference-%d.png", mediaIndex), MIME: "image/png"}
		mediaStrings[mediaIndex] = mediaInputs[mediaIndex].DataURI()
	}

	requestBody := map[string]any{}
	if err := addInputMedia(requestBody, catalog.InputMediaString, providerField, "images", mediaInputs); err != nil {
		t.Fatalf("💣 ten-input string media assembly failed: %v", err)
	}

	if !reflect.DeepEqual(requestBody[providerField], mediaStrings) {
		t.Errorf("✗ configured field = %#v, want ten ordered strings", requestBody[providerField])
	}
}

// checkFuzzMediaBody checks that zero inputs produce an empty body, one produces a string, and
// several produce a string slice of the same length.
func checkFuzzMediaBody(t *testing.T, requestBody map[string]any, providerField string, mediaInputs []media.Input) {
	t.Helper()

	if len(mediaInputs) == 0 {
		if len(requestBody) != 0 {
			t.Fatalf("empty media input changed request body: %#v", requestBody)
		}

		return
	}

	if len(mediaInputs) == 1 {
		if _, stringPresent := requestBody[providerField].(string); !stringPresent {
			t.Fatalf("one input produced %#v, want a string", requestBody[providerField])
		}

		return
	}

	mediaStrings, stringsPresent := requestBody[providerField].([]string)
	if !stringsPresent || len(mediaStrings) != len(mediaInputs) {
		t.Fatalf("%d inputs produced %#v, want an equally sized string array", len(mediaInputs), requestBody[providerField])
	}
}

// fuzzVerdict requires either ErrResponse with no artifacts or at least one artifact with an
// extension and exactly one data source.
func fuzzVerdict(t *testing.T, artifacts []artifact.Media, err error) {
	t.Helper()

	if err != nil {
		if !errors.Is(err, errs.ErrResponse) {
			t.Errorf("✗ unclassified normalization error: %v", err)
		}

		if len(artifacts) != 0 {
			t.Errorf("✗ an error return carried %d artifacts", len(artifacts))
		}

		if !t.Failed() {
			t.Log("✓ classified error, no artifacts")
		}

		return
	}

	if len(artifacts) == 0 {
		t.Errorf("✗ nil error with zero artifacts — the traversal must yield artifacts or a classified error")
	}

	for _, a := range artifacts {
		if (len(a.Data) > 0) == (a.TmpPath != "") || a.FileExt == "" {
			t.Errorf("✗ artifact violates exclusive carriage: %+v", a)
		}
	}

	if !t.Failed() {
		t.Logf("✓ %d artifacts, all exclusively carried", len(artifacts))
	}
}
