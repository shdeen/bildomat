package provider

// Invariants tested:
//  1. Multipart URL reference: Given a JPEG URL and multipart image-edit settings, submitImage must
//     download it once by GET without Authorization and upload one file part with the exact JPEG
//     bytes and image/jpeg type.
//  2. Multipart request assembly without references: Given multipart settings but no references,
//     submitImage must use /gen and send exactly model, prompt, size, and output_format as JSON,
//     preserving the expected model, prompt, 1024x1536 size, and webp format.
//  3. Multipart request assembly with one reference: Given one multipart reference, submitImage
//     must use /refs with exactly the expected text-field names, n=2, and output_format=png.
//  4. Single-reference request assembly without media: Given single-or-array settings without
//     references, submitImage must use /gen and send exactly the expected JSON keys with
//     aspect_ratio=16:9, resolution=2k, and fixed response_format=b64_json.
//  5. Single-reference request assembly with media: Given one single-or-array reference,
//     submitImage must use /refs, send the exact image object under image, and include model,
//     prompt, and fixed response_format with no extra keys.
//  6. JSON request assembly with multiple references: Given three single-or-array references,
//     submitImage must send their exact objects in order under images and include only model,
//     prompt, and fixed response_format alongside them.
//  7. JSON request image count: Given image count two, submitImage must send n as JSON number 2
//     with exactly the model, prompt, n, and response_format keys.
//  8. Nested input media request assembly: Given zero, one, or three references with nested-media
//     settings, submitImage must use /gen.
//  9. Nested response normalization: Given the listed image response shapes, submitImage must
//     decode the expected bytes in order and choose each extension from declared MIME, detected
//     content, or fallback.
//  10. Multipart reference order: Given three multipart references, submitImage must place their
//      exact bytes in source order under the repeated provider field, with only model and prompt as
//      text fields.
//  11. Fixed provider field separation: Given fixed response_format=b64_json and an adjusted aspect
//      input, submitImage must send the fixed field while leaving both that key and value absent
//      from the adjusted parameter map.
//  12. Static MIME fallback during response normalization: Given unidentifiable inline bytes and
//      requested JPEG output, submitImage must return one artifact with the configured .png
//      fallback extension.
//  13. Mixed response normalization: Given an inline JPEG followed by a URL response, submitImage
//      must return two artifacts in that order: the exact inline bytes with .jpg and no temporary
//      path, then a temporary-file-backed .png with no inline bytes.
//  14. Image endpoint selection: Given the fixture request layouts, submitImage must use /gen
//      without multipart references, /refs for multipart or single-or-array references, and /gen
//      for nested references.
//  15. Image submission results: Given a valid inline image response, submitImage must return one
//      nonempty bytes-backed .png artifact.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestFormURLReference verifies invariant #1: Multipart URL reference.
//
// What is being tested:
// Given a JPEG URL and multipart image-edit settings, submitImage must download it once by GET
// without Authorization and upload one file part with the exact JPEG bytes and image/jpeg type.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFormURLReference(t *testing.T) {
	jpegData := jpegStub(t)
	mediaRecorder, mediaServer := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(jpegData)
	})
	mediaURL := mediaServer.URL + "/Crane-Folding-Instructions_1.jpg"
	mediaInput := media.Input{URL: mediaURL, MIME: "image/jpeg"}

	out := driveSub(t, multipartImage, []media.Input{mediaInput}, params.Values{}, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	form := parseForm(t, req)

	parts := form.File[out.api.InputMediaProvParam]
	if len(parts) != 1 {
		t.Fatalf("💣 %d media parts, want 1", len(parts))
	}

	if got := partBytes(t, parts[0]); !bytes.Equal(got, jpegData) {
		t.Errorf("✗ uploaded media = %x, want exact downloaded JPEG %x", got, jpegData)
	}

	if got := parts[0].Header.Get("Content-Type"); got != "image/jpeg" {
		t.Errorf("✗ uploaded media type = %q, want %q", got, "image/jpeg")
	}

	if mediaRecorder.count() != 1 {
		t.Errorf("✗ media download count = %d, want 1", mediaRecorder.count())
	} else {
		downloadRequest := mediaRecorder.last(t)
		if downloadRequest.reqMethod != http.MethodGet {
			t.Errorf("✗ media download method = %q, want GET", downloadRequest.reqMethod)
		}

		if got := downloadRequest.header.Get("Authorization"); got != "" {
			t.Errorf("✗ media download carried Authorization %q, want none", got)
		}
	}

	if !t.Failed() {
		t.Log("✓ a URL reference is downloaded bare and uploaded as the exact multipart media file")
	}
}

// TestAsmPartsGen verifies invariant #2: Multipart request assembly without references.
//
// What is being tested:
// Given multipart settings but no references, submitImage must use /gen and send exactly model,
// prompt, size, and output_format as JSON, preserving the expected model, prompt, 1024x1536 size,
// and webp format.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmPartsGen(t *testing.T) {
	gp := params.Values{}
	gp[params.FlagTypeSize] = "1024x1536"
	gp[params.FlagTypeOutputFormat] = "webp"
	out := driveSub(t, multipartImage, nil, gp, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	if req.reqPath != "/gen" {
		t.Errorf("✗ path = %q, want /gen", req.reqPath)
	}

	body := jsonKeys(t, req.body, "model", "prompt", "size", "output_format")
	if body["model"] != out.model.ID || body["prompt"] != "a prompt" || body["size"] != "1024x1536" {
		t.Errorf("✗ body values = %v", body)
	}

	if body["output_format"] != "webp" {
		t.Errorf("✗ output_format = %v, want webp", body["output_format"])
	}

	if !t.Failed() {
		t.Log("✓ zero references are sent as JSON to GenURL with exact keys and no response_format")
	}
}

