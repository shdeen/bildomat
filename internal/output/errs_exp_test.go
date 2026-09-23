package output

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/term"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/config"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/provider"
)

// Invariants tested:
//  1. Usage error text fallback: Given ErrCLIPromptMissing without contextual text, PrintUsageError
//     must include prompt missing and the compact Usage heading on stderr.
//  2. Joined credential failures: Given a CredentialError, keyMissingStatement must name its
//     environment variable. Joining an output-write failure before or after that error, or adding
//     an outer wrapper, must leave the credential guidance identical.
//  3. Joined output failure paths: Given two real file-creation failures joined and wrapped
//     together, errorMessage must name both destination paths. The combined error must still match
//     both original filesystem errors.
//  4. Nested configuration details: Given an unknown-provider configuration error joined with
//     another failure and wrapped in prose, configWarningMessage must include the exact
//     configuration path and provider ID. The combined error must still match
//     ErrUserConfigUnknownProvider.
//  5. Operation and output failures: Given a provider rejection, missing credential, or missing
//     media input joined with a failed output creation, errorMessage must retain the provider
//     explanation, credential variable, or media path alongside the output path. It must do so in
//     either branch order and with an outer wrapper.
//  6. Temporary cleanup notice: When artifact.Cleanup cannot remove a temporary source because its
//     directory denies writes, it must return an error matching os.ErrPermission without
//     ErrOutputFileWrite. errorMessage must name the source path exactly once and omit the
//     output-write failure message.
//  7. Owned media read notice: Given errs.FileError for reading a missing generated-media file,
//     errorMessage must return OutputReadFailed formatted with that path. The error must still
//     match ErrOutputFileRead and os.ErrNotExist.
//  8. Arbitrary joined error details: Given arbitrary bytes encoded into distinct operation details
//     and an output path, errorMessage must retain the operation detail and print the output path
//     once despite duplicate output causes, changed branch order, and an outer wrapper. It must
//     omit the wrapper label and contain valid UTF-8 with no controls. The combined error must
//     retain the original operation and permission causes.
//  9. Error line nested chains: For a media-read error containing a transport-size failure,
//     errorMessage must name the media source and omit the transport notice and size detail. For a
//     wrapped provider rejection, it must include the provider's reason and omit the wrapper label.
//     A timeout wrapping a request failure must use the timeout message.
//  10. Error line relayed chains: Given a wrapped response-status failure, errorMessage must name
//      the failed provider request and omit the relayed status and body text. Given a wrapped
//      transport-request failure, it must use the provider's unreachable message and omit the
//      request URL and wrapper label.
//  11. Error line control safety: Given model and media-path details containing terminal controls,
//      errorMessage and rawErrorChain must return valid UTF-8 with no raw control characters. The
//      friendly message must retain the unknown-model wording and visible suffix, and the raw
//      message must retain the input-media classification. rawErrorChain must return no text for
//      nil.
//  12. Error line bare chains: Given bare error sentinels, errorMessage must return nonempty text
//      without the raw chain or empty parentheses, and avoid the unclassified fallback for the
//      tested categories other than model resolution. Media-format and provider-configuration
//      errors must name their path or provider, and an unterminated quoted wrapper must still
//      produce an output-write notice.
//  13. Error message extraction under arbitrary chains: Given arbitrary contextual text and
//      provider names around the tested error sentinels, errorMessage must return nonempty valid
//      UTF-8 with no control characters and must not return the raw error chain verbatim.

// TestUsageErrorTextFallback verifies invariant #1: Usage error text fallback.
//
// What is being tested:
// Given ErrCLIPromptMissing without contextual text, PrintUsageError must include prompt missing
// and the compact Usage heading on stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestUsageErrorTextFallback(t *testing.T) {
	stderr := captureStderr(t, func() {
		_ = PrintUsageError(os.Stderr, errs.ErrCLIPromptMissing, false, term.IsTerminal(int(os.Stderr.Fd())))
	})

	if !strings.Contains(stderr, "prompt missing") || !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ fallback usage output is incomplete: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ a usage error without quoted context still renders a diagnostic")
	}
}

