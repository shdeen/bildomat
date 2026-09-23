package main

// Invariants tested:
//  1. Generation validation errors: bildApp.generate must retain the corresponding
//     model-resolution, input-media, output-file, or flag-type error classification for each
//     invalid input. The missing-input error must not match errs.ErrCLI.
//  2. Correction of ambiguous model specifiers: For ambiguous input, resolveModelInput must render
//     both candidate keys in text mode, accept unique terminal replacements, repeat the candidates
//     after another ambiguous reply, and classify blank or closed terminal input with both
//     missing-input and conflict errors. Nonterminal input must return the conflict; an unknown
//     model must return its own error without a prompt.
//  3. Presence of generated artifacts: Given a generator returning no artifacts, testGenerate must
//     return errs.ErrOutputFile.
//  4. Reporting partially saved artifacts: Given a missing second artifact source, testGenerate
//     must return errs.ErrOutputFile and print exactly one line beginning with the selected output
//     directory.
//  5. Protection of an existing sidecar: Given an existing run.md and requested thoughts,
//     testGenerate must preserve the existing content and create run-02.png and run-02.md.
//  6. Truthful artifact extensions: testGenerate must allow .jpeg for a .jpg artifact without a
//     fallback notice. For a .png artifact requested as .jpg, it must create only the .png path and
//     print the extension adjustment.
//  7. Kling credential boundary: With Kling credentials absent, executeGeneration must return
//     errs.ErrKeyMissing without sending an HTTP request.
//  8. Sourceful credential boundary: With Sourceful credentials absent, executeGeneration must
//     return errs.ErrKeyMissing without sending an HTTP request.
//  9. Cleanup before persistence: After generation, executeGeneration must clean the temporary
//     artifact if extension preparation or directory creation fails. If cleanup is denied, it must
//     retain the source and preserve both failure causes.
//  10. Incomplete adjustment history: Given fewer final changes than prior changes,
//      applyFinalPreparation must succeed and retain the returned duration value and adjustment.
//  11. Notices before submission: executeGeneration must deliver text adjustments once before
//      submission and retain them in final JSON without diagnostics. If text notice delivery fails,
//      it must send no request and retain the write cause.
//  12. Interactive JSON choices: With interactive JSON, resolveModelInput must display both
//      candidates before the replacement prompt and resolve the replacement. With nonterminal JSON,
//      it must return the conflict without diagnostics. Both must leave results empty.
//  13. Help entry of a flag without names: Given a nameless flag, cliRenderFlagEntry must return
//      its description without a less-than character introducing a value placeholder.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/shdeen/bildomat/internal/provider"
	"github.com/urfave/cli/v3"
)

// TestGenerateEarlyErrors verifies invariant #1: Generation validation errors.
//
// What is being tested:
// bildApp.generate must return errs.ErrModelResolve for an unknown model, errs.ErrInputMedia
// without errs.ErrCLI for a missing input image, errs.ErrOutputFile for an output directory beneath
// a regular file, and errs.ErrParamValueTypeMismatch for a string supplied as include-thoughts.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestGenerateEarlyErrors(t *testing.T) {
	loadedCatalog := shippedCatalog(t)
	app := testApp(t, loadedCatalog)

	// an image model declaring the input-media parameter, so a missing input image is read.
	imageInputModel := imageModelWithInputMedia(t, loadedCatalog)
	imageInputSpecifier := qualifiedSpecifier(t, imageInputModel)

	// an unrecognized model input → a model-resolution error, before credentials.
	badModelInputs := RunFlags{Model: params.GetSetIf(true, "totally-bogus"), Prompt: params.GetSetIf(true, "x")}

	var err error

	_, err = app.generate(t.Context(), &badModelInputs, params.FlagInputs{})
	if err == nil || !errors.Is(err, errs.ErrModelResolve) {
		t.Errorf("✗ generate(bad model) = %v, want a wrapped ErrModelResolve", err)
	}

	// a flag value stored with another type → the parameter-value type mismatch, before
	// resolution.
	mistypedInputs := RunFlags{Model: params.GetSetIf(true, imageInputSpecifier), Prompt: params.GetSetIf(true, "x")}

	_, err = app.generate(t.Context(), &mistypedInputs, params.FlagInputs{params.FlagTypeThoughts: "yes"})
	if !errors.Is(err, errs.ErrParamValueTypeMismatch) {
		t.Errorf("✗ generate(mistyped flag value) = %v, want the parameter-value type mismatch", err)
	}

	// a valid model input but a missing input image → an input-media error, before the call.
	missingImageInputs := RunFlags{Model: params.GetSetIf(true, imageInputSpecifier), Prompt: params.GetSetIf(true, "x")}

	_, err = app.generate(t.Context(),
		&missingImageInputs,
		params.FlagInputs{params.FlagTypeInputMedia: []string{filepath.Join(t.TempDir(), "missing.png")}},
	)
	if err == nil || !errors.Is(err, errs.ErrInputMedia) {
		t.Errorf("✗ generate(missing input) = %v, want a wrapped ErrInputMedia", err)
	}

	if errors.Is(err, errs.ErrCLI) {
		t.Errorf("✗ generate(missing input) = %v, want an operational error outside ErrCLI", err)
	}

	// a valid model input, no inputs, but an unwritable output directory (a path under a file)
	// → an output-file error, before the call.
	dir := t.TempDir()

	blocker := filepath.Join(dir, "blocker")
	// #nosec G306 -- permissions are sufficient for this test-owned blocker file.
	if wErr := os.WriteFile(blocker, []byte("x"), 0o644); wErr != nil {
		t.Fatalf("💣 write blocker file: %v", wErr)
	}

	unwritableDirInputs := RunFlags{
		Model:   params.GetSetIf(true, imageInputSpecifier),
		Prompt:  params.GetSetIf(true, "x"),
		OutPath: params.GetSetIf(true, filepath.Join(blocker, "sub")+string(os.PathSeparator)),
	}

	_, err = app.generate(t.Context(),
		&unwritableDirInputs,
		params.FlagInputs{},
	)
	if err == nil || !errors.Is(err, errs.ErrOutputFile) {
		t.Errorf("✗ generate(unwritable outdir) = %v, want a wrapped ErrOutputFile", err)
	}

	if !t.Failed() {
		t.Log("✓ generate() surfaces resolution, input-media, output-directory, and mistyped-flag failures before any provider call")
	}
}