// TestAsmPartsRefs verifies invariant #3: Multipart request assembly with one reference.
//
// What is being tested:
// Given one multipart reference, submitImage must use /refs with exactly the expected text-field
// names, n=2, and output_format=png. The configured file field must contain the reference's exact
// bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmPartsRefs(t *testing.T) {
	imgs := makeInputMedia(t, 1)
	gp := params.Values{}
	gp[params.FlagTypeQuality] = "low"
	gp[params.FlagTypeImageN] = 2
	gp[params.FlagTypeOutputFormat] = "png"
	out := driveSub(t, multipartImage, imgs, gp, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	if req.reqPath != "/refs" {
		t.Errorf("✗ path = %q, want /refs", req.reqPath)
	}

	form := parseForm(t, req)
	formKeys(t, form, "model", "prompt", "quality", "n", "output_format")

	if v := form.Value["n"]; len(v) != 1 || v[0] != "2" {
		t.Errorf("✗ form n = %v, want [2]", v)
	}

	if v := form.Value["output_format"]; len(v) != 1 || v[0] != "png" {
		t.Errorf("✗ form output_format = %v, want [png]", v)
	}

	checkParts(t, form, out.api.InputMediaProvParam, imgs)

	if !t.Failed() {
		t.Log("✓ one reference is sent as form parts to InputMediaURL with exact form fields and no response_format")
	}
}

// TestAsmSingleGen verifies invariant #4: Single-reference request assembly without media.
//
// What is being tested:
// Given single-or-array settings without references, submitImage must use /gen and send exactly the
// expected JSON keys with aspect_ratio=16:9, resolution=2k, and fixed response_format=b64_json.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmSingleGen(t *testing.T) {
	gp := params.Values{}
	gp[params.FlagTypeAspect] = "16:9"
	gp[params.FlagTypeResolution] = "2k"
	out := driveSub(t, singleOrArrayImage, nil, gp, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	if req.reqPath != "/gen" {
		t.Errorf("✗ path = %q, want /gen", req.reqPath)
	}

	body := jsonKeys(t, req.body, "model", "prompt", "aspect_ratio", "resolution", "response_format")
	if body["aspect_ratio"] != "16:9" {
		t.Errorf("✗ aspect_ratio = %v, want 16:9", body["aspect_ratio"])
	}

	if body["resolution"] != "2k" {
		t.Errorf("✗ resolution = %v, want 2k", body["resolution"])
	}

	checkFixed(t, body)

	if !t.Failed() {
		t.Log("✓ a no-reference single-or-array generation carries exact keys and the fixed field")
	}
}

// TestAsmSingleRef verifies invariant #5: Single-reference request assembly with media.
//
// What is being tested:
// Given one single-or-array reference, submitImage must use /refs, send the exact image object
// under image, and include model, prompt, and fixed response_format with no extra keys.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmSingleRef(t *testing.T) {
	imgs := makeInputMedia(t, 1)
	out := driveSub(t, singleOrArrayImage, imgs, params.Values{}, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	if req.reqPath != "/refs" {
		t.Errorf("✗ path = %q, want /refs", req.reqPath)
	}

	body := jsonKeys(t, req.body, "model", "prompt", "image", "response_format")
	if !reflect.DeepEqual(body["image"], any(inputMediaObject(t, imgs[0], false))) {
		t.Errorf("✗ image object = %v, want the exact %s form", body["image"], catalog.InputMediaSingle)
	}

	checkFixed(t, body)

	if !t.Failed() {
		t.Log("✓ one reference is a single object on the references route, fixed field included")
	}
}

// TestAsmArrayRefs verifies invariant #6: JSON request assembly with multiple references.
//
// What is being tested:
// Given three single-or-array references, submitImage must send their exact objects in order under
// images and include only model, prompt, and fixed response_format alongside them.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmArrayRefs(t *testing.T) {
	imgs := makeInputMedia(t, 3)
	out := driveSub(t, singleOrArrayImage, imgs, params.Values{}, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	body := jsonKeys(t, req.body, "model", "prompt", "images", "response_format")
	if !reflect.DeepEqual(body["images"], any(inputMediaList(t, imgs, false))) {
		t.Errorf("✗ images list = %v, want the three ordered objects", body["images"])
	}

	checkFixed(t, body)

	if !t.Failed() {
		t.Log("✓ several references are sent under the pluralized field name in order, fixed field included")
	}
}

// TestAsmJSONCount verifies invariant #7: JSON request image count.
//
// What is being tested:
// Given image count two, submitImage must send n as JSON number 2 with exactly the model, prompt,
// n, and response_format keys.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmJSONCount(t *testing.T) {
	gp := params.Values{}
	gp[params.FlagTypeImageN] = 2
	out := driveSub(t, singleOrArrayImage, nil, gp, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	body := jsonKeys(t, req.body, "model", "prompt", "n", "response_format")
	if body["n"] != float64(2) {
		t.Errorf("✗ n = %v (%T), want the JSON number 2", body["n"], body["n"])
	}

	if !t.Failed() {
		t.Log("✓ a submitted count is sent in the JSON body as the integer n")
	}
}

