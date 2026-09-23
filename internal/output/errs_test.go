package output

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/provider"
)

// Invariants tested:
//  1. Usage error output: Given a UsageError for --bogus, PrintUsageError must write the offending
//     flag and compact Usage heading to stderr, omit the full help options, and write nothing to
//     stdout.
//  2. Raw usage error output: Given the same UsageError, PrintUsageError must produce different
//     normal and raw text on stderr. Both forms must include compact usage and omit the full
//     OPTIONS section; raw output must include the internal CLI error chain.
//  3. Operational error output: Given an unknown-model error, PrintError must name the rejected
//     specifier on stderr, omit compact usage, and write nothing to stdout.
//  4. Input media time notice: Given a MediaError with a frame-anchor conflict, errorMessage must
//     return that specific problem text exactly. Given ErrInputMediaTime without details, it must
//     return FrameTimeInvalidGeneric.
//  5. Input media unsendable notice: Given an unsendable-input MediaError wrapped in an outer
//     diagnostic, errorMessage must return the inner problem text exactly and omit the outer
//     diagnostic.
//  6. Error notices use stderr: When writing to stderr, PrintAmbiguity must include the ambiguous
//     model specifier and PrintReprompt must include the specifier prompt.
//  7. User configuration warnings: Given a user-configuration read, decode, unknown-setting,
//     unknown-provider, or location error, PrintUserConfigWarning must write exactly one stderr
//     line and nothing to stdout. Read and decode warnings must name the path; unknown-setting and
//     unknown-provider warnings must also name the offending key or provider ID.
//  8. Unclassified error fallback: Given an unclassified error, errorMessage must return its own
//     message; given the wrapped flag-parser fixture, it must return the innermost parser message
//     without the wrapper. Given nil, it must return an empty string.
//  9. Ambiguous model selection details: Given two matching models, PrintAmbiguity must write the
//     configured heading with the supplied specifier, followed by the two fully qualified model
//     keys in input order, each indented by two spaces.
//  10. Error line mapping: For the credential, cancellation, transport, provider-response, media,
//      model-selection, output-file, configuration, and working-directory fixtures, errorMessage
//      must include the specified user-facing message and relevant values. It must omit each case's
//      internal details, avoid both the raw chain and unclassified fallback, and contain valid
//      UTF-8 with no control characters.

// TestPrintUsageError verifies invariant #1: Usage error output.
//
// What is being tested:
// Given a UsageError for --bogus, PrintUsageError must write the offending flag and compact Usage
// heading to stderr, omit the full help options, and write nothing to stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintUsageError(t *testing.T) {
	usageErr := &errs.UsageError{Text: "--bogus", Cause: errs.ErrCLIFlagParse}

	var stderr string

	stdout := captureStdout(t, func() {
		stderr = captureStderr(t, func() {
			_ = PrintUsageError(os.Stderr, usageErr, false, term.IsTerminal(int(os.Stderr.Fd())))
		})
	})

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "--bogus") || !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ stderr lacks the error or compact usage: %q", stderr)
	}

	if strings.Contains(stderr, "OPTIONS:") || strings.Contains(stderr, "--aspect-ratio") {
		t.Errorf("✗ stderr contains the full help page: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ a usage error renders compact usage only on stderr")
	}
}

// TestPrintUsageErrorRaw verifies invariant #2: Raw usage error output.
//
// What is being tested:
// Given the same UsageError, PrintUsageError must produce different normal and raw text on stderr.
// Both forms must include compact usage and omit the full OPTIONS section; raw output must include
// the internal CLI error chain.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintUsageErrorRaw(t *testing.T) {
	usageErr := &errs.UsageError{Text: "--bogus", Cause: errs.ErrCLIFlagParse}
	normalOutput := captureStderr(t, func() {
		_ = PrintUsageError(os.Stderr, usageErr, false, term.IsTerminal(int(os.Stderr.Fd())))
	})
	rawOutput := captureStderr(t, func() {
		_ = PrintUsageError(os.Stderr, usageErr, true, term.IsTerminal(int(os.Stderr.Fd())))
	})

	if normalOutput == rawOutput {
		t.Errorf("✗ raw mode did not change the error text: %q", rawOutput)
	}

	outputs := map[string]string{"normal": normalOutput, "raw": rawOutput}
	for outputMode, renderedOutput := range outputs {
		if !strings.Contains(renderedOutput, "Usage:") {
			t.Errorf("✗ %s output lacks compact usage: %q", outputMode, renderedOutput)
		}

		if strings.Contains(renderedOutput, "OPTIONS:") {
			t.Errorf("✗ %s output contains the full help page: %q", outputMode, renderedOutput)
		}
	}

	if !strings.Contains(rawOutput, "CLI:") {
		t.Errorf("✗ raw output lacks the internal CLI chain: %q", rawOutput)
	}

	if !t.Failed() {
		t.Log("✓ raw mode changes only the error text inside compact usage")
	}
}