// TestRepromptNonTTY verifies invariant #2: Correction of ambiguous model specifiers.
//
// What is being tested:
// Given dup-model and nonterminal input, resolveModelInput must return
// errs.ErrModelResolveConflict. Its diagnostics must contain exactly the ambiguity heading followed
// by alpha/dup-model and beta/dup-model, without a replacement prompt.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRepromptNonTTY(t *testing.T) {
	loadedCatalog := ambigCatalog(t)

	_, rendered, err := resolveWith(t, loadedCatalog, "dup-model", "alpha/dup-model\n", false)
	if err == nil || !errors.Is(err, errs.ErrModelResolveConflict) {
		t.Errorf("✗ non-TTY ambiguous resolution = %v, want the disambiguation error", err)
	}

	want := fmt.Sprintf(output.AmbiguityHeading, "", "", "", "dup-model") + "\n" +
		" " + " " + fmt.Sprintf(output.AmbiguityCandidate, "", "alpha", "dup-model", "") + "\n" +
		" " + " " + fmt.Sprintf(output.AmbiguityCandidate, "", "beta", "dup-model", "") + "\n"
	if rendered != want {
		t.Errorf("✗ non-TTY candidate view = %q, want %q", rendered, want)
	}

	if !t.Failed() {
		t.Log("✓ the non-TTY path renders the candidates and returns the exit-1 error without reading input")
	}
}

// TestRepromptInteractive verifies invariant #2: Correction of ambiguous model specifiers.
//
// What is being tested:
// Given ambiguous dup-model and terminal input, resolveModelInput must print the replacement prompt
// and accept beta/dup-model with or without a final newline. A second dup-model followed by
// alpha/dup-model must resolve to Alpha and print the ambiguity heading twice. Blank input or EOF
// must return both errs.ErrCLINoModelInput and errs.ErrModelResolveConflict.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRepromptInteractive(t *testing.T) {
	loadedCatalog := ambigCatalog(t)

	bind, rendered, err := resolveWith(t, loadedCatalog, "dup-model", "beta/dup-model\n", true)
	if err != nil {
		t.Errorf("✗ corrected model input failed to resolve: %v", err)
	} else if bind.Provider.ID != "beta" || bind.Model.ID != "dup-model" {
		t.Errorf("✗ corrected model input resolved to %s/%s, want beta/dup-model", bind.Provider.ID, bind.Model.ID)
	}

	if !strings.Contains(rendered, fmt.Sprintf(output.Reprompt, "", "")) {
		t.Errorf("✗ the interactive prompt text is missing: %q", rendered)
	}

	// a second ambiguous answer re-renders and re-prompts until resolution.
	bind, rendered, err = resolveWith(t, loadedCatalog, "dup-model", "dup-model\nalpha/dup-model\n", true)
	if err != nil || bind.Provider.ID != "alpha" {
		t.Errorf("✗ looped reprompt = (%+v, %v), want alpha/dup-model after two rounds", bind, err)
	}

	if strings.Count(rendered, fmt.Sprintf(output.AmbiguityHeading, "", "", "", "dup-model")) != 2 {
		t.Errorf("✗ the candidate list did not re-render on the second round: %q", rendered)
	}

	// a correction ending at EOF without its newline is read like any other.
	bind, _, err = resolveWith(t, loadedCatalog, "dup-model", "beta/dup-model", true)
	if err != nil || bind.Provider.ID != "beta" {
		t.Errorf("✗ unterminated correction = (%+v, %v), want beta/dup-model", bind, err)
	}

	// blank input aborts with the no-model-input failure over the ambiguity.
	_, _, err = resolveWith(t, loadedCatalog, "dup-model", "\n", true)
	if !errors.Is(err, errs.ErrCLINoModelInput) || !errors.Is(err, errs.ErrModelResolveConflict) {
		t.Errorf("✗ blank input = %v, want the no-model-input failure over the unsettled ambiguity", err)
	}

	// a closed stdin aborts the same way.
	_, _, err = resolveWith(t, loadedCatalog, "dup-model", "", true)
	if !errors.Is(err, errs.ErrCLINoModelInput) || !errors.Is(err, errs.ErrModelResolveConflict) {
		t.Errorf("✗ EOF = %v, want the no-model-input failure over the unsettled ambiguity", err)
	}

	if !t.Failed() {
		t.Log("✓ the interactive path re-resolves a corrected model input, loops, and aborts on blank input or EOF")
	}
}