// TestAsmNested verifies invariant #8: Nested input media request assembly.
//
// What is being tested:
// Given zero, one, or three references with nested-media settings, submitImage must use /gen. It
// must send only model and prompt when empty, or add the exact ordered nested reference objects
// under the configured field.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmNested(t *testing.T) {
	for _, refs := range []int{0, 1, 3} {
		t.Run(fmt.Sprintf("%d refs", refs), func(t *testing.T) {
			imgs := makeInputMedia(t, refs)
			out := driveSub(t, nestedImage, imgs, params.Values{}, artOK(t))

			req, ok := okReq(t, out)
			if !ok {
				return
			}

			if req.reqPath != "/gen" {
				t.Errorf("✗ path = %q, want /gen (no references endpoint exists)", req.reqPath)
			}

			if refs == 0 {
				jsonKeys(t, req.body, "model", "prompt")

				return
			}

			body := jsonKeys(t, req.body, "model", "prompt", out.api.InputMediaProvParam)
			if !reflect.DeepEqual(body[out.api.InputMediaProvParam], any(inputMediaList(t, imgs, true))) {
				t.Errorf("✗ %s = %v, want the ordered nested objects", out.api.InputMediaProvParam, body[out.api.InputMediaProvParam])
			}

			if !t.Failed() {
				t.Logf("✓ %s", fmt.Sprintf("%d refs", refs))
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the nested style is sent in the generation body in order, absent at zero references")
	}
}

// TestNormWalk verifies invariant #9: Nested response normalization.
//
// What is being tested:
// Given the listed image response shapes, submitImage must decode the expected bytes in order and
// choose each extension from declared MIME, detected content, or fallback. Empty data, invalid
// encoding, and failed status cases must return their expected error category, name the
// provider/model label, and return no artifacts.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestNormWalk(t *testing.T) {
	for _, c := range normCases(t) {
		t.Run(c.name, func(t *testing.T) {
			out := driveSub(t, c.layout, nil, params.Values{}, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				fmt.Fprint(w, c.body)
			})
			if c.want != nil {
				checkNormFail(t, out.label, out.artifacts, out.err, c.want)

				return
			}

			checkNormOK(t, c, out)

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ the normalization traversal converts, classifies, and labels exactly as required")
	}
}

// TestAsmPartsOrder verifies invariant #10: Multipart reference order.
//
// What is being tested:
// Given three multipart references, submitImage must place their exact bytes in source order under
// the repeated provider field, with only model and prompt as text fields.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAsmPartsOrder(t *testing.T) {
	imgs := makeInputMedia(t, 3)
	out := driveSub(t, multipartImage, imgs, params.Values{}, artOK(t))

	req, ok := okReq(t, out)
	if !ok {
		return
	}

	form := parseForm(t, req)
	formKeys(t, form, "model", "prompt")
	checkParts(t, form, out.api.InputMediaProvParam, imgs)

	if !t.Failed() {
		t.Log("✓ three references are sent as ordered parts under the repeated provider field")
	}
}

// TestFixedNotSent verifies invariant #11: Fixed provider field separation.
//
// What is being tested:
// Given fixed response_format=b64_json and an adjusted aspect input, submitImage must send the
// fixed field while leaving both that key and value absent from the adjusted parameter map.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestFixedNotSent(t *testing.T) {
	d := fixtureProvider(t, singleOrArrayImage)
	m := fixtureModel(t, d)
	recorder, srv := recSrv(t, artOK(t))
	api := imgAPI(t, d, srv.URL+"/gen", srv.URL+"/refs")

	gp, _, adjustmentErr := params.Adjust(params.FlagInputs{params.FlagTypeAspect: "16:9"}, m.Params, m.ID)
	if adjustmentErr != nil {
		t.Errorf("✗ parameter adjustment failed: %v", adjustmentErr)
	}

	run := imgReq(t, d, m, "a prompt", gp, nil)
	if _, err := submitImage(t.Context(), &api, "fixture-key", &run); err != nil {
		t.Fatalf("💣 submit failed: %v", err)
	}

	var wire map[string]any
	if json.Unmarshal(recorder.last(t).body, &wire) != nil || wire["response_format"] != "b64_json" {
		t.Errorf("✗ the request lacks the fixed field: %v", wire)
	}
	// The adjusted map is the display's one source, so a fixed pair absent from it can never
	// render.
	for param, paramVal := range gp {
		if param == "response_format" || paramVal == "b64_json" {
			t.Errorf("✗ FixedProvFields leaked into the adjusted values: %s = %v", param, paramVal)
		}
	}

	if !t.Failed() {
		t.Log("✓ the fixed field is sent in the request but never appears in the adjusted values")
	}
}

// TestNormFallbackStatic verifies invariant #12: Static MIME fallback during response
// normalization.
//
// What is being tested:
// Given unidentifiable inline bytes and requested JPEG output, submitImage must return one artifact
// with the configured .png fallback extension.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestNormFallbackStatic(t *testing.T) {
	gp := params.Values{}
	gp[params.FlagTypeOutputFormat] = "jpeg" // the conflicting requested format — must not decide the fallback
	body := fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, b64(t, []byte("plain text payload")))

	out := driveSub(t, multipartImage, nil, gp, textOK(t, body))
	if out.err != nil {
		t.Errorf("✗ submit failed: %v", out.err)

		return
	}

	if len(out.artifacts) != 1 {
		t.Errorf("✗ %d artifacts, want 1", len(out.artifacts))

		return
	}

	if out.artifacts[0].FileExt != ".png" {
		t.Errorf("✗ FileExt = %q, want the declared .png fallback — never a value derived from the requested output format", out.artifacts[0].FileExt)
	}

	if !t.Failed() {
		t.Log("✓ an unidentifiable result takes the declared static fallback over the requested format")
	}
}

