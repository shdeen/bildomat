package google

// Invariants tested:
//  1. Original Veo references: For direct and saved Veo references, adjustReuse must preserve the
//     full original URI, including query values containing equals signs. Given duration four and
//     resolution 1080p, it must set duration eight and resolution 720p and return exactly two
//     adjustments without an error. Given a valid Veo URI and no parameters, adjustReuse must
//     succeed with two adjustment records whose InputVal fields are empty. Calling it again with
//     the resulting parameter map must succeed without further adjustments.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/params"
)

// TestVeoReuse verifies invariant #1: Original Veo references.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// For direct and saved Veo references, adjustReuse must preserve the full original URI, including
// query values containing equals signs. Given duration four and resolution 1080p, it must set
// duration eight and resolution 720p and return exactly two adjustments without an error.
// Kind: permanent.
func TestVeoReuse(t *testing.T) {
	videoURI := "https://generativelanguage.googleapis.com/v1beta/files/original:download?alt=media&token=a=b"
	for _, reuse := range []*metadata.Reuse{
		{URI: videoURI},
		{Record: reuseRecord(t, "google", "veo-3.1-generate-preview", "720p", videoURI)},
	} {
		model := mustModel(t, "veo-3.1-generate-preview")
		parameterValues := params.Values{params.FlagTypeDuration: 4, params.FlagTypeResolution: "1080p"}

		resolvedURI, changes, err := adjustReuse(&model, parameterValues, nil, reuse)
		if err != nil || resolvedURI != videoURI {
			t.Errorf("✗ original URI %q, error %v", resolvedURI, err)
		}

		if parameterValues[params.FlagTypeDuration] != 8 || parameterValues[params.FlagTypeResolution] != "720p" {
			t.Errorf("✗ extension parameters %v", parameterValues)
		}

		if len(changes) != 2 {
			t.Errorf("✗ adjustment count %d", len(changes))
		}
	}

	if !t.Failed() {
		t.Log("✓ Veo reuses original references with visible parameter adjustments")
	}
}

// TestVeoReuseUnspecifiedParams verifies invariant #1: Original Veo references.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Given a valid Veo URI and no parameters, adjustReuse must succeed with two adjustment records
// whose InputVal fields are empty. Calling it again with the resulting parameter map must succeed
// without further adjustments.
// Kind: permanent.
func TestVeoReuseUnspecifiedParams(t *testing.T) {
	model := mustModel(t, "veo-3.1-generate-preview")
	reuse := &metadata.Reuse{URI: "https://generativelanguage.googleapis.com/v1beta/files/source:download?alt=media"}
	parameterValues := params.Values{}

	_, changes, err := adjustReuse(&model, parameterValues, nil, reuse)
	if err != nil || len(changes) != 2 {
		t.Errorf("✗ absent parameter adjustments: %v, error %v", changes, err)
	}

	for _, change := range changes {
		if change.InputVal != "" {
			t.Errorf("✗ invented supplied value %q", change.InputVal)
		}
	}

	_, changes, err = adjustReuse(&model, parameterValues, nil, reuse)
	if err != nil || len(changes) != 0 {
		t.Errorf("✗ unchanged settings produced adjustments: %v, error %v", changes, err)
	}

	if !t.Failed() {
		t.Log("✓ missing and unchanged settings retain the established notice meanings")
	}
}

// reuseRecord decodes a provisional record whose artifact is unavailable.
func reuseRecord(t *testing.T, provider, model, resolution, uri string) *metadata.Record {
	t.Helper()

	document := `{"schema-version":1,"provider":"PROVIDER","model":"MODEL","status":"failed","request":{"adjusted-parameters":{"resolution":"RESOLUTION"},"provider-requests":[]},"provider-responses":[],"artifacts":[],"return-values":[{"provider":"PROVIDER","model":"MODEL","retained-data":{"operation":"operations/previous","videos":[{"uri":"URI"}]}}]}`
	document = strings.NewReplacer("PROVIDER", provider, "MODEL", model, "RESOLUTION", resolution, "URI", uri).Replace(document)

	var record metadata.Record
	if err := json.Unmarshal([]byte(document), &record); err != nil {
		t.Fatalf("💣 record fixture: %v", err)
	}

	return &record
}