// TestJoinedCredentialFailure verifies invariant #2: Joined credential failures.
//
// What is being tested:
// Given a CredentialError, keyMissingStatement must name its environment variable. Joining an
// output-write failure before or after that error, or adding an outer wrapper, must leave the
// credential guidance identical.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// Kind: permanent.
func TestJoinedCredentialFailure(t *testing.T) {
	const variable = "BILD_TEST_MISSING_KEY"
	t.Setenv(variable, "")
	credentialErr := &errs.CredentialError{EnvVar: variable, ProviderID: "provider"}

	ordinary := keyMissingStatement(credentialErr)
	if !strings.Contains(ordinary, variable) {
		t.Errorf("✗ ordinary guidance lacks the credential variable: %s", ordinary)
	}

	for _, combined := range []error{
		errors.Join(credentialErr, errs.ErrOutputFileWrite),
		errors.Join(errs.ErrOutputFileWrite, credentialErr),
		fmt.Errorf("%q: %w", "generation", errors.Join(credentialErr, errs.ErrOutputFileWrite)),
	} {
		if statement := keyMissingStatement(combined); statement != ordinary {
			t.Errorf("✗ joined failure changed credential guidance: %s", statement)
		}
	}

	if !t.Failed() {
		t.Log("✓ persistence failures preserve the generation's credential-setting guidance")
	}
}