// TestNormMixed verifies invariant #13: Mixed response normalization.
//
// What is being tested:
// Given an inline JPEG followed by a URL response, submitImage must return two artifacts in that
// order: the exact inline bytes with .jpg and no temporary path, then a temporary-file-backed .png
// with no inline bytes.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestNormMixed(t *testing.T) {
	_, dlSrv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		fmt.Fprint(w, "URLBYTES")
	})
	body := fmt.Sprintf(`{"data":[{"b64_json":%q},{"url":%q}]}`, b64(t, jpegStub(t)), dlSrv.URL+"/img")

	out := driveSub(t, singleOrArrayImage, nil, params.Values{}, textOK(t, body))
	for _, a := range out.artifacts {
		rmArtifact(t, a)
	}

	if out.err != nil {
		t.Errorf("✗ submit failed: %v", out.err)

		return
	}

	if len(out.artifacts) != 2 {
		t.Errorf("✗ %d artifacts, want 2 in response order", len(out.artifacts))

		return
	}

	first, second := out.artifacts[0], out.artifacts[1]
	if !bytes.Equal(first.Data, jpegStub(t)) || first.TmpPath != "" || first.FileExt != ".jpg" {
		t.Errorf("✗ artifact 0 = (%d bytes, %q, %q), want the inline bytes-backed .jpg first", len(first.Data), first.TmpPath, first.FileExt)
	}

	if second.TmpPath == "" || len(second.Data) != 0 || second.FileExt != ".png" {
		t.Errorf("✗ artifact 1 = (%d bytes, %q, %q), want the streamed file-backed .png second", len(second.Data), second.TmpPath, second.FileExt)
	}

	checkCarriage(t, "mixed inline", first)
	checkCarriage(t, "mixed url", second)

	if !t.Failed() {
		t.Log("✓ a mixed inline/URL response yields both carriages in response order")
	}
}

// TestImageEndpointSelection verifies invariant #14: Image endpoint selection.
//
// What is being tested:
// Given the fixture request layouts, submitImage must use /gen without multipart references, /refs
// for multipart or single-or-array references, and /gen for nested references. The nested case must
// include references under input_references.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestImageEndpointSelection(t *testing.T) {
	cases := []struct {
		name            string
		layout          string
		inputMediaCount int
		wantPath        string
		inBody          string // body key the references must be sent under ("" = none)
	}{
		{"multipart image 0 refs to GenURL", multipartImage, 0, "/gen", ""},
		{"multipart image refs to InputMediaURL", multipartImage, 1, "/refs", ""},
		{"single-or-array image refs to InputMediaURL", singleOrArrayImage, 1, "/refs", ""},
		{"nested image refs in the GenURL body", nestedImage, 1, "/gen", "input_references"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := fixtureProvider(t, c.layout)
			m := fixtureModel(t, d)
			recorder, srv := recSrv(t, artOK(t))

			api := imgAPI(t, d, srv.URL+"/gen", srv.URL+"/refs")

			run := imgReq(t, d, m, "p", params.Values{}, makeInputMedia(t, c.inputMediaCount))
			if _, err := submitImage(t.Context(), &api, "k", &run); err != nil {
				t.Errorf("✗ submit failed: %v", err)

				return
			}

			req := recorder.last(t)
			if req.reqPath != c.wantPath {
				t.Errorf("✗ recorded path = %q, want %q", req.reqPath, c.wantPath)
			}

			if c.inBody != "" {
				var body map[string]any
				if json.Unmarshal(req.body, &body) != nil || body[c.inBody] == nil {
					t.Errorf("✗ references were not sent in the %s body under %q", c.wantPath, c.inBody)
				}
			}

			if !t.Failed() {
				t.Logf("✓ %s", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ submitImage routes by GenURL/InputMediaURL/in-body exactly as described")
	}
}

// TestImgSubmitRun verifies invariant #15: Image submission results.
//
// What is being tested:
// Given a valid inline image response, submitImage must return one nonempty bytes-backed .png
// artifact. Given HTTP 400 with a provider message, it must return ErrResponseStatus and name the
// provider/model label, status, and message.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestImgSubmitRun(t *testing.T) {
	d := fixtureProvider(t, multipartImage)
	m := fixtureModel(t, d)

	t.Run("valid data yields a bytes-backed artifact", func(t *testing.T) {
		body := fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, b64(t, pngStub(t)))
		_, srv := recSrv(t, textOK(t, body))
		api := imgAPI(t, d, srv.URL+"/gen", srv.URL+"/refs")
		run := imgReq(t, d, m, "a prompt", params.Values{}, nil)

		artifacts, err := submitImage(t.Context(), &api, "fixture-key", &run)
		if err != nil {
			t.Fatalf("💣 run failed: %v", err)
		}

		if len(artifacts) != 1 {
			t.Fatalf("💣 %d artifacts, want 1 — nothing to inspect", len(artifacts))
		}

		generatedMedia := artifacts[0]
		if len(generatedMedia.Data) == 0 || generatedMedia.TmpPath != "" || generatedMedia.FileExt != ".png" {
			t.Errorf("✗ artifact = (%d bytes, %q, %q), want bytes-backed .png", len(generatedMedia.Data), generatedMedia.TmpPath, generatedMedia.FileExt)
		}

		if !t.Failed() {
			t.Log("✓ valid data yields a bytes-backed artifact")
		}
	})

	t.Run("4xx surfaces the labeled APIErr", func(t *testing.T) {
		_, srv := recSrv(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `{"error":{"message":"quota exhausted"}}`)
		})
		api := imgAPI(t, d, srv.URL+"/gen", srv.URL+"/refs")
		run := imgReq(t, d, m, "p", params.Values{}, nil)

		_, err := submitImage(t.Context(), &api, "k", &run)
		if !errors.Is(err, errs.ErrResponseStatus) {
			t.Errorf("✗ err = %v, want ErrResponseStatus", err)
		}

		for _, want := range []string{pairLabel(d, m), "quota exhausted", "400"} {
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("✗ error %v missing %q", err, want)
			}
		}

		if !t.Failed() {
			t.Log("✓ 4xx surfaces the labeled APIErr")
		}
	})

	if !t.Failed() {
		t.Log("✓ the descriptor image submit generates and labels its API errors")
	}
}

// subOut holds a submitImage result, its captured request, and the provider/model fixture.
type subOut struct {
	req       recReq
	artifacts []artifact.Media
	err       error
	api       catalog.ImageAPI
	model     catalog.Model
	label     string
}

