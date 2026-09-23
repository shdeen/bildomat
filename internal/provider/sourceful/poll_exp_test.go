package sourceful

// Invariants tested:
//  1. Observation recovery: After a pending response followed by HTTP 429, 502, 503, or 504,
//     httpapi.Poll with Sourceful's jobPoll must issue a third GET and accept the completed
//     response without an error.
//  2. Output field alternatives: For a completed response, classifyJobResponse must prefer each
//     string field in data.job.result.output independently, including empty strings, and use
//     data.result.output when the primary field is absent or has the wrong type. It must tolerate
//     unknown fields and invalid optional MIME values. An empty primary URL must return
//     ErrResponseNoData without completion, even when the alternate URL is nonempty.
//  3. Sourceful poll response: For arbitrary response bytes, classifyJobResponse must return
//     pending without a URL, completion with a nonempty URL, or an error without completion or a
//     URL. Errors must match ErrResponseDecode, ErrResponseNoData, ErrResponseGen, or
//     ErrResponseUnknown. The call must not panic.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
)

// TestSourcefulPollRecovery verifies invariant #1: Observation recovery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
// What is being tested:
// After a pending response followed by HTTP 429, 502, 503, or 504, httpapi.Poll with Sourceful's
// jobPoll must issue a third GET and accept the completed response without an error.
func TestSourcefulPollRecovery(t *testing.T) {
	for _, statusCode := range []int{429, 502, 503, 504} {
		t.Run(strconv.Itoa(statusCode), func(t *testing.T) {
			observations := 0

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method != http.MethodGet {
					t.Errorf("✗ observation used %s instead of GET", request.Method)
				}

				observations++
				switch observations {
				case 1:
					_, _ = writer.Write([]byte(`{"data":{"job":{"status":"pending"}}}`))
				case 2:
					writer.WriteHeader(statusCode)
					_, _ = writer.Write([]byte(`{"message":"temporary observation failure"}`))
				default:
					_, _ = writer.Write([]byte(`{"data":{"job":{"status":"completed","result":{"output":{"url":"https://result.example/image.jpg"}}}}}`))
				}
			}))
			defer server.Close()

			poller := &jobPoll{adapterAPI: &catalog.AdapterAPI{APIBase: server.URL, PendingStatusText: []string{"pending"}, ReadyStatusText: "completed"}, jobID: "sourceful-job", providerModelName: "sourceful/model"}

			err := httpapi.Poll(t.Context(), time.Millisecond, time.Second, poller)
			if err != nil || observations != 3 {
				t.Errorf("✗ completion error=%v observations=%d; want successful third observation", err, observations)
			}

			if !t.Failed() {
				t.Log("✓ temporary observation failure recovers without resubmission")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every selected HTTP status recovers to completion")
	}
}