// TestJoinedOutputPaths verifies invariant #3: Joined output failure paths.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given two real file-creation failures joined and wrapped together, errorMessage must name both
// destination paths. The combined error must still match both original filesystem errors.
// Kind: permanent.
func TestJoinedOutputPaths(t *testing.T) {
	firstPath := filepath.Join(t.TempDir(), "missing-first", "first.png")
	secondPath := filepath.Join(t.TempDir(), "missing-second", "second.png")
	// #nosec G304 -- this destination is beneath the test-owned temporary directory.
	_, firstErr := os.OpenFile(firstPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)

	// #nosec G304 -- this destination is beneath the test-owned temporary directory.
	_, secondErr := os.OpenFile(secondPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if firstErr == nil || secondErr == nil {
		t.Fatal("💣 missing destination parents unexpectedly allowed file creation")
	}

	combined := fmt.Errorf("generation output: %w", errors.Join(&os.PathError{Op: "create", Path: firstPath, Err: errors.Join(errs.ErrOutputFileCreate, firstErr)}, &os.PathError{Op: "create", Path: secondPath, Err: errors.Join(errs.ErrOutputFileCreate, secondErr)}))

	message := errorMessage(combined, "provider", "model")
	if !strings.Contains(message, firstPath) || !strings.Contains(message, secondPath) {
		t.Errorf("✗ joined output rendering lost a failed path: %q", message)
	}

	if !errors.Is(combined, firstErr) || !errors.Is(combined, secondErr) {
		t.Errorf("✗ joined output causes lost: %v", combined)
	}

	if !t.Failed() {
		t.Log("✓ joined output failures retain each failed path and cause")
	}
}

// TestNestedConfigContext verifies invariant #4: Nested configuration details.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given an unknown-provider configuration error joined with another failure and wrapped in prose,
// configWarningMessage must include the exact configuration path and provider ID. The combined
// error must still match ErrUserConfigUnknownProvider.
// Kind: permanent.
func TestNestedConfigContext(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yml")
	providerID := "provider.with:punctuation"
	fault := config.UnknownProviderFault(configPath, providerID)
	combined := fmt.Errorf("configuration loading: %w", errors.Join(errors.New("separate failure"), fault))

	message := configWarningMessage(combined)
	if !strings.Contains(message, configPath) || !strings.Contains(message, providerID) {
		t.Errorf("✗ nested configuration warning lost its path or provider: %q", message)
	}

	if !errors.Is(combined, errs.ErrUserConfigUnknownProvider) {
		t.Errorf("✗ configuration classification lost: %v", combined)
	}

	if !t.Failed() {
		t.Log("✓ configuration ownership survives nested and joined causes")
	}
}

// TestOperationAndCleanupNotices verifies invariant #5: Operation and output failures.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a provider rejection, missing credential, or missing media input joined with a failed
// output creation, errorMessage must retain the provider explanation, credential variable, or media
// path alongside the output path. It must do so in either branch order and with an outer wrapper.
// Kind: permanent.
func TestOperationAndCleanupNotices(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "missing", "incomplete.png")

	// #nosec G304 -- this destination is beneath the test-owned temporary directory.
	_, outputErr := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if !errors.Is(outputErr, os.ErrNotExist) {
		t.Fatalf("💣 output failure fixture: %v", outputErr)
	}

	outputErr = &os.PathError{Op: "create", Path: outputPath, Err: errors.Join(errs.ErrOutputFileCreate, outputErr)}

	mediaPath := filepath.Join(t.TempDir(), "missing.jpg")

	_, mediaErr := media.ReadInputs([]string{mediaPath})
	if !errors.Is(mediaErr, errs.ErrInputMediaNotFound) {
		t.Fatalf("💣 input failure fixture: %v", mediaErr)
	}

	const variable = "BILD_TEST_CONTEXT_KEY"
	t.Setenv(variable, "")

	credentialErr := &errs.CredentialError{EnvVar: variable, ProviderID: "fixture"}
	for _, failureCase := range []struct {
		name       string
		primaryErr error
		required   string
	}{
		{"provider", provider.APIErr("fixture", 422, []byte(`{"error":{"message":"provider explanation"}}`)), "provider explanation"},
		{"credential", credentialErr, variable},
		{"input", mediaErr, mediaPath},
	} {
		t.Run(failureCase.name, func(t *testing.T) {
			for _, combined := range []error{
				errors.Join(failureCase.primaryErr, outputErr),
				errors.Join(outputErr, failureCase.primaryErr),
				fmt.Errorf("outer context: %w", errors.Join(failureCase.primaryErr, outputErr)),
			} {
				notice := errorMessage(combined, "Fixture Provider", "Fixture Model")
				if !strings.Contains(notice, failureCase.required) || !strings.Contains(notice, outputPath) {
					t.Errorf("✗ principal operation or cleanup path missing: %s", notice)
				}
			}

			if !t.Failed() {
				t.Log("✓ operation guidance and cleanup path survive joined error composition")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ cleanup details never displace the principal operation's meaning")
	}
}

// TestTemporaryCleanupNotice verifies invariant #6: Temporary cleanup notice.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// When artifact.Cleanup cannot remove a temporary source because its directory denies writes, it
// must return an error matching os.ErrPermission without ErrOutputFileWrite. errorMessage must name
// the source path exactly once and omit the output-write failure message.
// Kind: permanent.
func TestTemporaryCleanupNotice(t *testing.T) {
	directory := t.TempDir()

	sourcePath := filepath.Join(directory, "download.jpg")
	if err := os.WriteFile(sourcePath, []byte("media"), 0o600); err != nil {
		t.Fatalf("💣 create temporary source: %v", err)
	}

	// #nosec G302 -- denying directory writes establishes the removal failure under test.
	if err := os.Chmod(directory, 0o500); err != nil {
		t.Fatalf("💣 deny source cleanup: %v", err)
	}

	// #nosec G302 -- restore access so the test runner can remove its temporary directory.
	t.Cleanup(func() { _ = os.Chmod(directory, 0o700) })

	cleanupErr := artifact.Cleanup([]artifact.Media{{TmpPath: sourcePath}})
	if !errors.Is(cleanupErr, os.ErrPermission) {
		t.Fatalf("💣 source cleanup was not denied: %v", cleanupErr)
	}

	notice := errorMessage(cleanupErr, "", "")
	if errors.Is(cleanupErr, errs.ErrOutputFileWrite) || strings.Contains(notice, formPrefix(t, OutputWriteFailed)) || strings.Count(notice, sourcePath) != 1 {
		t.Errorf("✗ source cleanup misreported as an output write failure: %q (%v)", notice, cleanupErr)
	}

	if !t.Failed() {
		t.Log("✓ source cleanup identifies the leftover file without denying successful output")
	}
}

// TestOwnedMediaReadNotice verifies invariant #7: Owned media read notice.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given errs.FileError for reading a missing generated-media file, errorMessage must return
// OutputReadFailed formatted with that path. The error must still match ErrOutputFileRead and
// os.ErrNotExist.
// Kind: permanent.
func TestOwnedMediaReadNotice(t *testing.T) {
	path := "/tmp/unavailable-generated.png"
	readErr := errs.FileError(errs.FileOpRead, path, errs.ErrOutputFileRead, os.ErrNotExist)
	notice := errorMessage(readErr, "", "")

	expectedNotice := fmt.Sprintf(OutputReadFailed, path)
	if notice != expectedNotice || !errors.Is(readErr, os.ErrNotExist) || !errors.Is(readErr, errs.ErrOutputFileRead) {
		t.Errorf("✗ owned media read explanation or cause: %q, %v", notice, readErr)
	}

	if !t.Failed() {
		t.Log("✓ owned media reads identify the failed path and operation")
	}
}

// FuzzJoinedErrorDetails verifies invariant #8: Arbitrary joined error details.
// Test class: Expanded.
// Test layer: Fuzzing.
// What is being tested:
// Given arbitrary bytes encoded into distinct operation details and an output path, errorMessage
// must retain the operation detail and print the output path once despite duplicate output causes,
// changed branch order, and an outer wrapper. It must omit the wrapper label and contain valid
// UTF-8 with no controls. The combined error must retain the original operation and permission
// causes.
// Kind: permanent.
func FuzzJoinedErrorDetails(f *testing.F) {
	for selection := range uint8(10) {
		f.Add([]byte("sample"), selection)
	}

	f.Fuzz(func(t *testing.T, arbitraryData []byte, selection uint8) {
		suffix := hex.EncodeToString(arbitraryData)
		primaryValue := "PRIMARY_" + suffix
		outputPath := "/OUTPUT_" + suffix

		var primaryFailure error

		switch selection % 5 {
		case 0:
			primaryFailure = &errs.MediaError{Source: primaryValue, Cause: errs.ErrInputMediaNotFound}
		case 1:
			primaryFailure = &errs.ProviderError{Message: primaryValue, Cause: errs.ErrResponseGen}
		case 2:
			primaryFailure = &errs.ConfigError{Provider: "provider", Problem: primaryValue, Cause: errs.ErrProvConfigInvalid}
		case 3:
			primaryFailure = &errs.CredentialError{EnvVar: primaryValue, ProviderID: "provider"}
		case 4:
			primaryFailure = errors.New(primaryValue)
		}

		outputFailure := &os.PathError{Op: "write", Path: outputPath, Err: errors.Join(errs.ErrOutputFileWrite, os.ErrPermission)}

		combinedFailure := errors.Join(primaryFailure, outputFailure, outputFailure)
		if selection >= 5 {
			combinedFailure = errors.Join(outputFailure, primaryFailure, outputFailure)
		}

		combinedFailure = fmt.Errorf("outer-request-label: %w", combinedFailure)

		message := errorMessage(combinedFailure, "provider", "model")
		if !strings.Contains(message, primaryValue) || strings.Count(message, outputPath) != 1 || strings.Contains(message, "outer-request-label") {
			t.Errorf("✗ joined context lost or duplicated: %q", message)
		}

		if !errors.Is(combinedFailure, primaryFailure) || !errors.Is(combinedFailure, os.ErrPermission) {
			t.Errorf("✗ original operation or filesystem cause lost: %v", combinedFailure)
		}

		checkRenderedBytes(t, message, "joined error details")

		if !t.Failed() {
			t.Log("✓ joined failures retain values, distinct paths, and causes")
		}
	})
}

// TestErrorLineNestedChains verifies invariant #9: Error line nested chains.
//
// What is being tested:
// For a media-read error containing a transport-size failure, errorMessage must name the media
// source and omit the transport notice and size detail. For a wrapped provider rejection, it must
// include the provider's reason and omit the wrapper label. A timeout wrapping a request failure
// must use the timeout message.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestErrorLineNestedChains(t *testing.T) {
	oversized := &errs.MediaError{Source: "/x/huge.bin", Cause: errors.Join(errs.ErrInputMediaRead, fmt.Errorf("%q, %w", "limit 67108864 bytes", errs.ErrTransportSize))}

	got := errorMessage(oversized, "Fixture Provider", "")
	if !strings.Contains(got, fmt.Sprintf(InputMediaSourceInvalid, "/x/huge.bin")) {
		t.Errorf("✗ oversized input read = %q, want the input-media message naming the source", got)
	}

	if strings.Contains(got, fmt.Sprintf(TransportUnreachable, "Fixture Provider")) || strings.Contains(got, "limit 67108864 bytes") {
		t.Errorf("✗ chain detail leaked into the input-media message: %q", got)
	}

	labeled := fmt.Errorf("%q, %w, %w", "fixture-provider (fixture-video)", errs.ErrResponseGen,
		&errs.ProviderError{Message: "job-123 (failed): frame rejected by safety system", Cause: errs.ErrResponseGen})

	got = errorMessage(labeled, "Fixture Provider", "Fixture Video")
	if !strings.Contains(got, formBody(t, ProviderServerError)) || !strings.Contains(got, "frame rejected by safety system") {
		t.Errorf("✗ label-wrapped diagnostic = %q, want the formula with the deepest provider reason", got)
	}

	if strings.Contains(got, "fixture-provider (fixture-video)") {
		t.Errorf("✗ the internal label leaked into the mapped message: %q", got)
	}

	deadline := fmt.Errorf("%q, %w, %w", "fixture-provider (fixture-video)", errs.ErrTransportTimeout,
		fmt.Errorf("%q, %w, %w", "after 4m0s", errs.ErrTransportTimeout,
			fmt.Errorf("%q, %w, %w", "https://api.example.test/jobs/1", errs.ErrTransportRequest, errors.New("read: reset"))))

	got = errorMessage(deadline, "Fixture Provider", "")
	if !strings.Contains(got, fmt.Sprintf(TransportTimeout, "Fixture Provider")) {
		t.Errorf("✗ deadline wrapping its transient fell through to the general transport notice: %q", got)
	}

	if !t.Failed() {
		t.Log("✓ nested chains classify by category and leak no chain detail")
	}
}

// TestErrorLineRelayedChains verifies invariant #10: Error line relayed chains.
//
// What is being tested:
// Given a wrapped response-status failure, errorMessage must name the failed provider request and
// omit the relayed status and body text. Given a wrapped transport-request failure, it must use the
// provider's unreachable message and omit the request URL and wrapper label.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestErrorLineRelayedChains(t *testing.T) {
	relayed := fmt.Errorf("%q, %w, %w", "fixture-provider (fixture-video)", errs.ErrResponseStatus,
		fmt.Errorf("%q, %w", "fixture-provider (fixture-video): status 503: overloaded", errs.ErrResponseStatus))

	got := errorMessage(relayed, "Fixture Provider", "Fixture Video")
	if !strings.Contains(got, "The request to Fixture Provider failed") {
		t.Errorf("✗ relayed status = %q, want the response classification", got)
	}

	if strings.Contains(got, "status 503") || strings.Contains(got, "overloaded") {
		t.Errorf("✗ the relayed status detail leaked into the mapped message: %q", got)
	}

	wrappedTransport := fmt.Errorf("%q, %w", "fixture-provider (fixture-image)",
		fmt.Errorf("%q, %w, %w", "https://api.example.test/gen", errs.ErrTransportRequest, errors.New("dial tcp: refused")))

	got = errorMessage(wrappedTransport, "Fixture Provider", "")
	if !strings.Contains(got, fmt.Sprintf(TransportUnreachable, "Fixture Provider")) {
		t.Errorf("✗ label-wrapped transport = %q, want the unreachable classification", got)
	}

	if strings.Contains(got, "https://api.example.test/gen") || strings.Contains(got, "fixture-provider (fixture-image)") {
		t.Errorf("✗ transport chain detail leaked into the mapped message: %q", got)
	}

	if !t.Failed() {
		t.Log("✓ relayed chains classify by their sentinels and leak no chain detail")
	}
}

// TestErrorLineControlSafety verifies invariant #11: Error line control safety.
//
// What is being tested:
// Given model and media-path details containing terminal controls, errorMessage and rawErrorChain
// must return valid UTF-8 with no raw control characters. The friendly message must retain the
// unknown-model wording and visible suffix, and the raw message must retain the input-media
// classification. rawErrorChain must return no text for nil.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestErrorLineControlSafety(t *testing.T) {
	oscInput := "mo\x1b]52;c;evil\x07del"
	unknown := &errs.ModelError{Specifier: oscInput, Cause: errs.ErrModelResolveUnknown}
	got := errorMessage(unknown, "", "")
	checkRenderedBytes(t, got, "the friendly message")

	if !strings.Contains(got, formTail(t, ModelUnknown)) {
		t.Errorf("✗ the control-bearing input lost its classification: %q", got)
	}

	if !strings.Contains(got, "del") {
		t.Errorf("✗ the control-bearing input lost its visible content: %q", got)
	}

	oscPath := "/tmp/\x1b]52;c;evil\x07img.png"
	notFound := &errs.MediaError{Source: oscPath, Cause: errors.Join(errs.ErrInputMediaNotFound, errors.New("open "+oscPath+": no such file or directory"))}

	raw := rawErrorChain(notFound)
	checkRenderedBytes(t, raw, "the raw diagnostic chain")

	if !strings.Contains(raw, "input media") {
		t.Errorf("✗ the raw diagnostic chain lost its classification text: %q", raw)
	}

	if rawErrorChain(nil) != "" {
		t.Errorf("✗ rawErrorChain(nil) rendered output")
	}
}