// wantArt is one expected normalized artifact: its bytes and extension.
type wantArt struct {
	data []byte
	ext  string
}

// normCase pairs an image response with its expected artifacts or error.
type normCase struct {
	name      string
	layout    string
	status    int
	body      string
	artifacts []wantArt
	want      error // nil = success
}

// The fixture provider config documents, one per request layout. Each declares the fixture
// provider, one model named after the layout, and the API section the layout exercises.
//   - multipartImage: image references travel as multipart file parts to the input-media endpoint
//   - singleOrArrayImage: one image reference is a single JSON object, several are an array, with a
//     fixed provider field on every request and response URLs downloaded
//   - nestedImage: image references are nested objects in the generation body
//   - contentEndpointVideo: a multipart start whose local images are resized to the request size,
//     and a download from the job's authenticated content route
//   - bareURLVideo: a JSON start at a separate start path, a required URL in the poll body, and a
//     download without a credential
//   - sameOriginVideo: a JSON start with nested references, an optional URL in the poll body, and
//     an authenticated fallback to the content route
const (
	multipartImage       = "multipart-image"
	singleOrArrayImage   = "single-or-array-image"
	nestedImage          = "nested-image"
	contentEndpointVideo = "content-endpoint-video"
	bareURLVideo         = "bare-url-video"
	sameOriginVideo      = "same-origin-video"
)

// multipartImageDoc declares image generation with multipart editing inputs.
const multipartImageDoc = `{
	"id": "fixture-provider",
	"displayName": "Fixture Provider",
	"apiKeyEnvVar": "FIXTURE_API_KEY",
	"models": [
		{
			"id": "multipart-image",
			"name": "Multipart Image",
			"media": "image",
			"params": [
				{"paramID": "quality", "flagID": "quality", "allowedValues": ["low", "medium", "high"]},
				{"paramID": "n", "flagID": "num-images", "minValue": 1, "maxValue": 10},
				{"paramID": "output_format", "flagID": "output-format", "allowedValues": ["png", "jpeg", "webp"]},
				{"flagID": "input-media", "maxMultiple": 16},
				{"paramID": "size", "flagID": "size", "allowedValues": ["1024x1024", "1024x1536", "1536x1024"]}
			]
		}
	],
	"config": {
		"imageAPI": {
			"genURL": "https://fixture.example/images/generations",
			"inputMediaURL": "https://fixture.example/images/edits",
			"inputMediaPayloadType": "form",
			"inputMediaProvParam": "image[]",
			"inputMediaStyle": "parts",
			"fallbackExt": ".png"
		}
	}
}`

// singleOrArrayImageDoc declares image requests with singular or plural media objects.
const singleOrArrayImageDoc = `{
	"id": "fixture-provider",
	"displayName": "Fixture Provider",
	"apiKeyEnvVar": "FIXTURE_API_KEY",
	"models": [
		{
			"id": "single-or-array-image",
			"name": "Single Or Array Image",
			"media": "image",
			"params": [
				{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["1:1", "16:9", "9:16"]},
				{"paramID": "resolution", "flagID": "resolution", "allowedValues": ["1k", "2k"]},
				{"paramID": "n", "flagID": "num-images", "minValue": 1, "maxValue": 10},
				{"flagID": "input-media", "maxMultiple": 3}
			]
		}
	],
	"config": {
		"imageAPI": {
			"genURL": "https://fixture.example/images/generations",
			"inputMediaURL": "https://fixture.example/images/edits",
			"inputMediaPayloadType": "json",
			"inputMediaProvParam": "image",
			"inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
			"fixedProvFields": {"response_format": "b64_json"},
			"fallbackExt": ".jpg",
			"respImageURL": true
		}
	}
}`

// nestedImageDoc declares image requests with nested typed media references.
const nestedImageDoc = `{
	"id": "fixture-provider",
	"displayName": "Fixture Provider",
	"apiKeyEnvVar": "FIXTURE_API_KEY",
	"models": [
		{
			"id": "nested-image",
			"name": "Nested Image",
			"media": "image",
			"params": [
				{"flagID": "input-media", "maxMultiple": 1}
			]
		}
	],
	"config": {
		"imageAPI": {
			"genURL": "https://fixture.example/images",
			"inputMediaPayloadType": "json",
			"inputMediaProvParam": "input_references",
			"inputMediaStyle": "nested",
			"fallbackExt": ".png"
		}
	}
}`

// contentEndpointVideoDoc declares video jobs with an authenticated content endpoint.
const contentEndpointVideoDoc = `{
	"id": "fixture-provider",
	"displayName": "Fixture Provider",
	"apiKeyEnvVar": "FIXTURE_API_KEY",
	"models": [
		{
			"id": "content-endpoint-video",
			"name": "Content Endpoint Video",
			"media": "video",
			"params": [
				{"paramID": "seconds", "flagID": "duration", "allowedValues": ["4", "8"]},
				{"flagID": "input-media", "maxMultiple": 1},
				{"paramID": "size", "flagID": "size", "allowedValues": ["1280x720", "720x1280"]}
			]
		}
	],
	"config": {
		"stringParams": ["duration"],
		"videoAPI": {
			"asyncJobsURL": "https://fixture.example/videos",
			"jobIDField": "id",
			"progressStatusText": ["in_progress"],
			"completedStatusText": "completed",
			"failedStatusText": ["failed"],
			"urlContentPath": "/content",
			"authDownload": true,
			"inputMediaPayloadType": "form",
			"inputMediaProvParam": "input_reference",
			"inputMediaStyle": "parts",
			"inputMediaMustResize": true,
			"fallbackExt": ".mp4",
			"pollInterval": 1,
			"pollTimeout": 60
		}
	}
}`