// TestRepromptOnlyAmbig verifies invariant #2: Correction of ambiguous model specifiers.
//
// What is being tested:
// Given unknown model no-such and terminal input containing a valid replacement, resolveModelInput
// must return errs.ErrModelResolveUnknown and leave diagnostics empty.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRepromptOnlyAmbig(t *testing.T) {
	loadedCatalog := ambigCatalog(t)

	_, rendered, err := resolveWith(t, loadedCatalog, "no-such", "alpha/dup-model\n", true)
	if err == nil || !errors.Is(err, errs.ErrModelResolveUnknown) {
		t.Errorf("✗ unknown model input = %v, want the plain fatal unrecognized error", err)
	}

	if rendered != "" {
		t.Errorf("✗ a non-ambiguous failure rendered reprompt output: %q", rendered)
	}

	if !t.Failed() {
		t.Log("✓ only a model input several models answer to is intercepted; every resolution failure stays a plain fatal")
	}
}

// TestGenerateEmptyArtifacts verifies invariant #3: Presence of generated artifacts.
//
// What is being tested:
// Given a generator that returns no artifacts, the generation stages run by testGenerate must
// return an error matching errs.ErrOutputFile.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestGenerateEmptyArtifacts(t *testing.T) {
	loadedCatalog := fixtureCatalog(t, fixtSource(t, "fixt", "Fixture", "FIXT_KEY",
		`{"id": "fixt-img", "name": "fixt-img", "media": "image", "params": []}`))
	genInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "p"),
		OutPath: params.GetSetIf(true, t.TempDir()+string(os.PathSeparator)),
	}

	var err error

	app := testApp(t, loadedCatalog)

	_ = captureBoth(t, func() { err = testGenerate(t, app, &stubGen{test: t}, genInputs, params.FlagInputs{}, false, "") })
	if err == nil || !errors.Is(err, errs.ErrOutputFile) {
		t.Errorf("✗ an empty artifact set = %v, want the categorized disk error", err)
	}

	if !t.Failed() {
		t.Log("✓ an empty artifact set surfaces the categorized disk error")
	}
}

// TestPrintFilenamePartialLanding verifies invariant #4: Reporting partially saved artifacts.
//
// What is being tested:
// Given two artifacts whose second temporary source is missing, the generation stages run by
// testGenerate must return errs.ErrOutputFile. With filename printing enabled, stdout must contain
// exactly one line beginning with the selected output directory.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestPrintFilenamePartialLanding(t *testing.T) {
	loadedCatalog, genInputs, gen, outDir := fixtImgGeneration(t)
	gen.result.Artifacts[1] = artifact.Media{TmpPath: filepath.Join(t.TempDir(), "absent-source"), FileExt: ".png"}

	app := testApp(t, loadedCatalog)

	stdout, err := captureGenerate(t, app, gen, genInputs, params.FlagInputs{}, true, "")
	if err == nil || !errors.Is(err, errs.ErrOutputFile) {
		t.Errorf("✗ the missing second source = %v, want the categorized disk error", err)
	}

	pathLines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(pathLines) != 1 {
		t.Fatalf("💣 stdout carries %d lines, want exactly the 1 landed path: %q", len(pathLines), stdout)
	}

	if !strings.HasPrefix(pathLines[0], outDir+string(os.PathSeparator)) {
		t.Errorf("✗ the landed path line %q is outside the output directory", pathLines[0])
	}

	if !t.Failed() {
		t.Log("✓ a partial landing prints exactly the landed paths")
	}
}

