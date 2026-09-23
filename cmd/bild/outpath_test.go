package main

// Invariants tested:
//  1. Output directory for path: resolveOutputTarget must retain absolute directories, expand
//     home-relative directories, and resolve relative paths against the invocation directory.
//  2. Output path format precedence: For a model supporting output-format, applyOutPathFormat must
//     use the path format and record a conflicting explicit value. Without a format or support, it
//     must leave the input unchanged.
//  3. Requested artifact extension compatibility: applyLandingExt must use the requested spelling
//     only for a matching format. Incompatible extensions must remain unchanged with a derived
//     adjustment, and repeated identical fallbacks must produce one record.
//  4. Artifact extension selection: landingExt must prefer the explicit extension, then the
//     adjusted format for a named file, and otherwise return an empty extension. A numeric format
//     must return errs.ErrParamValueTypeMismatch.
//  5. Output directory selection: resolveOutputTarget must use the configured default only when
//     OutPath is omitted and use the invocation directory for a bare filename.

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// TestOutDirForPath verifies invariant #1: Output directory for path.
// Test class: Expanded.
// Test layer: Coverage.
// What is being tested:
// Given a bare filename, relative directory, absolute directory, or home-relative directory in
// OutPath, resolveOutputTarget must return the expected absolute directory without an error.
// Relative paths must use the invocation directory, and ~/x must use the supplied HOME directory.
func TestOutDirForPath(t *testing.T) {
	workDir := t.TempDir()
	homeDir := t.TempDir()
	absoluteDir := t.TempDir()
	t.Chdir(workDir)
	t.Setenv("HOME", homeDir)

	cases := []struct{ name, path, wantDir string }{
		{"no directory portion", "picture.png", workDir},
		{"relative directory", "rel/sub/picture.png", filepath.Join(workDir, "rel/sub")},
		{"absolute directory", filepath.Join(absoluteDir, "picture.png"), absoluteDir},
		{"home-anchored directory", "~/x/picture.png", filepath.Join(homeDir, "x")},
	}
	for _, testCase := range cases {
		inputs := RunFlags{OutPath: params.GetSetIf(true, testCase.path)}

		location, _, err := resolveOutputTarget(&inputs, &catalog.Model{}, params.FlagInputs{}, "")
		if err != nil {
			t.Fatalf("💣 %s: %v", testCase.name, err)
		}

		if !sameDirectory(t, location.Dir, testCase.wantDir) {
			t.Errorf("✗ %s: dir = %q, want %q", testCase.name, location.Dir, testCase.wantDir)
		}
	}

	if !t.Failed() {
		t.Log("✓ output-path selects its directory relative to the invocation directory")
	}
}