// bareURLVideoDoc declares video jobs with unauthenticated result URLs.
const bareURLVideoDoc = `{
	"id": "fixture-provider",
	"displayName": "Fixture Provider",
	"apiKeyEnvVar": "FIXTURE_API_KEY",
	"models": [
		{
			"id": "bare-url-video",
			"name": "Bare URL Video",
			"media": "video",
			"params": [
				{"paramID": "aspect_ratio", "flagID": "aspect-ratio", "allowedValues": ["16:9", "9:16"]},
				{"paramID": "resolution", "flagID": "resolution", "allowedValues": ["480p", "720p"]},
				{"paramID": "duration", "flagID": "duration", "minValue": 1, "maxValue": 15},
				{"flagID": "input-media", "maxMultiple": 1}
			]
		}
	],
	"config": {
		"videoAPI": {
			"asyncJobsURL": "https://fixture.example/videos",
			"urlStartPath": "/generations",
			"jobIDField": "request_id",
			"progressStatusText": ["pending"],
			"completedStatusText": "done",
			"failedStatusText": ["failed"],
			"urlPathSeq": ["video", "url"],
			"urlPathRequired": true,
			"inputMediaPayloadType": "json",
			"inputMediaProvParam": "image",
			"inputMediaStyle": "single-or-array", "inputMediaListProvParam": "images",
			"fallbackExt": ".mp4",
			"pollInterval": 1,
			"pollTimeout": 60
		}
	}
}`

// sameOriginVideoDoc declares video jobs whose same-origin results require authentication.
const sameOriginVideoDoc = `{
	"id": "fixture-provider",
	"displayName": "Fixture Provider",
	"apiKeyEnvVar": "FIXTURE_API_KEY",
	"models": [
		{
			"id": "same-origin-video",
			"name": "Same Origin Video",
			"media": "video",
			"params": [
				{"flagID": "input-media"}
			]
		}
	],
	"config": {
		"videoAPI": {
			"asyncJobsURL": "https://fixture.example/videos",
			"jobIDField": "id",
			"progressStatusText": ["in_progress"],
			"completedStatusText": "completed",
			"failedStatusText": ["failed"],
			"urlPathSeq": ["unsigned_urls", "0"],
			"urlContentPath": "/content?index=0",
			"authDownload": true,
			"inputMediaPayloadType": "json",
			"inputMediaProvParam": "input_references",
			"inputMediaStyle": "nested",
			"fallbackExt": ".mp4",
			"pollInterval": 1,
			"pollTimeout": 60
		}
	}
}`

// capture accumulates the requests a recording harness receives.
type capture struct {
	test testing.TB
	mu   sync.Mutex
	reqs []recReq
}

// fixtureProvider decodes the fixture provider config document of one layout through the strict
// parser — the same code path the embedded provider configs use.
func fixtureProvider(t *testing.T, layout string) catalog.Provider {
	t.Helper()

	var doc string

	switch layout {
	case multipartImage:
		doc = multipartImageDoc
	case singleOrArrayImage:
		doc = singleOrArrayImageDoc
	case nestedImage:
		doc = nestedImageDoc
	case contentEndpointVideo:
		doc = contentEndpointVideoDoc
	case bareURLVideo:
		doc = bareURLVideoDoc
	case sameOriginVideo:
		doc = sameOriginVideoDoc
	default:
		t.Fatalf("💣 no fixture provider config for layout %q", layout)
	}

	return loadTestProvider(t, fixtureProviderID, []byte(doc))
}

// fixtureModel returns the one model a fixture provider config declares.
func fixtureModel(t *testing.T, d catalog.Provider) catalog.Model {
	t.Helper()

	if len(d.Models) != 1 {
		t.Fatalf("💣 fixture provider config %s declares %d models, want 1", d.ID, len(d.Models))
	}

	return d.Models[0]
}

// pairLabel returns the provider-and-model label the submit functions prefix their errors with,
// built from the fixture's own identifiers.
func pairLabel(d catalog.Provider, m catalog.Model) string {
	return fmt.Sprintf("%s (%s)", d.ID, m.ID)
}

// imgAPI copies a provider config image description with its endpoints pointed at the harness.
func imgAPI(t *testing.T, d catalog.Provider, genURL, refsURL string) catalog.ImageAPI {
	t.Helper()

	if d.Config.ImageAPI == nil {
		t.Fatalf("💣 provider config has no ImageAPI")
	}

	api := *d.Config.ImageAPI

	api.GenURL = genURL
	if api.InputMediaURL != "" {
		api.InputMediaURL = refsURL
	}

	return api
}

// imgReq creates the post-adjustment run fixture for one provider config.
func imgReq(test testing.TB, d catalog.Provider, m catalog.Model, prompt string, gp params.Values, inputs []media.Input) generation.Generation {
	test.Helper()

	return generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: d.Identity(), Model: m}, Prompt: params.GetSetIf(true, prompt).ValOr(""), Preparation: generation.Preparation{Params: gp, InputMedia: inputs}, APIKey: os.Getenv((catalog.ProvModelPair{Provider: d.Identity(), Model: m}).Provider.APIKeyEnvVar)}
}

// makeInputMedia creates distinct byte-backed references labeled as PNG so tests can check order.
// The bytes are not decodable images.
func makeInputMedia(test testing.TB, n int) []media.Input {
	test.Helper()

	imgs := make([]media.Input, 0, n)
	for i := range n {
		imgs = append(imgs, media.Input{
			Bytes:    []byte(fmt.Sprintf("img-%c", 'A'+i)),
			MIME:     "image/png",
			Filepath: fmt.Sprintf("/in/ref-%c.png", 'A'+i),
		})
	}

	return imgs
}

// wantURI is the expected data-URI request form of one reference image, built independently of the
// production helpers.
func wantURI(test testing.TB, img media.Input) string {
	test.Helper()

	return "data:" + img.MIME + ";base64," + base64.StdEncoding.EncodeToString(img.Bytes)
}

