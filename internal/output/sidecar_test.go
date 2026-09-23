package output

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// Invariants tested:
//  1. Sidecar front matter identifiers: Given a prompt, model ID, and adjusted parameters,
//     RenderSidecar must begin with a front-matter fence and include the supplied values under flag
//     IDs, without the tested provider request spellings. Its timestamp must parse as RFC 3339, end
//     in Z, and fall within the preceding minute.
//  2. Sidecar thoughts and fences: Given two thought chunks, RenderSidecar must include both after
//     the closing front-matter fence and include a --- separator in the body. With no thoughts, it
//     must retain the prompt and closing fence and leave no body content.
//  3. Sidecar input media: Given two input paths, RenderSidecar must write their quoted values in
//     input order under input-media and omit the provider request name reference_paths.
//  4. Omitted sidecar parameters: Given an empty quality value and no other parameters or input
//     media, RenderSidecar must omit quality, aspect-ratio, duration, num-images, and input-media
//     from the output.

// TestRenderSidecarFlagIDKeys verifies invariant #1: Sidecar front matter identifiers.
//
// What is being tested:
// Given a prompt, model ID, and adjusted parameters, RenderSidecar must begin with a front-matter
// fence and include the supplied values under flag IDs, without the tested provider request
// spellings. Its timestamp must parse as RFC 3339, end in Z, and fall within the preceding minute.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRenderSidecarFlagIDKeys(t *testing.T) {
	md := renderedSidecarFixture(t, []string{"first thought", "second thought"})
	if !strings.HasPrefix(md, "---\n") {
		t.Errorf("✗ sidecar does not open with the front-matter fence:\n%s", md)
	}

	for _, want := range []string{
		`prompt: "a red cube"`,
		`model: "fixture-model"`,
		`aspect-ratio: "16:9"`,
		`resolution: "2k"`,
		`strength: "0.4"`,
		`thinking-level: "high"`,
		`include-thoughts: "true"`,
	} {
		if !strings.Contains(md, want+"\n") {
			t.Errorf("✗ front matter missing the line %q in:\n%s", want, md)
		}
	}

	for _, old := range []string{"aspect_ratio:", "thinking_level:", "include_thoughts:"} {
		if strings.Contains(md, old) {
			t.Errorf("✗ the old wire spelling %q still keys the front matter:\n%s", old, md)
		}
	}

	tsRe := regexp.MustCompile(`(?m)^timestamp: "([^"]+)"$`)

	timestampLine := tsRe.FindStringSubmatch(md)
	if timestampLine == nil {
		t.Errorf("✗ timestamp line missing:\n%s", md)
	} else if parsed, err := time.Parse(time.RFC3339Nano, timestampLine[1]); err != nil || !strings.HasSuffix(timestampLine[1], "Z") {
		t.Errorf("✗ timestamp %q is not UTC in the RFC 3339 form the JSON document uses: %v", timestampLine[1], err)
	} else if time.Since(parsed) > time.Minute || time.Since(parsed) < 0 {
		t.Errorf("✗ timestamp %q is not the current UTC time", timestampLine[1])
	}

	if !t.Failed() {
		t.Log("✓ the front matter keys every entry by its flag ID with the old wire spellings absent")
	}
}