// TestPrintError verifies invariant #3: Operational error output.
//
// What is being tested:
// Given an unknown-model error, PrintError must name the rejected specifier on stderr, omit compact
// usage, and write nothing to stdout.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintError(t *testing.T) {
	operationalErr := &errs.ModelError{Specifier: "no-such-model", Cause: errs.ErrModelResolveUnknown}

	var stderr string

	stdout := captureStdout(t, func() {
		stderr = captureStderr(t, func() {
			_ = PrintError(os.Stderr, operationalErr, "", "", false, term.IsTerminal(int(os.Stderr.Fd())))
		})
	})

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "no-such-model") {
		t.Errorf("✗ stderr does not identify the failed input: %q", stderr)
	}

	if strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ operational error output contains compact usage: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ an operational error renders only its classified message on stderr")
	}
}

// TestInputMediaTimeNotice verifies invariant #4: Input media time notice.
//
// What is being tested:
// Given a MediaError with a frame-anchor conflict, errorMessage must return that specific problem
// text exactly. Given ErrInputMediaTime without details, it must return FrameTimeInvalidGeneric.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestInputMediaTimeNotice(t *testing.T) {
	specific := fmt.Sprintf(generation.FrameAnchorConflict, "first:one.png", "first:two.png", "first")

	err := &errs.MediaError{Problem: specific, Cause: errs.ErrInputMediaTime}
	if notice := errorMessage(err, "", ""); notice != specific {
		t.Errorf("✗ frame-time notice = %q, want the specific statement %q", notice, specific)
	}

	generic := FrameTimeInvalidGeneric
	if notice := errorMessage(errs.ErrInputMediaTime, "", ""); notice != generic {
		t.Errorf("✗ a detail-less frame-time chain = %q, want the generic timing message %q", notice, generic)
	}

	if !t.Failed() {
		t.Log("✓ frame-timing failures surface their specific statement, with the generic fallback for detail-less chains")
	}
}

// TestInputMediaUnsendableNotice verifies invariant #5: Input media unsendable notice.
//
// What is being tested:
// Given an unsendable-input MediaError wrapped in an outer diagnostic, errorMessage must return the
// inner problem text exactly and omit the outer diagnostic.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestInputMediaUnsendableNotice(t *testing.T) {
	want := "reference.mp4 cannot be sent by this route"
	inner := &errs.MediaError{Problem: want, Cause: errs.ErrInputMediaUnsendable}
	err := fmt.Errorf("%q, %w", "outer diagnostic", inner)

	if notice := errorMessage(err, "", ""); notice != want {
		t.Errorf("✗ unsendable-input notice = %q, want deepest detail %q", notice, want)
	}

	if !t.Failed() {
		t.Log("✓ an unsendable input renders its deepest quoted detail verbatim")
	}
}