// TestGenerateSidecarAvoidsExistingDestination verifies invariant #5: Protection of an existing
// sidecar.
//
// What is being tested:
// Given an existing run.md containing existing notes and a generation that requests thoughts, the
// generation stages run by testGenerate must succeed, preserve that file's content, and create both
// run-02.png and run-02.md.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestGenerateSidecarAvoidsExistingDestination(t *testing.T) {
	gen := &stubGen{
		test:     t,
		adjusted: params.Values{params.FlagTypeThoughts: true},
		result: generation.Result{
			Artifacts: []artifact.Media{{Data: []byte("IMG"), FileExt: ".png"}},
			Thoughts:  []string{"new thought"},
		},
	}
	loadedCatalog := fixtureCatalog(t, fixtSource(t, "fixt", "Fixture", "FIXT_KEY",
		`{"id": "fixt-img", "name": "fixt-img", "media": "image", "params": [{"flagID": "include-thoughts"}]}`))
	outDir := t.TempDir()

	// #nosec G306 -- permissions are sufficient for this test-owned file.
	if err := os.WriteFile(filepath.Join(outDir, "run.md"), []byte("existing notes"), 0o644); err != nil {
		t.Fatalf("💣 seed existing sidecar: %v", err)
	}

	genInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "a prompt"),
		OutPath: params.GetSetIf(true, filepath.Join(outDir, "run")),
	}
	app := testApp(t, loadedCatalog)
	_ = captureBoth(t, func() {
		if err := testGenerate(t, app, gen, genInputs, params.FlagInputs{params.FlagTypeThoughts: true}, false, ""); err != nil {
			t.Errorf("✗ generate with an existing sidecar failed: %v", err)
		}
	})

	// #nosec G304 -- the seeded sidecar belongs to this test's temporary directory.
	existingContent, err := os.ReadFile(filepath.Join(outDir, "run.md"))
	if err != nil {
		t.Errorf("✗ read existing sidecar: %v", err)
	} else if string(existingContent) != "existing notes" {
		t.Errorf("✗ existing sidecar was changed to %q", existingContent)
	}

	for _, expectedPath := range []string{"run-02.png", "run-02.md"} {
		if _, err := os.Stat(filepath.Join(outDir, expectedPath)); err != nil {
			t.Errorf("✗ expected %s beside the preserved sidecar: %v", expectedPath, err)
		}
	}

	if !t.Failed() {
		t.Log("✓ a sidecar destination is protected only when the run will write that sidecar")
	}
}

// TestGenerateExtOverride verifies invariant #6: Truthful artifact extensions.
//
// What is being tested:
// Given a .jpg artifact requested as pic.jpeg, the generation stages run by testGenerate must
// create pic.jpeg without an output-path fallback notice. Given a .png artifact requested as
// pic.jpg, they must create pic.png, leave pic.jpg absent, and report the output-path adjustment to
// .png. Both runs must succeed.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestGenerateExtOverride(t *testing.T) {
	loadedCatalog := fixtureCatalog(t, fixtSource(t, "fixt", "Fixture", "FIXT_KEY",
		`{"id": "fixt-img", "name": "fixt-img", "media": "image", "params": []}`))

	// Agreeing class: a truthful .jpg artifact is saved under the requested .jpeg spelling.
	agreeGen := &stubGen{
		test:     t,
		adjusted: params.Values{},
		result: generation.Result{
			Artifacts: []artifact.Media{{Data: []byte("IMG"), FileExt: ".jpg"}},
		},
	}
	agreeDir := t.TempDir()
	agreeInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "p"),
		OutPath: params.GetSetIf(true, filepath.Join(agreeDir, "pic.jpeg")),
	}
	app := testApp(t, loadedCatalog)

	agreeOut := captureBoth(t, func() {
		if err := testGenerate(t, app, agreeGen, agreeInputs, params.FlagInputs{}, false, ""); err != nil {
			t.Errorf("✗ agreeing generate failed: %v", err)
		}
	})
	if _, err := os.Stat(filepath.Join(agreeDir, "pic.jpeg")); err != nil {
		t.Errorf("✗ pic.jpeg missing: the agreeing extension did not name the file: %v", err)
	}

	if strings.Contains(agreeOut, "--output-path .jpeg") {
		t.Errorf("✗ an agreeing extension rendered a fallback notice:\n%s", agreeOut)
	}

	// Disagreeing class: a truthful .png artifact refuses the requested .jpg name and is saved
	// truthfully, with the derived notice.
	clashGen := &stubGen{
		test:     t,
		adjusted: params.Values{},
		result: generation.Result{
			Artifacts: []artifact.Media{{Data: []byte("IMG"), FileExt: ".png"}},
		},
	}
	clashDir := t.TempDir()
	clashInputs := RunFlags{
		Model:   params.GetSetIf(true, "fixt-img"),
		Prompt:  params.GetSetIf(true, "p"),
		OutPath: params.GetSetIf(true, filepath.Join(clashDir, "pic.jpg")),
	}

	clashOut := captureBoth(t, func() {
		if err := testGenerate(t, app, clashGen, clashInputs, params.FlagInputs{}, false, ""); err != nil {
			t.Errorf("✗ disagreeing generate failed: %v", err)
		}
	})
	if _, err := os.Stat(filepath.Join(clashDir, "pic.png")); err != nil {
		t.Errorf("✗ pic.png missing: the truthful extension did not land: %v", err)
	}

	if _, err := os.Stat(filepath.Join(clashDir, "pic.jpg")); err == nil {
		t.Errorf("✗ pic.jpg landed over content of another format")
	}

	wantExtNotice := fmt.Sprintf(output.FlagAdjusted, output.OutPathDisplayName, "'.png'")
	if !strings.Contains(clashOut, wantExtNotice) {
		t.Errorf("✗ the extension fallback rendered no adjusted notice:\n%s", clashOut)
	}

	if !t.Failed() {
		t.Log("✓ the named extension lands only over agreeing content; a clash lands truthfully with its notice")
	}
}