// b64 is shorthand for standard base64 of raw bytes.
func b64(test testing.TB, data []byte) string {
	test.Helper()

	return base64.StdEncoding.EncodeToString(data)
}

// artOK serves the minimal valid one-entry image response, so assembly assertions run against a
// completed submit.
func artOK(test testing.TB) http.HandlerFunc {
	test.Helper()

	return textOK(test, fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, b64(test, []byte("OUT"))))
}

// driveSub calls submitImage against a recording server and returns the request, result, and
// fixture settings for the caller to check.
func driveSub(t *testing.T, layout string, inputs []media.Input, gp params.Values, respond http.HandlerFunc) subOut {
	t.Helper()
	d := fixtureProvider(t, layout)
	m := fixtureModel(t, d)
	recorder, srv := recSrv(t, respond)
	api := imgAPI(t, d, srv.URL+"/gen", srv.URL+"/refs")
	run := imgReq(t, d, m, "a prompt", gp, inputs)
	artifacts, err := submitImage(t.Context(), &api, "k", &run)

	out := subOut{artifacts: artifacts, err: err, api: api, model: m, label: pairLabel(d, m)}
	if recorder.count() > 0 {
		out.req = recorder.last(t)
	}

	return out
}

// okReq returns the recorded request of a submit that must have succeeded; ok is false (with the
// failure reported) otherwise.
func okReq(t *testing.T, out subOut) (recReq, bool) {
	t.Helper()

	if out.err != nil {
		t.Errorf("✗ submit failed: %v", out.err)

		return recReq{}, false
	}

	if out.req.reqMethod == "" {
		t.Errorf("✗ no request reached the harness")

		return recReq{}, false
	}

	return out.req, true
}

// jsonKeys decodes a recorded JSON body and checks that its keys match want exactly.
func jsonKeys(t *testing.T, body []byte, want ...string) map[string]any {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("💣 recorded body is not JSON: %v (%q)", err, body)
	}

	wantSet := make(map[string]bool, len(want))
	for _, k := range want {
		wantSet[k] = true
	}

	for k := range m {
		if !wantSet[k] {
			t.Errorf("✗ undeclared key %q in the request body", k)
		}
	}

	for k := range wantSet {
		if _, ok := m[k]; !ok {
			t.Errorf("✗ expected key %q absent from the request body", k)
		}
	}

	return m
}

// checkParts asserts the ordered file parts under field match the reference images exactly.
func checkParts(t *testing.T, form *multipart.Form, field string, imgs []media.Input) {
	t.Helper()

	parts := form.File[field]
	if len(parts) != len(imgs) {
		t.Errorf("✗ %d parts under %q, want %d", len(parts), field, len(imgs))

		return
	}

	for i, p := range parts {
		if !bytes.Equal(partBytes(t, p), imgs[i].Bytes) {
			t.Errorf("✗ part %d out of order or corrupted", i)
		}
	}
}

// partBytes reads one multipart file part's content.
func partBytes(t *testing.T, fh *multipart.FileHeader) []byte {
	t.Helper()

	f, err := fh.Open()
	if err != nil {
		t.Fatalf("💣 open form part: %v", err)
	}

	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("💣 read form part: %v", err)
	}

	return data
}

// inputMediaObject is the expected InputMediaSingle/InputMediaNested reference object for one image
// (the {"type":"image_url", …} forms).
func inputMediaObject(test testing.TB, image media.Input, inputMediaNested bool) map[string]any {
	test.Helper()

	if inputMediaNested {
		return map[string]any{"type": "image_url", "image_url": map[string]any{"url": wantURI(test, image)}}
	}

	return map[string]any{"type": "image_url", "url": wantURI(test, image)}
}

// inputMediaList is the expected ordered reference-object list for imgs.
func inputMediaList(test testing.TB, images []media.Input, inputMediaNested bool) []any {
	test.Helper()

	list := make([]any, 0, len(images))
	for _, image := range images {
		list = append(list, inputMediaObject(test, image, inputMediaNested))
	}

	return list
}

// checkFixed asserts the single-or-array layout's fixed field arrived verbatim from
// FixedProvFields.
func checkFixed(t *testing.T, body map[string]any) {
	t.Helper()

	if body["response_format"] != "b64_json" {
		t.Errorf(`✗ response_format = %v, want "b64_json" from FixedProvFields`, body["response_format"])
	}
}