// TestErrorNoticesUseStderr verifies invariant #6: Error notices use stderr.
//
// What is being tested:
// When writing to stderr, PrintAmbiguity must include the ambiguous model specifier and
// PrintReprompt must include the specifier prompt.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestErrorNoticesUseStderr(t *testing.T) {
	originalInput := os.Stdin

	emptyInput, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("💣 empty prompt input: %v", err)
	}

	os.Stdin = emptyInput

	t.Cleanup(func() { os.Stdin = originalInput; _ = emptyInput.Close() })
	stderr := captureStderr(t, func() {
		_ = PrintAmbiguity(os.Stderr, "dup-model", nil, term.IsTerminal(int(os.Stderr.Fd())))
		_ = PrintReprompt(os.Stderr, term.IsTerminal(int(os.Stderr.Fd())))
	})

	if !strings.Contains(stderr, "dup-model") {
		t.Errorf("✗ ambiguity output does not identify the model input: %q", stderr)
	}

	if !strings.Contains(stderr, "specifier") {
		t.Errorf("✗ reprompt output is missing: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ ambiguity and reprompt notices own their stderr destination")
	}
}

// TestPrintUserConfigWarning verifies invariant #7: User configuration warnings.
//
// What is being tested:
// Given a user-configuration read, decode, unknown-setting, unknown-provider, or location error,
// PrintUserConfigWarning must write exactly one stderr line and nothing to stdout. Read and decode
// warnings must name the path; unknown-setting and unknown-provider warnings must also name the
// offending key or provider ID.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestPrintUserConfigWarning(t *testing.T) {
	configPath := "/scratch-home/.bildomat/config.yml"
	cases := []struct {
		name       string
		fault      error
		wantValues []string
	}{
		{
			"unreadable file",
			&errs.ConfigError{Path: configPath, Cause: errors.Join(errs.ErrUserConfigRead, errs.ErrProcess)},
			[]string{configPath},
		},
		{
			"undecodable file",
			&errs.ConfigError{Path: configPath, Cause: errors.Join(errs.ErrUserConfigDecode, errs.ErrProcess)},
			[]string{configPath},
		},
		{
			"unknown setting",
			&errs.ConfigError{Path: configPath, Setting: "default-modle", Cause: errs.ErrUserConfigUnknownSetting},
			[]string{configPath, "default-modle"},
		},
		{
			"unknown provider",
			&errs.ConfigError{Path: configPath, Provider: "not-a-provider", Cause: errs.ErrUserConfigUnknownProvider},
			[]string{configPath, "not-a-provider"},
		},
		{
			"unresolved location",
			&errs.ConfigError{Path: ".bildomat/config.yml", Cause: errors.Join(errs.ErrUserConfigLocate, errs.ErrProcess)},
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stderr string

			stdout := captureStdout(t, func() {
				stderr = captureStderr(t, func() {
					_ = PrintUserConfigWarning(os.Stderr, c.fault, term.IsTerminal(int(os.Stderr.Fd())))
				})
			})

			if stdout != "" {
				t.Errorf("✗ rendering wrote to stdout: %q", stdout)
			}

			if stderr == "" || strings.Count(stderr, "\n") != 1 {
				t.Errorf("✗ rendering wrote %q, want exactly one stderr line", stderr)
			}

			for _, wantValue := range c.wantValues {
				if !strings.Contains(stderr, wantValue) {
					t.Errorf("✗ the warning %q does not name %q", stderr, wantValue)
				}
			}

			if !t.Failed() {
				t.Logf("✓ the %s fault renders one stderr warning naming its values, and stdout stays untouched", c.name)
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ each classified fault renders one stderr warning naming its specific value, and stdout stays untouched")
	}
}

// TestErrorLineInnermostFallback verifies invariant #8: Unclassified error fallback.
//
// What is being tested:
// Given an unclassified error, errorMessage must return its own message; given the wrapped
// flag-parser fixture, it must return the innermost parser message without the wrapper. Given nil,
// it must return an empty string.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestErrorLineInnermostFallback(t *testing.T) {
	if got := errorMessage(errors.New("mystery"), "Fixture Provider", ""); got != "mystery" {
		t.Errorf("✗ a bare unclassified error = %q, want its own message", got)
	}

	libraryChain := fmt.Errorf("%q, %w, %w", "flag provided but not defined: -bogus",
		errors.New("outer wrapper"), errors.New("flag provided but not defined: -bogus"))

	got := errorMessage(libraryChain, "Fixture Provider", "")
	if got != "flag provided but not defined: -bogus" {
		t.Errorf("✗ a wrapped unclassified chain = %q, want the innermost message", got)
	}

	if got := errorMessage(nil, "Fixture Provider", ""); got != "" {
		t.Errorf("✗ errorMessage(nil) = %q, want empty", got)
	}

	if !t.Failed() {
		t.Log("✓ an unclassified error renders its innermost message; nil renders nothing")
	}
}