// TestErrorLineBareChains verifies invariant #12: Error line bare chains.
//
// What is being tested:
// Given bare error sentinels, errorMessage must return nonempty text without the raw chain or empty
// parentheses, and avoid the unclassified fallback for the tested categories other than model
// resolution. Media-format and provider-configuration errors must name their path or provider, and
// an unterminated quoted wrapper must still produce an output-write notice.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestErrorLineBareChains(t *testing.T) {
	bare := []struct {
		name string
		err  error
	}{
		{"bare transport root", errs.ErrTransport},
		{"bare generation failure", errs.ErrResponseGen},
		{"bare response root", errs.ErrResponse},
		{"bare input-media root", errs.ErrInputMedia},
		{"bare provider-config root", errs.ErrProvConfig},
		{"bare working-dir sentinel", errs.ErrProcessWorkingDir},
		{"bare resolution root", errs.ErrModelResolve},
	}

	unclassified := errorMessage(errors.New("unclassified bare-chain test error"), "", "")
	for _, testCase := range bare {
		got := errorMessage(testCase.err, "", "")
		if got == "" {
			t.Errorf("✗ %s rendered no user notice", testCase.name)
		}

		if got == unclassified && !errors.Is(testCase.err, errs.ErrModelResolve) {
			t.Errorf("✗ %s fell through to the unclassified notice", testCase.name)
		}

		if got == testCase.err.Error() {
			t.Errorf("✗ %s exposed the raw error chain", testCase.name)
		}

		if strings.Contains(got, "()") {
			t.Errorf("✗ %s rendered an empty parenthetical: %q", testCase.name, got)
		}
	}

	plainMIME := &errs.MediaError{Source: "/plain.png", Cause: errs.ErrInputMediaMIME}
	if got := errorMessage(plainMIME, "", ""); !strings.Contains(got, fmt.Sprintf(InputMediaFormat, "/plain.png")) {
		t.Errorf("✗ the MIME message is not the mapped format notice naming the path: %q", got)
	}

	headOwner := &errs.ConfigError{Provider: "somefault", Problem: "somefault: detail", Cause: errs.ErrProvConfigInvalid}
	if got := errorMessage(headOwner, "", ""); !strings.Contains(got, fmt.Sprintf(ProviderConfigBroken, "somefault")) {
		t.Errorf("✗ a config detail without a .json token did not fall back to its leading token: %q", got)
	}

	unterminated := fmt.Errorf(`"unterminated %w`, errs.ErrOutputFile)
	if got := errorMessage(unterminated, "", ""); !strings.Contains(got, formPrefix(t, OutputWriteFailed)) {
		t.Errorf("✗ an unterminated quoted context broke the output-file line: %q", got)
	}

	if !t.Failed() {
		t.Log("✓ bare and malformed chains render classified lines with empty details dropped")
	}
}