// TestKlingMissingCredential verifies invariant #7: Kling credential boundary.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// With Kling's credential absent, executeGeneration must return errs.ErrKeyMissing and send no HTTP
// request to the configured local endpoint.
func TestKlingMissingCredential(t *testing.T) { checkAdapterMissingCredential(t, "kling") }

// TestSourcefulMissingCredential verifies invariant #8: Sourceful credential boundary.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// With Sourceful's credential absent, executeGeneration must return errs.ErrKeyMissing and send no
// HTTP request to the configured local endpoint.
func TestSourcefulMissingCredential(t *testing.T) { checkAdapterMissingCredential(t, "sourceful") }

// TestGenerationFailureBeforePersistence verifies invariant #9: Cleanup before persistence.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// After the generator returns a temporary artifact, executeGeneration must return no saved files
// and errs.ErrParamValueTypeMismatch for a boolean output-format, or errs.ErrOutputFileMkdir for a
// directory blocked by a file. It must remove the temporary source unless permission denies
// removal. In that case, it must retain the source and also return os.ErrPermission.
func TestGenerationFailureBeforePersistence(t *testing.T) {
	for _, failure := range []string{"extension", "directory", "extension and cleanup"} {
		t.Run(failure, func(t *testing.T) {
			source := filepath.Join(t.TempDir(), "generated.png")
			if err := os.WriteFile(source, []byte("generated bytes"), 0o600); err != nil {
				t.Fatalf("💣 generated source fixture: %v", err)
			}

			outputDirectory := t.TempDir()
			adjusted := params.Values{}
			failureClass := errs.ErrParamValueTypeMismatch

			if strings.HasPrefix(failure, "extension") {
				adjusted[params.FlagTypeOutputFormat] = true
			} else {
				outputDirectory = filepath.Join(outputDirectory, "blocker")
				if err := os.WriteFile(outputDirectory, []byte("occupied"), 0o600); err != nil {
					t.Fatalf("💣 output directory blocker: %v", err)
				}

				failureClass = errs.ErrOutputFileMkdir
			}

			if failure == "extension and cleanup" {
				// #nosec G302 -- owner directory permissions establish and restore the test failure.
				if err := os.Chmod(filepath.Dir(source), 0o500); err != nil {
					t.Fatalf("💣 source permissions: %v", err)
				}

				t.Cleanup(func() {
					// #nosec G302 -- owner directory permissions establish and restore the test failure.
					if err := os.Chmod(filepath.Dir(source), 0o700); err != nil {
						t.Errorf("✗ restore source permissions: %v", err)
					}
				})

				if err := os.Remove(source); !errors.Is(err, os.ErrPermission) {
					t.Fatalf("💣 source removal was not denied: %v", err)
				}
			}

			generator := &stubGen{test: t, adjusted: adjusted, result: generation.Result{Artifacts: []artifact.Media{{TmpPath: source, FileExt: ".png"}}}}
			application := &bildApp{catalog: &catalog.Catalog{}, apiKeys: map[string]string{"fixture": "test-key"}}
			run := &generation.Generation{ProvModelPair: catalog.ProvModelPair{Provider: catalog.Provider{ID: "fixture"}, Model: catalog.Model{Media: media.Image}}}

			application.invocation = newInvocation(os.Stdin, io.Discard, io.Discard)
			application.invocation.jsonOutput = true
			application.invocation.outcome = &output.GenerationOutcome{}

			_, savedFiles, err := application.executeGeneration(context.Background(), generator, &run.ProvModelPair, &RunFlags{}, params.FlagInputs{}, nil, artifact.Location{Dir: outputDirectory, Stem: "boat"}, nil, nil)
			if !errors.Is(err, failureClass) || len(savedFiles) != 0 {
				t.Errorf("✗ preparation failure or empty completed facts lost: %+v, %v", savedFiles, err)
			}

			_, statErr := os.Stat(source)
			if failure == "extension and cleanup" {
				if !errors.Is(err, os.ErrPermission) || statErr != nil {
					t.Errorf("✗ denied early cleanup lost its cause or retained source: %v, %v", err, statErr)
				}
			} else if !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("✗ successfully generated source leaked before persistence: %v", statErr)
			}

			if !t.Failed() {
				t.Log("✓ preparation failure cleans command-owned generation artifacts")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ ownership cleanup begins immediately after successful generation")
	}
}

// TestIncompleteAdjustmentHistory verifies invariant #10: Incomplete adjustment history.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a run with two prior changes and a final preparation with one duration change,
// applyFinalPreparation must succeed without panicking, update duration to 8, and record exactly
// one outcome adjustment whose Used value is 8.
func TestIncompleteAdjustmentHistory(t *testing.T) {
	application := &bildApp{catalog: &catalog.Catalog{Flags: params.Flags()}}
	run := &generation.Generation{Preparation: generation.Preparation{Changes: make([]params.Adjustment, 2)}}
	preparedGeneration := &generation.Preparation{
		Params:  params.Values{params.FlagTypeDuration: 8},
		Changes: []params.Adjustment{{FlagID: params.FlagTypeDuration, Type: params.ChangeConformed, InputVal: "5", WireVal: "8"}},
	}

	outcome := &output.GenerationOutcome{}

	application.invocation.outcome = outcome
	if err := application.applyFinalPreparation(run, preparedGeneration); err != nil {
		t.Errorf("✗ final preparation failed: %v", err)
	}

	if run.Params[params.FlagTypeDuration] != 8 || len(outcome.Adjustments) != 1 || outcome.Adjustments[0].Used != "8" {
		t.Errorf("✗ returned preparation facts were lost: %+v, %+v", run.Preparation, outcome.Adjustments)
	}

	if !t.Failed() {
		t.Log("✓ incomplete adjustment history remains reportable without a panic")
	}
}

// TestNoticesBeforeSubmission verifies invariant #11: Notices before submission.
//
// What is being tested:
// For an ignored seed flag, executeGeneration must send one request that fails and preserve that
// failure through reportGeneration. In text mode, diagnostics must contain the notice once before
// submission and once in total after reporting, while results omit it. In JSON mode, diagnostics
// must remain empty and the final document must contain one adjustment. If diagnostics reject
// writes, executeGeneration must send no request and reportGeneration must retain io.ErrClosedPipe.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestNoticesBeforeSubmission(t *testing.T) {
	for _, mode := range []string{"text", "json", "refused diagnostic"} {
		t.Run(mode, func(t *testing.T) {
			diagnosticPath := filepath.Join(t.TempDir(), "stderr")

			// #nosec G304 -- the diagnostic file is inside the test-owned temporary directory.
			diagnostics, err := os.Create(diagnosticPath)
			if err != nil {
				t.Fatalf("💣 diagnostic destination: %v", err)
			}

			t.Cleanup(func() { _ = diagnostics.Close() })

			var submittedDiagnostics []byte

			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				requestCount++

				var readErr error

				// #nosec G304 -- this is the test-owned diagnostic capture.
				submittedDiagnostics, readErr = os.ReadFile(diagnosticPath)
				if readErr != nil {
					t.Errorf("✗ reading submission diagnostics: %v", readErr)
				}

				response.WriteHeader(http.StatusBadRequest)
			}))
			t.Cleanup(server.Close)
			loadedCatalog := shippedCatalog(t)

			description, present := loadedCatalog.Provider("xai")
			if !present || len(description.Models) == 0 {
				t.Fatal("💣 xAI description unavailable")
			}

			description.Config.ImageAPI.GenURL = server.URL

			generator, err := provider.NewProvider(&description)
			if err != nil {
				t.Fatalf("💣 descriptor construction: %v", err)
			}

			pair := catalog.ProvModelPair{Provider: description.Identity(), Model: description.Models[0]}

			var results bytes.Buffer

			application := &bildApp{catalog: loadedCatalog, apiKeys: map[string]string{"xai": "test-credential"}}
			application.invocation = newInvocation(os.Stdin, &results, diagnostics)
			application.invocation.jsonOutput = mode == "json"

			application.invocation.outcome = &output.GenerationOutcome{}
			if mode == "refused diagnostic" {
				application.invocation.stderr = failingWriter{test: t}
			}

			_, _, generationErr := application.executeGeneration(t.Context(), generator, &pair, &RunFlags{}, params.FlagInputs{"seed": 5}, nil, artifact.Location{Dir: t.TempDir(), Stem: "notice"}, nil, nil)

			server.Close()

			reportErr := application.invocation.reportGeneration(time.Now(), pair.Provider.DisplayName, pair.Model.Name, generationErr)
			if mode == "refused diagnostic" {
				if requestCount != 0 || !errors.Is(reportErr, io.ErrClosedPipe) {
					t.Errorf("✗ failed notice delivery sent %d requests or lost cause: %v", requestCount, reportErr)
				}
			} else {
				if requestCount != 1 || generationErr == nil || reportErr == nil {
					t.Errorf("✗ provider failure not retained: %d requests, %v, %v", requestCount, generationErr, reportErr)
				}

				if len(application.invocation.outcome.Adjustments) != 1 {
					t.Fatalf("💣 expected one ignored seed adjustment, got %+v", application.invocation.outcome.Adjustments)
				}

				notice := application.invocation.outcome.Adjustments[0].Notice

				// #nosec G304 -- this is the test-owned diagnostic capture.
				completedDiagnostics, readErr := os.ReadFile(diagnosticPath)
				if readErr != nil {
					t.Fatalf("💣 completed diagnostics: %v", readErr)
				}

				if mode == "json" {
					var document output.GenerationOutcome
					if json.Unmarshal(results.Bytes(), &document) != nil || len(document.Adjustments) != 1 || len(submittedDiagnostics) != 0 || len(completedDiagnostics) != 0 {
						t.Errorf("✗ JSON adjustment routing: %s, stderr %q", results.Bytes(), completedDiagnostics)
					}
				} else if strings.Count(string(submittedDiagnostics), notice) != 1 || strings.Count(string(completedDiagnostics), notice) != 1 || strings.Contains(results.String(), notice) {
					t.Errorf("✗ adjustment routing or timing: before %q, after %q, stdout %q", submittedDiagnostics, completedDiagnostics, results.String())
				}
			}

			if !t.Failed() {
				t.Log("✓ adjustment delivery respects submission timing and report mode")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ notices precede submission and failed notice delivery prevents a request")
	}
}

// TestJSONAmbiguousModelChoices verifies invariant #12: Interactive JSON choices.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given dup-model with JSON enabled and terminal input, resolveModelInput must print both candidate
// keys before the replacement prompt on diagnostics and resolve alpha/dup-model successfully. With
// nonterminal input, it must return errs.ErrModelResolveConflict and leave diagnostics empty. Both
// cases must leave the results stream empty.
func TestJSONAmbiguousModelChoices(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		t.Run(fmt.Sprintf("terminal=%t", interactive), func(t *testing.T) {
			useStdin(t, "alpha/dup-model\n", interactive)

			var (
				results bytes.Buffer
				pair    catalog.ProvModelPair
				err     error
			)

			diagnostics := captureDiagnosticOutput(t, interactive, func() {
				application := &bildApp{catalog: ambigCatalog(t), invocation: newInvocation(os.Stdin, &results, os.Stderr)}
				application.invocation.jsonOutput = true
				pair, err = application.resolveModelInput("dup-model")
			})

			if results.Len() != 0 {
				t.Errorf("✗ resolution wrote into the JSON result stream: %q", results.String())
			}

			if interactive {
				if err != nil || pair.Provider.ID != "alpha" || pair.Model.ID != "dup-model" {
					t.Errorf("✗ replacement did not resolve: %+v, %v", pair, err)
				}

				promptPosition := strings.Index(diagnostics, fmt.Sprintf(output.Reprompt, "", ""))
				for _, key := range []string{"alpha/dup-model", "beta/dup-model"} {
					candidatePosition := strings.Index(diagnostics, key)
					if candidatePosition < 0 || promptPosition < candidatePosition {
						t.Errorf("✗ candidate %q does not precede the prompt: %q", key, diagnostics)
					}
				}
			} else if !errors.Is(err, errs.ErrModelResolveConflict) || len(diagnostics) != 0 {
				t.Errorf("✗ noninteractive JSON ambiguity: %v, diagnostic %q", err, diagnostics)
			}

			if !t.Failed() {
				t.Log("✓ model choices respect the input mode and JSON result stream")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ interactive JSON supplies the information needed to resolve ambiguity")
	}
}

// TestFlagEntryWithoutNames verifies invariant #13: Help entry of a flag without names.
//
// What is being tested:
// Given a string flag without a name, cliRenderFlagEntry must return text containing its
// description and no less-than character that would introduce a value placeholder.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestFlagEntryWithoutNames(t *testing.T) {
	bild := realApp(t)

	entry := bild.cliRenderFlagEntry(&cli.StringFlag{Usage: "A description without a flag name."})
	if !strings.Contains(entry, "A description without a flag name.") || strings.Contains(entry, "<") {
		t.Errorf("✗ the entry of a nameless flag = %q, want its description and no value word", entry)
	}

	if !t.Failed() {
		t.Log("✓ a flag without names renders its description alone")
	}
}

// imageModelWithInputMedia returns the first image model of the catalog that declares the
// input-media parameter.
func imageModelWithInputMedia(t *testing.T, loadedCatalog *catalog.Catalog) catalog.ProvModelPair {
	t.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Model.Media == media.Image && pair.Model.SupportsParam(params.FlagTypeInputMedia) {
			return pair
		}
	}

	t.Fatalf("💣 the shipped catalog holds no image model declaring the input-media parameter")

	return catalog.ProvModelPair{}
}

// checkAdapterMissingCredential executes the command against an adapter with a local endpoint and
// no key.
func checkAdapterMissingCredential(test *testing.T, providerID string) {
	test.Helper()

	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		requests++

		response.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	loadedCatalog := shippedCatalog(test)

	description, present := loadedCatalog.Provider(providerID)
	if !present || len(description.Models) == 0 {
		test.Fatal("💣 adapter description is unavailable")
	}

	description.Config.AdapterAPI.APIBase = server.URL
	test.Setenv(description.APIKeyEnvVar, "")

	var generator generation.Generator

	for _, registration := range providerRegistrations() {
		if registration.ProviderID != providerID {
			continue
		}

		var constructionErr error

		generator, constructionErr = registration.New(&description)
		if constructionErr != nil {
			test.Fatalf("💣 adapter construction failed: %v", constructionErr)
		}
	}

	if generator == nil {
		test.Fatal("💣 adapter constructor is unavailable")
	}

	application := &bildApp{catalog: loadedCatalog}
	pair := catalog.ProvModelPair{Provider: description.Identity(), Model: description.Models[0]}

	application.invocation = newInvocation(os.Stdin, io.Discard, io.Discard)
	application.invocation.outcome = &output.GenerationOutcome{}
	application.invocation.jsonOutput = true

	_, _, generationErr := application.executeGeneration(test.Context(), generator, &pair, &RunFlags{}, nil, nil, artifact.Location{Dir: test.TempDir(), Stem: "missing-key"}, nil, nil)
	if !errors.Is(generationErr, errs.ErrKeyMissing) {
		test.Errorf("✗ missing credential error = %v, want key-missing classification", generationErr)
	}

	if requests != 0 {
		test.Errorf("✗ missing credential sent %d requests", requests)
	}

	if !test.Failed() {
		test.Log("✓ missing adapter credential stops at the command before HTTP")
	}
}

// fixtureApp creates a test application over the supplied catalog.
func fixtureApp(t *testing.T, loadedCatalog *catalog.Catalog) *bildApp {
	t.Helper()
	app := testApp(t, loadedCatalog)

	return app
}

// ambigCatalog returns a catalog in which two providers declare the same bare model identifier.
func ambigCatalog(t *testing.T) *catalog.Catalog {
	t.Helper()

	return fixtureCatalog(t,
		fixtSource(t, "alpha", "Alpha", "ALPHA_KEY", `{"id": "dup-model", "name": "dup-model", "media": "image", "params": []}`),
		fixtSource(t, "beta", "Beta", "BETA_KEY", `{"id": "dup-model", "name": "dup-model", "media": "image", "params": []}`),
	)
}

// resolveWith resolves a model against supplied terminal or pipe input and captures its
// diagnostics.
func resolveWith(t *testing.T, loadedCatalog *catalog.Catalog, modelInput, stdin string, tty bool) (bind catalog.ProvModelPair, rendered string, err error) {
	t.Helper()
	app := testApp(t, loadedCatalog)
	useStdin(t, stdin, tty)
	rendered = captureDiagnosticOutput(t, tty, func() {
		app.invocation = newInvocation(os.Stdin, os.Stdout, os.Stderr)
		bind, err = app.resolveModelInput(modelInput)
	})

	return bind, rendered, err
}