// TestAmbigView verifies invariant #9: Ambiguous model selection details.
//
// What is being tested:
// Given two matching models, PrintAmbiguity must write the configured heading with the supplied
// specifier, followed by the two fully qualified model keys in input order, each indented by two
// spaces.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestAmbigView(t *testing.T) {
	modelMatches := []catalog.ProvModelPair{
		{Provider: catalog.Provider{ID: "alpha"}, Model: catalog.Model{ID: "dup-model"}},
		{Provider: catalog.Provider{ID: "beta"}, Model: catalog.Model{ID: "dup-model"}},
	}

	stderr := captureStderr(t, func() {
		_ = PrintAmbiguity(os.Stderr, "dup-model", modelMatches, term.IsTerminal(int(os.Stderr.Fd())))
	})

	heading := fmt.Sprintf(AmbiguityHeading, "", "", "", "dup-model")

	want := heading + "\n  alpha/dup-model\n  beta/dup-model\n"
	if stderr != want {
		t.Errorf("✗ PrintAmbiguity rendered %q, want %q", stderr, want)
	}

	if !t.Failed() {
		t.Log("✓ the disambiguation view renders the model specifier line and indented candidates")
	}
}

// TestErrorLineMapping verifies invariant #10: Error line mapping.
//
// What is being tested:
// For the credential, cancellation, transport, provider-response, media, model-selection,
// output-file, configuration, and working-directory fixtures, errorMessage must include the
// specified user-facing message and relevant values. It must omit each case's internal details,
// avoid both the raw chain and unclassified fallback, and contain valid UTF-8 with no control
// characters.
//
// Test class: Expanded.
// Test layer: Coverage.
func TestErrorLineMapping(t *testing.T) {
	for _, mappingCase := range messageMappingCases(t) {
		checkMessageMappingCase(t, mappingCase)
	}

	if !t.Failed() {
		t.Log("✓ every message-table case renders its user-facing contract")
	}
}

// messageMappingCase pairs an error with required and forbidden message fragments.
type messageMappingCase struct {
	name         string
	err          error
	providerName string
	modelName    string
	wantParts    []string
	wantAbsent   []string
}

// captureStderr runs fn with os.Stderr redirected and returns what was written.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("💣 os.Pipe: %v", err)
	}

	os.Stderr = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("💣 close pipe: %v", err)
	}

	os.Stderr = old

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("💣 read captured stderr: %v", err)
	}

	return string(out)
}