// TestApplyOutPathFormat verifies invariant #2: Output path format precedence.
//
// What is being tested:
// When a model consumes output-format, applyOutPathFormat must use the path's format and return one
// ChangeIgnored record for a conflicting explicit value, with the output-path reason. An equal
// format must produce no record. Without a path format or model support, the supplied output-format
// value and its presence must remain unchanged.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestApplyOutPathFormat(t *testing.T) {
	cases := []struct {
		name        string
		parts       artifact.Location
		consumes    bool
		explicit    string
		wantValue   string
		wantPresent bool
		wantRecords int
	}{
		{"token injected on a consuming model", artifact.Location{Ext: ".png", Format: "png"}, true, "", "png", true, 0},
		{"differing explicit value superseded", artifact.Location{Ext: ".png", Format: "png"}, true, "jpeg", "png", true, 1},
		{"equal explicit value joins silently", artifact.Location{Ext: ".jpg", Format: "jpeg"}, true, "JPEG", "jpeg", true, 0},
		{"non-consuming model left untouched", artifact.Location{Ext: ".png", Format: "png"}, false, "jpeg", "jpeg", true, 0},
		{"no token leaves the inputs alone", artifact.Location{Stem: "x"}, true, "jpeg", "jpeg", true, 0},
		{"no token and no explicit value", artifact.Location{Stem: "x"}, true, "", "", false, 0},
	}
	for _, c := range cases {
		userInputs := params.FlagInputs{}
		if c.explicit != "" {
			userInputs[params.FlagTypeOutputFormat] = c.explicit
		}

		model := formatModel(t, c.consumes)
		records := applyOutPathFormat(c.parts, "raw-value", &model, userInputs)

		got, present := userInputs[params.FlagTypeOutputFormat].(string)
		if present != c.wantPresent || got != c.wantValue {
			t.Errorf("✗ %s: output-format input = (%q, %v), want (%q, %v)", c.name, got, present, c.wantValue, c.wantPresent)
		}

		if len(records) != c.wantRecords {
			t.Errorf("✗ %s: %d records, want %d", c.name, len(records), c.wantRecords)

			continue
		}

		if c.wantRecords == 1 {
			r := records[0]
			if r.FlagID != params.FlagTypeOutputFormat || r.Type != params.ChangeIgnored ||
				!strings.Contains(r.Comment, fmt.Sprintf(ReasonSupersededByOutputPath, "raw-value")) {
				t.Errorf("✗ %s: record = %+v, want the output-format supersession", c.name, r)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ the extension's format request injects, supersedes, joins, or stays out per the consumption contract")
	}
}

// TestApplyLandingExt verifies invariant #3: Requested artifact extension compatibility.
//
// What is being tested:
// Given a requested extension of the same format, applyLandingExt must use its spelling, including
// .PNG or .jpeg. For incompatible extensions, it must retain the artifact extension and return a
// ChangeDerived record containing the requested and retained values. Two PNG artifacts falling back
// from .jpg must produce only one record. An empty request must leave all extensions unchanged and
// return no records.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestApplyLandingExt(t *testing.T) {
	cases := []struct {
		name        string
		artifacts   []artifact.Media
		requested   string
		wantExts    []string
		wantRecords int
	}{
		{
			"agreeing class keeps the requested spelling",
			[]artifact.Media{{FileExt: ".png"}},
			".PNG",
			[]string{".PNG"},
			0,
		},
		{
			"jpeg class agreement across spellings",
			[]artifact.Media{{FileExt: ".jpg"}},
			".jpeg",
			[]string{".jpeg"},
			0,
		},
		{
			"a disagreeing extension falls back to the truthful one",
			[]artifact.Media{{FileExt: ".png"}},
			".jpg",
			[]string{".png"},
			1,
		},
		{
			"a video extension never lands on image content",
			[]artifact.Media{{FileExt: ".png"}},
			".mp4",
			[]string{".png"},
			1,
		},
		{
			"no request leaves every artifact untouched",
			[]artifact.Media{
				{FileExt: ".png"},
				{FileExt: ".jpg"},
			},
			"",
			[]string{".png", ".jpg"},
			0,
		},
		{
			"a uniform batch fallback records once",
			[]artifact.Media{
				{FileExt: ".png"},
				{FileExt: ".png"},
			},
			".jpg",
			[]string{".png", ".png"},
			1,
		},
	}
	for _, c := range cases {
		records := applyLandingExt(c.artifacts, c.requested)
		for i, wantExt := range c.wantExts {
			if c.artifacts[i].FileExt != wantExt {
				t.Errorf("✗ %s: artifact %d ext = %q, want %q", c.name, i, c.artifacts[i].FileExt, wantExt)
			}
		}

		if len(records) != c.wantRecords {
			t.Errorf("✗ %s: %d records, want %d", c.name, len(records), c.wantRecords)

			continue
		}

		if c.wantRecords == 1 {
			r := records[0]
			if string(r.FlagID) != "output-path" || r.Type != params.ChangeDerived ||
				r.InputVal != c.requested || r.WireVal != c.wantExts[0] {
				t.Errorf("✗ %s: record = %+v, want the derived extension fallback", c.name, r)
			}
		}
	}

	if !t.Failed() {
		t.Log("✓ the requested extension lands only over agreeing content; fallbacks record as derived")
	}
}