// TestRenderSidecarThoughtsAndFences verifies invariant #2: Sidecar thoughts and fences.
//
// What is being tested:
// Given two thought chunks, RenderSidecar must include both after the closing front-matter fence
// and include a --- separator in the body. With no thoughts, it must retain the prompt and closing
// fence and leave no body content.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRenderSidecarThoughtsAndFences(t *testing.T) {
	md := renderedSidecarFixture(t, []string{"first thought", "second thought"})

	_, sidecarBody, ok := strings.Cut(md, "---\n\n")
	if !ok {
		t.Fatalf("💣 no closing front-matter fence; cannot locate the body in:\n%s", md)
	}

	if !strings.Contains(sidecarBody, "first thought") || !strings.Contains(sidecarBody, "second thought") {
		t.Errorf("✗ thought chunks missing from the body:\n%s", sidecarBody)
	}

	if !strings.Contains(sidecarBody, "\n---\n") {
		t.Errorf("✗ thought chunks are not joined by a --- separator:\n%s", sidecarBody)
	}

	alone := renderedSidecarFixture(t, nil)
	if alone == "" {
		t.Fatalf("💣 zero thoughts rendered no sidecar at all")
	}

	if !strings.Contains(alone, `prompt: "a red cube"`) {
		t.Errorf("✗ zero-thoughts sidecar lost the front matter:\n%s", alone)
	}

	closing := strings.Index(alone[4:], "---\n")
	if closing < 0 {
		t.Errorf("✗ zero-thoughts sidecar has no closing fence:\n%s", alone)
	} else if rest := strings.TrimSpace(alone[4+closing+4:]); rest != "" {
		t.Errorf("✗ zero-thoughts sidecar carries body content %q, want front matter alone", rest)
	}

	if !t.Failed() {
		t.Log("✓ thought chunks join with --- separators and zero thoughts yield the front matter alone")
	}
}

// TestRenderSidecarInputMedia verifies invariant #3: Sidecar input media.
//
// What is being tested:
// Given two input paths, RenderSidecar must write their quoted values in input order under
// input-media and omit the provider request name reference_paths.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRenderSidecarInputMedia(t *testing.T) {
	paramFlags := params.Flags()

	imgs := []media.Input{{Filepath: "/a.png"}, {Filepath: "/b.png"}}
	md := string(RenderSidecar("p", "m", paramFlags, params.Values{}, imgs, nil))

	want := `input-media: "\"/a.png\", \"/b.png\""`
	if !strings.Contains(md, want+"\n") {
		t.Errorf("✗ front matter missing the input-media entry %q in:\n%s", want, md)
	}

	if strings.Contains(md, "reference_paths") {
		t.Errorf("✗ the old reference_paths key still renders:\n%s", md)
	}

	if !t.Failed() {
		t.Log("✓ the sent input images key the input-media entry with the pre-change value form")
	}
}

// TestRenderSidecarOmitsAbsent verifies invariant #4: Omitted sidecar parameters.
//
// What is being tested:
// Given an empty quality value and no other parameters or input media, RenderSidecar must omit
// quality, aspect-ratio, duration, num-images, and input-media from the output.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestRenderSidecarOmitsAbsent(t *testing.T) {
	paramFlags := params.Flags()

	md := string(RenderSidecar("p", "m", paramFlags, params.Values{params.FlagTypeQuality: ""}, nil, nil))
	if strings.Contains(md, "quality") {
		t.Errorf("✗ an empty-valued parameter leaked into the front matter:\n%s", md)
	}

	for _, absent := range []string{"aspect-ratio:", "duration:", "num-images:", "input-media:"} {
		if strings.Contains(md, absent) {
			t.Errorf("✗ an absent parameter rendered a front-matter entry %q:\n%s", absent, md)
		}
	}

	if !t.Failed() {
		t.Log("✓ absent and empty-valued parameters stay out of the front matter")
	}
}

// renderedSidecarFixture calls RenderSidecar with a fixed prompt, model ID, adjusted parameter
// values, and the supplied thoughts.
func renderedSidecarFixture(t *testing.T, thoughts []string) string {
	t.Helper()

	paramFlags := params.Flags()

	adjusted := params.Values{
		params.FlagTypeAspect:        "16:9",
		params.FlagTypeResolution:    "2k",
		"strength":                   0.4,
		params.FlagTypeThinkingLevel: "high",
		params.FlagTypeThoughts:      true,
	}

	return string(RenderSidecar("a red cube", "fixture-model", paramFlags, adjusted, nil, thoughts))
}