// messageMappingCases returns the shared error-to-user-message mapping cases.
func messageMappingCases(test testing.TB) []messageMappingCase {
	test.Helper()

	return []messageMappingCase{
		{
			name:         "missing credential",
			err:          &errs.CredentialError{EnvVar: "FIXTURE_API_KEY"},
			providerName: "Fixture Provider",
			wantParts:    []string{fmt.Sprintf(CredentialMissingProvider, "FIXTURE_API_KEY "+errs.ErrKeyMissing.Error(), "Fixture Provider")},
		},
		{
			name:         "missing credential without a run provider",
			err:          &errs.CredentialError{EnvVar: "OTHER_FIXTURE_API_KEY"},
			providerName: "",
			wantParts:    []string{fmt.Sprintf(CredentialMissing, "OTHER_FIXTURE_API_KEY "+errs.ErrKeyMissing.Error())},
		},
		{
			name:         "canceled",
			err:          fmt.Errorf("%w, %w", errs.ErrCanceled, errors.New("context canceled")),
			providerName: "Fixture Labs",
			wantParts:    []string{GenerationCanceled},
			wantAbsent:   []string{"context canceled"},
		},
		{
			name:         "poll deadline",
			err:          fmt.Errorf("%q, %w", "after 4m0s", errs.ErrTransportTimeout),
			providerName: "Fixture Provider",
			wantParts:    []string{fmt.Sprintf(TransportTimeout, "Fixture Provider")},
			wantAbsent:   []string{"transport:", "after 4m0s"},
		},
		{
			name:         "other transport failure",
			err:          fmt.Errorf("%q, %w, %w", "https://api.example.test/gen", errs.ErrTransportRequest, errors.New("dial tcp: connection refused")),
			providerName: "Fixture Provider",
			wantParts:    []string{fmt.Sprintf(TransportUnreachable, "Fixture Provider")},
			wantAbsent:   []string{"transport:", "https://api.example.test/gen", "connection refused"},
		},
		{
			name:         "provider-reported failure",
			err:          &errs.ProviderError{Message: "Request Moderated", Cause: errs.ErrResponseGen},
			providerName: "Fixture Labs",
			modelName:    "Fixture Image Model",
			wantParts:    []string{fmt.Sprintf(ProviderServerError, "Fixture Labs", "Fixture Image Model", "Request Moderated")},
			wantAbsent:   []string{"provider response:"},
		},
		{
			name:         "provider server error with an extracted message",
			err:          provider.APIErr("fixture-provider (fixture-image)", 429, []byte(`{"error":{"message":"rate limited, slow down"}}`)),
			providerName: "Fixture Provider",
			modelName:    "Fixture Image",
			wantParts:    []string{fmt.Sprintf(ProviderServerError, "Fixture Provider", "Fixture Image", "rate limited, slow down")},
			wantAbsent:   []string{"provider response:", "status 429", `{"error"`},
		},
		{
			name:         "unusable response without an extractable message",
			err:          provider.APIErr("fixture-provider (fixture-video)", 500, []byte("<html>boom</html>")),
			providerName: "Fixture Provider",
			modelName:    "Fixture Video",
			wantParts:    []string{fmt.Sprintf(ResponseUnusable, "Fixture Provider")},
			wantAbsent:   []string{"provider response:", "status 500", "boom", formBody(test, ProviderServerError)},
		},
		{
			name:         "input image not found",
			err:          &errs.MediaError{Source: "/x/nope.png", Cause: errors.Join(errs.ErrInputMediaNotFound, errors.New("open /x/nope.png: no such file or directory"))},
			providerName: "Fixture Provider",
			wantParts:    []string{fmt.Sprintf(InputMediaNotFound, "/x/nope.png")},
			wantAbsent:   []string{"input media:", "no such file or directory"},
		},
		{
			name:         "unsupported input-media type",
			err:          &errs.MediaError{Source: "/x/fake.png", Problem: "image/gif", Cause: errs.ErrInputMediaMIME},
			providerName: "",
			wantParts:    []string{fmt.Sprintf(InputMediaFormat, "/x/fake.png")},
			wantAbsent:   []string{"input media:", "image/gif"},
		},
		{
			name:         "other input-media failure",
			err:          &errs.MediaError{Source: "/x/locked.png", Cause: errors.Join(errs.ErrInputMediaRead, errors.New("permission denied"))},
			providerName: "",
			wantParts:    []string{fmt.Sprintf(InputMediaSourceInvalid, "/x/locked.png")},
			wantAbsent:   []string{"input media:", "permission denied"},
		},
		{
			name:         "unknown model specifier",
			err:          &errs.ModelError{Specifier: "no-such-model", Cause: errs.ErrModelResolveUnknown},
			providerName: "",
			wantParts:    []string{fmt.Sprintf(ModelUnknown, "no-such-model")},
			wantAbsent:   []string{"model resolution:"},
		},
		{
			name:         "ambiguous specifier",
			err:          fmt.Errorf("%q, %w", "dup-model", errs.ErrModelResolveConflict),
			providerName: "",
			wantParts:    []string{ModelAmbiguous},
			wantAbsent:   []string{"model resolution:", "dup-model"},
		},
		{
			name:         "output-file failure",
			err:          &os.PathError{Op: "mkdir", Path: "/out/locked/sub", Err: errors.Join(errs.ErrOutputFileMkdir, errors.New("mkdir /out/locked/sub: permission denied"))},
			providerName: "",
			wantParts:    []string{"/out/locked/sub"},
			wantAbsent:   []string{"output file:", "permission denied"},
		},
		{
			name:         "broken provider config, decode",
			err:          &errs.ConfigError{Path: "fixture.json", Provider: "fixture", Cause: errors.Join(errs.ErrProvConfigDecode, errors.New("invalid character"))},
			providerName: "",
			wantParts:    []string{fmt.Sprintf(ProviderConfigBroken, "fixture"), "fixture.json"},
			wantAbsent:   []string{"provider config:", "invalid character"},
		},
		{
			name:         "broken provider config, invalid content",
			err:          &errs.ConfigError{Path: "fixture.json", Provider: "fixture", Problem: "no AdapterAPI section", Cause: errs.ErrProvConfigInvalid},
			providerName: "Fixture Labs",
			wantParts:    []string{fmt.Sprintf(ProviderConfigBroken, "fixture"), "no AdapterAPI section"},
			wantAbsent:   []string{"provider config:"},
		},
		{
			name:         "broken provider config, no constructor",
			err:          &errs.ConfigError{Provider: "fixture", Cause: errs.ErrProvConfigNoConstructor},
			providerName: "Fixture Provider",
			wantParts:    []string{"Fixture Provider"},
			wantAbsent:   []string{"provider config:"},
		},
		{
			name:         "working-directory failure",
			err:          fmt.Errorf("%q, %w, %w", "getwd: no such file or directory", errs.ErrProcessWorkingDir, errors.New("getwd: no such file or directory")),
			providerName: "",
			wantParts:    []string{WorkingDirFailed},
			wantAbsent:   []string{"CLI:", "getwd"},
		},
	}
}