// TestOutputFieldAlternatives verifies invariant #2: Output field alternatives.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// For a completed response, classifyJobResponse must prefer each string field in
// data.job.result.output independently, including empty strings, and use data.result.output when
// the primary field is absent or has the wrong type. It must tolerate unknown fields and invalid
// optional MIME values. An empty primary URL must return ErrResponseNoData without completion, even
// when the alternate URL is nonempty.
// Kind: permanent.
func TestOutputFieldAlternatives(t *testing.T) {
	for _, responseCase := range []struct {
		name, document, resultURL, resultMIME string
		classification                        error
	}{
		{"primary", `{"extra":1,"data":{"job":{"status":"completed","result":{"output":{"url":"primary","mimeType":"image/png","extra":true}}},"result":{"output":{"url":"secondary","mimeType":"image/jpeg"}}}}`, "primary", "image/png", nil},
		{"mixed", `{"data":{"job":{"status":"completed","result":{"output":{"url":"primary","mimeType":17}}},"result":{"output":{"url":"secondary","mimeType":"image/jpeg"}}}}`, "primary", "image/jpeg", nil},
		{"empty MIME", `{"data":{"job":{"status":"completed","result":{"output":{"url":"primary","mimeType":""}}},"result":{"output":{"mimeType":"image/jpeg"}}}}`, "primary", "", nil},
		{"wrong primary URL", `{"data":{"job":{"status":"completed","result":{"output":{"url":17,"mimeType":"image/png"}}},"result":{"output":{"url":"secondary"}}}}`, "secondary", "image/png", nil},
		{"empty primary URL", `{"data":{"job":{"status":"completed","result":{"output":{"url":""}}},"result":{"output":{"url":"secondary"}}}}`, "", "", errs.ErrResponseNoData},
		{"optional MIME object", `{"data":{"job":{"status":"completed","result":{"output":{"url":"primary","mimeType":{}}}}}}`, "primary", "", nil},
	} {
		t.Run(responseCase.name, func(t *testing.T) {
			poller := &jobPoll{adapterAPI: &catalog.AdapterAPI{ReadyStatusText: "completed"}, jobID: "identity", providerModelName: "sourceful/model"}
			complete, err := poller.classifyJobResponse([]byte(responseCase.document))

			resultURL := poller.resultURL
			if !errors.Is(err, responseCase.classification) || complete != (responseCase.classification == nil) || resultURL != responseCase.resultURL || poller.resultMIME != responseCase.resultMIME {
				t.Errorf("✗ URL=%q complete=%t MIME=%q error=%v; want URL=%q MIME=%q error=%v", resultURL, complete, poller.resultMIME, err, responseCase.resultURL, responseCase.resultMIME, responseCase.classification)
			}

			if !t.Failed() {
				t.Log("✓ output alternatives preserve each field's precedence and tolerance")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ both published output shapes preserve optional-field behavior")
	}
}

// FuzzSourcefulPollResponse verifies invariant #3: Sourceful poll response.
//
// What is being tested:
// For arbitrary response bytes, classifyJobResponse must return pending without a URL, completion
// with a nonempty URL, or an error without completion or a URL. Errors must match
// ErrResponseDecode, ErrResponseNoData, ErrResponseGen, or ErrResponseUnknown. The call must not
// panic.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSourcefulPollResponse(f *testing.F) {
	f.Add([]byte(`{"data":{"job":{"status":"completed","result":{"output":{"url":17,"mimeType":"image/png"}}},"result":{"output":{"url":"https://media.example/a.png","mimeType":false}}}}`))

	providerConfig, loaded := loadSourcefulTestProvider(f)
	if !loaded {
		f.Fatal("💣 Sourceful config failed to load")
	}

	adapterAPI := providerConfig.Config.AdapterAPI

	f.Add(sourcefulPollDocument(f, adapterAPI.PendingStatusText[0], ""))
	f.Add(sourcefulCompletedPollDocument(f, "https://media.example/result.webp", "image/webp", adapterAPI.ReadyStatusText))
	f.Add(sourcefulPollDocument(f, adapterAPI.FailedStatusText[0], "fuzz failure"))
	f.Add([]byte("{"))
	f.Add([]byte(`{"data":null}`))

	f.Fuzz(func(t *testing.T, responseBody []byte) {
		verifySourcefulFuzzPollResponse(t, adapterAPI, responseBody)
	})
}

// verifySourcefulPollResultConsistency checks agreement between completion, result URL, and error.
func verifySourcefulPollResultConsistency(t *testing.T, resultURL string, jobDone bool, classificationErr error) {
	t.Helper()

	switch {
	case classificationErr != nil && (resultURL != "" || jobDone):
		t.Errorf("✗ classified poll error returned URL %q and done=%v", resultURL, jobDone)
	case classificationErr == nil && jobDone && resultURL == "":
		t.Errorf("✗ completed poll result has no URL")
	case classificationErr == nil && !jobDone && resultURL != "":
		t.Errorf("✗ pending poll result returned URL %q", resultURL)
	}
}

// sourcefulPollErrorIsClassified reports whether an error belongs to an accepted response category.
func sourcefulPollErrorIsClassified(test testing.TB, classificationErr error) bool {
	test.Helper()

	return classificationErr == nil ||
		errors.Is(classificationErr, errs.ErrResponseDecode) ||
		errors.Is(classificationErr, errs.ErrResponseNoData) ||
		errors.Is(classificationErr, errs.ErrResponseGen) ||
		errors.Is(classificationErr, errs.ErrResponseUnknown)
}

// verifySourcefulFuzzPollResponse checks response classification and result consistency without
// HTTP requests.
func verifySourcefulFuzzPollResponse(t *testing.T, adapterAPI *catalog.AdapterAPI, responseBody []byte) {
	t.Helper()

	pollRequest := &jobPoll{
		adapterAPI:        adapterAPI,
		jobID:             "sourceful-fuzz-job",
		providerModelName: ProviderID,
	}

	jobDone, classificationErr := pollRequest.classifyJobResponse(responseBody)
	resultURL := pollRequest.resultURL
	verifySourcefulPollResultConsistency(t, resultURL, jobDone, classificationErr)

	if !sourcefulPollErrorIsClassified(t, classificationErr) {
		t.Errorf("✗ poll body returned unclassified error: %v", classificationErr)
	}

	if !t.Failed() {
		t.Log("✓ the Sourceful poll body classified without a network request")
	}
}