// TestLandingExt verifies invariant #4: Artifact extension selection.
//
// What is being tested:
// landingExt must prefer the path's explicit extension, otherwise derive .png or .jpg from the
// adjusted format for a named file. It must return an empty extension when no format is available
// or the path names only a directory. A numeric output-format must return
// errs.ErrParamValueTypeMismatch.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestLandingExt(t *testing.T) {
	cases := []struct {
		name     string
		parts    artifact.Location
		adjusted params.Values
		want     string
	}{
		{"parsed extension verbatim", artifact.Location{Stem: "x", Ext: ".PNG", Format: "png"}, params.Values{}, ".PNG"},
		{"adjusted format names the file", artifact.Location{Stem: "x"}, params.Values{params.FlagTypeOutputFormat: "png"}, ".png"},
		{"jpeg maps to its canonical .jpg", artifact.Location{Stem: "x"}, params.Values{params.FlagTypeOutputFormat: "jpeg"}, ".jpg"},
		{"no accepted format leaves the truthful chain", artifact.Location{Stem: "x"}, params.Values{}, ""},
		{"a directory-only path never forces an extension", artifact.Location{Dir: "d/"}, params.Values{params.FlagTypeOutputFormat: "png"}, ""},
	}
	for _, c := range cases {
		got, err := landingExt(c.parts, c.adjusted)
		if err != nil || got != c.want {
			t.Errorf("✗ %s: landingExt = (%q, %v), want %q without error", c.name, got, err, c.want)
		}
	}

	if _, err := landingExt(artifact.Location{Stem: "x"}, params.Values{params.FlagTypeOutputFormat: 7}); !errors.Is(err, errs.ErrParamValueTypeMismatch) {
		t.Errorf("✗ a format stored with another type returned %v, want the type mismatch sentinel", err)
	}

	if !t.Failed() {
		t.Log("✓ the landed extension follows path extension, then accepted format, then the truthful chain")
	}
}

// TestResolveOutputTargetDirectory verifies invariant #5: Output directory selection.
//
// What is being tested:
// When OutPath is absent, resolveOutputTarget must select the configured default directory. Given
// the bare filename out.png, it must select the invocation directory. Both calls must succeed.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestResolveOutputTargetDirectory(t *testing.T) {
	workDir := t.TempDir()
	defaultDir := filepath.Join(t.TempDir(), "configured")
	t.Chdir(workDir)

	model := formatModel(t, false)

	for _, c := range []struct {
		name    string
		outPath string
		wantDir string
	}{
		{"a bare filename lands in the invocation directory", "out.png", workDir},
		{"no output path lands in the default directory", "", defaultDir},
	} {
		genInputs := RunFlags{OutPath: params.GetSetIf(c.outPath != "", c.outPath)}

		location, _, err := resolveOutputTarget(&genInputs, &model, params.FlagInputs{}, defaultDir)
		if err != nil {
			t.Fatalf("💣 %s: %v", c.name, err)
		}

		if !sameDirectory(t, location.Dir, c.wantDir) {
			t.Errorf("✗ %s: dir = %q, want %q", c.name, location.Dir, c.wantDir)
		}
	}

	if !t.Failed() {
		t.Log("✓ the default directory applies only when no output path is given")
	}
}

// sameDirectory reports whether two directory paths name the same directory once every symbolic
// link in them is resolved.
func sameDirectory(t *testing.T, first, second string) bool {
	t.Helper()

	firstResolved, err := filepath.EvalSymlinks(first)
	if err != nil {
		t.Fatalf("💣 resolve %q: %v", first, err)
	}

	secondResolved, err := filepath.EvalSymlinks(second)
	if err != nil {
		t.Fatalf("💣 resolve %q: %v", second, err)
	}

	return firstResolved == secondResolved
}

// formatModel returns an image model that optionally supports PNG, JPEG, and WebP output.
func formatModel(test testing.TB, consumes bool) catalog.Model {
	test.Helper()

	model := catalog.Model{ID: "fixt-img", Media: media.Image}
	if consumes {
		model.Params = []params.Definition{{
			ParamID: "output_format", FlagID: params.FlagTypeOutputFormat,
			AllowedValues: []string{"png", "jpeg", "webp"},
		}}
	}

	return model
}