// checkMessageMappingCase checks required and forbidden message fragments, valid UTF-8, and the
// absence of control characters. It also rejects the raw error chain and the unclassified fallback.
func checkMessageMappingCase(t *testing.T, mappingCase messageMappingCase) {
	t.Helper()

	got := errorMessage(mappingCase.err, mappingCase.providerName, mappingCase.modelName)
	if got == mappingCase.err.Error() {
		t.Errorf("✗ %s: the raw chain rendered verbatim: %q", mappingCase.name, got)
	}

	checkRenderedBytes(t, got, mappingCase.name)

	if got == errorMessage(errors.New("unclassified mapping test error"), mappingCase.providerName, mappingCase.modelName) {
		t.Errorf("✗ %s: classified error fell through to the unclassified notice", mappingCase.name)
	}

	for _, part := range mappingCase.wantParts {
		if !strings.Contains(got, part) {
			t.Errorf("✗ %s: line %q lacks %q", mappingCase.name, got, part)
		}
	}

	for _, absent := range mappingCase.wantAbsent {
		if strings.Contains(got, absent) {
			t.Errorf("✗ %s: line %q carries the internal form %q", mappingCase.name, got, absent)
		}
	}
}

// checkRenderedBytes rejects any raw control at the byte level: an invalid UTF-8 byte (how an
// eight-bit C1 control such as 0x9b arrives) or a decoded control rune anywhere in the rendered
// text.
func checkRenderedBytes(t *testing.T, rendered, label string) {
	t.Helper()

	for i := 0; i < len(rendered); {
		r, size := utf8.DecodeRuneInString(rendered[i:])
		if r == utf8.RuneError && size == 1 {
			t.Errorf("✗ %s carries a raw invalid byte 0x%02x: %q", label, rendered[i], rendered)

			return
		}

		if unicode.IsControl(r) {
			t.Errorf("✗ %s carries a control rune %q: %q", label, r, rendered)

			return
		}

		i += size
	}
}

// formVerbPattern matches one formatting verb of a catalog form, indexed or not.
func formVerbPattern(t testing.TB) *regexp.Regexp {
	t.Helper()

	return regexp.MustCompile(`%\[?[0-9]*\]?[-+# 0-9.]*[a-zA-Z]`)
}

// formBody removes formatting verbs from a message template and returns its longest literal
// segment, trimmed of surrounding whitespace.
func formBody(t testing.TB, form string) string {
	t.Helper()

	longest := ""

	for _, segment := range formVerbPattern(t).Split(form, -1) {
		if len(segment) > len(longest) {
			longest = segment
		}
	}

	return strings.TrimSpace(longest)
}