// normCases returns the shared response-normalization cases and their expected artifacts.
func normCases(test testing.TB) []normCase {
	test.Helper()

	return []normCase{
		{
			name: "b64 decodes, unsniffable falls to the declared fallback", layout: multipartImage,
			status: 200, body: fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, b64(test, []byte("PNGDATA"))),
			artifacts: []wantArt{{[]byte("PNGDATA"), ".png"}},
		},
		{
			name: "absent MIME sniffs the bytes", layout: multipartImage,
			status: 200, body: fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, b64(test, jpegStub(test))),
			artifacts: []wantArt{{jpegStub(test), ".jpg"}},
		},
		{
			name: "mime_type decides", layout: multipartImage,
			status: 200, body: fmt.Sprintf(`{"data":[{"b64_json":%q,"mime_type":"image/webp"}]}`, b64(test, jpegStub(test))),
			artifacts: []wantArt{{jpegStub(test), ".webp"}},
		},
		{
			name: "media_type decides", layout: nestedImage,
			status: 200, body: fmt.Sprintf(`{"data":[{"b64_json":%q,"media_type":"image/webp"}]}`, b64(test, jpegStub(test))),
			artifacts: []wantArt{{jpegStub(test), ".webp"}},
		},
		{
			name: "media_type svg maps to .svg", layout: nestedImage,
			status: 200, body: fmt.Sprintf(`{"data":[{"b64_json":%q,"media_type":"image/svg+xml"}]}`, b64(test, svgStub(test))),
			artifacts: []wantArt{{svgStub(test), ".svg"}},
		},
		{
			name: "mime_type svg maps to .svg", layout: singleOrArrayImage,
			status: 200, body: fmt.Sprintf(`{"data":[{"b64_json":%q,"mime_type":"image/svg+xml"}]}`, b64(test, svgStub(test))),
			artifacts: []wantArt{{svgStub(test), ".svg"}},
		},
		{
			name: "two entries convert in order", layout: multipartImage,
			status:    200,
			body:      fmt.Sprintf(`{"data":[{"b64_json":%q},{"b64_json":%q}]}`, b64(test, []byte("img-one")), b64(test, []byte("img-two"))),
			artifacts: []wantArt{{[]byte("img-one"), ".png"}, {[]byte("img-two"), ".png"}},
		},
		{
			name: "empty data is the no-data error", layout: multipartImage,
			status: 200, body: `{"data":[]}`, want: errs.ErrResponseNoData,
		},
		{
			name: "a dataless entry is the no-data error", layout: multipartImage,
			status: 200, body: `{"data":[{}]}`, want: errs.ErrResponseNoData,
		},
		{
			name: "an empty inline entry never becomes an artifact", layout: multipartImage,
			status: 200, body: `{"data":[{"b64_json":""}]}`, want: errs.ErrResponseNoData,
		},
		{
			name: "invalid base64 wraps the decode sentinel", layout: multipartImage,
			status: 200, body: `{"data":[{"b64_json":"!!!notbase64!!!"}]}`, want: errs.ErrResponseDecode,
		},
		{
			name: "malformed JSON wraps the decode sentinel", layout: multipartImage,
			status: 200, body: `not json`, want: errs.ErrResponseDecode,
		},
		{
			name: "a data field that is not an array wraps the decode sentinel", layout: multipartImage,
			status: 200, body: `{"data":"oops"}`, want: errs.ErrResponseDecode,
		},
		{
			name: "a data entry that is not an object wraps the decode sentinel", layout: multipartImage,
			status: 200, body: `{"data":[1]}`, want: errs.ErrResponseDecode,
		},
		{
			name: "a non-2xx response is the provider API error", layout: multipartImage,
			status: 500, body: `{"error":{"message":"down"}}`, want: errs.ErrResponseStatus,
		},
	}
}

// checkNormOK asserts one successful normalization outcome: the expected artifacts in order, with
// exactly one data source per artifact.
func checkNormOK(t *testing.T, c normCase, out subOut) {
	t.Helper()

	if out.err != nil {
		t.Errorf("✗ submit failed: %v", out.err)

		return
	}

	if len(out.artifacts) != len(c.artifacts) {
		t.Errorf("✗ %d artifacts, want %d", len(out.artifacts), len(c.artifacts))

		return
	}

	for i, w := range c.artifacts {
		if !bytes.Equal(out.artifacts[i].Data, w.data) || out.artifacts[i].FileExt != w.ext {
			t.Errorf("✗ artifact %d = (%d bytes, %q), want (%d bytes, %q)", i, len(out.artifacts[i].Data), out.artifacts[i].FileExt, len(w.data), w.ext)
		}

		checkCarriage(t, c.name, out.artifacts[i])
	}
}

// checkNormFail asserts one failing normalization outcome: the classified sentinel, the provider
// label in the text, and zero artifacts.
func checkNormFail(t *testing.T, label string, artifacts []artifact.Media, err, want error) {
	t.Helper()

	if !errors.Is(err, want) {
		t.Errorf("✗ err = %v, want %v", err, want)
	}

	if err != nil && !strings.Contains(err.Error(), label) {
		t.Errorf("✗ error %q does not name the provider label %q", err.Error(), label)
	}

	if len(artifacts) != 0 {
		t.Errorf("✗ a failing traversal returned %d artifacts, want none", len(artifacts))
	}
}

// add records one incoming request, body included.
func (c *capture) add(r *http.Request) {
	c.test.Helper()

	body, _ := io.ReadAll(r.Body)

	c.mu.Lock()
	defer c.mu.Unlock()

	c.reqs = append(c.reqs, recReq{reqMethod: r.Method, reqPath: r.URL.Path, header: r.Header.Clone(), body: body})
}

// count returns how many requests the harness has received.
func (c *capture) count() int {
	c.test.Helper()

	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.reqs)
}

// last returns the most recently captured request; fatal when none arrived, since no request
// assertion after it could run.
func (c *capture) last(t *testing.T) recReq {
	t.Helper()
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.reqs) == 0 {
		t.Fatalf("💣 no request reached the harness — nothing to inspect")
	}

	return c.reqs[len(c.reqs)-1]
}

// recSrv records requests and serves them through respond, closing the server after the test.
func recSrv(t *testing.T, respond http.HandlerFunc) (*capture, *httptest.Server) {
	t.Helper()

	c := &capture{test: t}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c.add(r)
		respond(w, r)
	}))
	t.Cleanup(srv.Close)

	return c, srv
}

// textOK responds 200 with a fixed body.
func textOK(test testing.TB, body string) http.HandlerFunc {
	test.Helper()

	return func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, body) }
}

// jpegStub returns a JPEG header sufficient for MIME detection, but not image decoding.
func jpegStub(test testing.TB) []byte {
	test.Helper()

	return []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 0x4a, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00}
}

// pngStub is the 8-byte PNG signature — enough magic for content-type detection.
func pngStub(test testing.TB) []byte {
	test.Helper()

	return []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
}

// svgStub returns an SVG document for tests that select the extension from a declared MIME type.
func svgStub(test testing.TB) []byte {
	test.Helper()

	return []byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"/>`)
}