// FuzzErrorMessage verifies invariant #13: Error message extraction under arbitrary chains.
//
// What is being tested:
// Given arbitrary contextual text and provider names around the tested error sentinels,
// errorMessage must return nonempty valid UTF-8 with no control characters and must not return the
// raw error chain verbatim.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzErrorMessage(f *testing.F) {
	f.Add("some context", "Fixture Provider", uint8(0))
	f.Add("/tmp/x.png (image/gif; want PNG, JPEG, or WebP)", "", uint8(7))
	f.Add(`quoted "inner" text`, "Fixture Provider", uint8(12))
	f.Add("", "Fixture Labs", uint8(3))
	f.Add("/tmp/\x9bevil.png", "", uint8(3))

	sentinels := []error{
		errs.ErrKeyMissing, errs.ErrCanceled, errs.ErrProcessWorkingDir,
		errs.ErrInputMediaNotFound, errs.ErrInputMediaMIME, errs.ErrInputMediaRead,
		errs.ErrModelResolveUnknown,
		errs.ErrModelResolveConflict, errs.ErrProvConfigDecode,
		errs.ErrOutputFileMkdir, errs.ErrTransportTimeout, errs.ErrResponseGen,
		errs.ErrResponseStatus, errs.ErrTransportRequest, errs.ErrCLIFlagParse,
	}

	f.Fuzz(func(t *testing.T, chainContext, providerDisplayName string, pick uint8) {
		sentinel := sentinels[int(pick)%len(sentinels)]
		err := fmt.Errorf("%q, %w", chainContext, sentinel)

		got := errorMessage(err, providerDisplayName, "Model X")
		if got == "" {
			t.Errorf("✗ a classified error rendered nothing (sentinel %v, context %q)", sentinel, chainContext)
		}

		if got == err.Error() {
			t.Errorf("✗ the raw chain rendered verbatim: %q", got)
		}

		checkRenderedBytes(t, got, "the rendered message")
	})
}

// formPrefix returns the text before the first formatting verb in a message template.
func formPrefix(t testing.TB, form string) string {
	t.Helper()

	prefix, _, _ := strings.Cut(form, "%")

	return prefix
}

// formTail returns the text after the last formatting verb in a message template, trimmed of
// surrounding whitespace.
func formTail(t testing.TB, form string) string {
	t.Helper()

	segments := formVerbPattern(t).Split(form, -1)
	if len(segments) == 0 {
		return ""
	}

	return strings.TrimSpace(segments[len(segments)-1])
}
