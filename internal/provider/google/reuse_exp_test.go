package google

// Invariants tested:
//  1. Veo reuse eligibility: adjustReuse must return an error and an empty URI for Lite or image
//     targets, a non-Google source, an incompatible source model, source resolution 1080p, a
//     missing source URI, or simultaneous reuse and image input. adjustReuse must recover the
//     original URI from full operation responses when retained values are absent and accept
//     repeated copies of that URI. Distinct video references must return ErrReuseVideoMultiple;
//     source duration 142 or aspect 1:1 must return ErrReuseVideoModel. Accepted landscape,
//     portrait, or unspecified source settings must preserve the URI, and every rejection must
//     return an empty URI.
//  2. Original Veo references: For arbitrary direct URI strings, adjustReuse must either return the
//     string unchanged with duration eight and resolution 720p, or return an error and an empty
//     URI. It must not panic.

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// TestVeoReuseEligibility verifies invariant #1: Veo reuse eligibility.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// adjustReuse must return an error and an empty URI for Lite or image targets, a non-Google source,
// an incompatible source model, source resolution 1080p, a missing source URI, or simultaneous
// reuse and image input.
// Kind: permanent.
func TestVeoReuseEligibility(t *testing.T) {
	videoURI := "https://generativelanguage.googleapis.com/v1beta/files/original:download?alt=media"
	for _, testCase := range []struct {
		name, target, provider, source, resolution, uri string
		inputs                                          []media.Input
	}{
		{name: "lite target", target: "veo-3.1-lite-generate-preview", provider: "google", source: "veo-3.1-generate-preview", resolution: "720p", uri: videoURI},
		{name: "image target", target: "gemini-3.1-flash-image", provider: "google", source: "veo-3.1-generate-preview", resolution: "720p", uri: videoURI},
		{name: "provider", target: "veo-3.1-generate-preview", provider: "other", source: "veo-3.1-generate-preview", resolution: "720p", uri: videoURI},
		{name: "source model", target: "veo-3.1-generate-preview", provider: "google", source: "image-model", resolution: "720p", uri: videoURI},
		{name: "source resolution", target: "veo-3.1-generate-preview", provider: "google", source: "veo-3.1-generate-preview", resolution: "1080p", uri: videoURI},
		{name: "missing URI", target: "veo-3.1-generate-preview", provider: "google", source: "veo-3.1-generate-preview", resolution: "720p"},
		{name: "conflicting input", target: "veo-3.1-generate-preview", provider: "google", source: "veo-3.1-generate-preview", resolution: "720p", uri: videoURI, inputs: []media.Input{{URL: "https://example.com/photo", MIME: "image/png"}}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			model := mustModel(t, testCase.target)
			reuse := &metadata.Reuse{Record: reuseRecord(t, testCase.provider, testCase.source, testCase.resolution, testCase.uri)}

			resolvedURI, _, err := adjustReuse(&model, params.Values{}, testCase.inputs, reuse)
			if err == nil || resolvedURI != "" {
				t.Errorf("✗ accepted incompatible reference %q, %v", resolvedURI, err)
			}

			if !t.Failed() {
				t.Log("✓ incompatible reuse rejected")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ Veo reuse validates eligibility before submission")
	}
}

// TestVeoReuseRecordReferences verifies invariant #1: Veo reuse eligibility.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// adjustReuse must recover the original URI from full operation responses when retained values are
// absent and accept repeated copies of that URI. Distinct video references must return
// ErrReuseVideoMultiple; source duration 142 or aspect 1:1 must return ErrReuseVideoModel. Accepted
// landscape, portrait, or unspecified source settings must preserve the URI, and every rejection
// must return an empty URI.
// Kind: permanent.
func TestVeoReuseRecordReferences(t *testing.T) {
	videoURI := "https://generativelanguage.googleapis.com/v1beta/files/source:download?alt=media"
	secondURI := "https://generativelanguage.googleapis.com/v1beta/files/second:download?alt=media"

	for _, testCase := range []struct {
		name        string
		retained    bool
		responseURI string
		adjusted    params.Values
		failure     error
	}{
		{name: "full response fallback", responseURI: videoURI},
		{name: "duplicate reference", retained: true, responseURI: videoURI},
		{name: "multiple references", retained: true, responseURI: secondURI, failure: errs.ErrReuseVideoMultiple},
		{name: "long source", retained: true, adjusted: params.Values{params.FlagTypeDuration: 142}, failure: errs.ErrReuseVideoModel},
		{name: "square source", retained: true, adjusted: params.Values{params.FlagTypeAspect: "1:1"}, failure: errs.ErrReuseVideoModel},
		{name: "landscape source", retained: true, adjusted: params.Values{params.FlagTypeAspect: "16:9"}},
		{name: "portrait source", retained: true, adjusted: params.Values{params.FlagTypeAspect: "9:16"}},
		{name: "no source settings", retained: true, adjusted: params.Values{}},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			record := reuseRecord(t, "google", "veo-3.1-generate-preview", "720p", videoURI)
			if !testCase.retained {
				record.Returns = nil
			}

			if testCase.adjusted != nil {
				serialized, err := json.Marshal(testCase.adjusted)
				if err != nil {
					t.Fatalf("💣 serialize source parameter fixture: %v", err)
				}

				record.Request.Adjusted = serialized
			}

			if testCase.responseURI != "" {
				body := `{"done":true,"response":{"generateVideoResponse":{"generatedSamples":[{"video":{"uri":"` + testCase.responseURI + `"}}]}}}`
				record.Responses = []metadata.Response{{Body: json.RawMessage(body)}, {Body: json.RawMessage(body)}}
			}

			model := mustModel(t, "veo-3.1-fast-generate-preview")

			resolvedURI, _, err := adjustReuse(&model, params.Values{}, nil, &metadata.Reuse{Record: record})
			if !errors.Is(err, testCase.failure) {
				t.Errorf("✗ reuse error %v, expected %v", err, testCase.failure)
			}

			if testCase.failure == nil && resolvedURI != videoURI || testCase.failure != nil && resolvedURI != "" {
				t.Errorf("✗ resolved URI %q", resolvedURI)
			}

			if !t.Failed() {
				t.Log("✓ record references and known source restrictions respected")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ full response references remain usable and ambiguous records fail")
	}
}

// FuzzVeoReuseURI verifies invariant #2: Original Veo references.
// Test class: Expanded.
// Test layer: Fuzzing.
// What is being tested:
// For arbitrary direct URI strings, adjustReuse must either return the string unchanged with
// duration eight and resolution 720p, or return an error and an empty URI. It must not panic.
// Kind: permanent.
func FuzzVeoReuseURI(f *testing.F) {
	f.Add("https://generativelanguage.googleapis.com/v1beta/files/source:download?alt=media&token=a=b")
	f.Add("https://example.com/video")
	f.Add("")
	f.Fuzz(func(t *testing.T, suppliedURI string) {
		model := mustModel(t, "veo-3.1-generate-preview")
		parameterValues := params.Values{}

		resolvedURI, _, err := adjustReuse(&model, parameterValues, nil, &metadata.Reuse{URI: suppliedURI})
		if err == nil && (resolvedURI != suppliedURI || parameterValues[params.FlagTypeDuration] != 8 || parameterValues[params.FlagTypeResolution] != "720p") {
			t.Errorf("✗ accepted reference changed or parameters invalid: %q, %v", resolvedURI, parameterValues)
		}

		if err != nil && resolvedURI != "" {
			t.Errorf("✗ rejected URI remained usable: %q", resolvedURI)
		}

		if !t.Failed() {
			t.Log("✓ direct reuse is unchanged or rejected")
		}
	})
}
