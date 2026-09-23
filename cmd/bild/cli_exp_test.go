package main

// Invariants tested:
//  1. Usage failure reporting: Given zero or two info arguments, bild info must exit 2, leave
//     stdout empty, and include Usage: on stderr while omitting the full OPTIONS: section. Given no
//     prompt, an empty prompt, or an empty prompt followed by another argument, bild must exit 2
//     and leave stdout empty. Stderr must include Usage: and omit OPTIONS: and --aspect-ratio.
//     Given --bogus, -m without a value, strength abc, num-images abc, or duration 2.5, bild must
//     exit 2 and leave stdout empty. Stderr must name the rejected flag or value, include Usage:,
//     and omit OPTIONS: and --aspect-ratio.
//  2. Generation failure reporting: Given -m no-such-model-xyz and prompt p, bild must exit 1,
//     leave stdout empty, and name the model in stderr without Usage: or OPTIONS: text.
//  3. JSON errors: bild --json --bogus and bild list --json extra must exit 2; bild info --json
//     no-such-model-json must exit 1. Each must leave stderr empty and emit exactly one JSON object
//     with one error identifying the rejected flag, argument count, or model.
//  4. Command ownership: bild list extra and info with two arguments must fail with empty stdout.
//     list --version must exit 2 with empty stdout and name the rejected flag in stderr.
//  5. Provider configuration failures in `info`: After replacing the selected provider's
//     configuration with malformed JSON, running bild info for that provider through Command.Run
//     must return an error matching errs.ErrProvConfigDecode.
//  6. Continuation after prompt confirmation: Given terminal prompt hellp and reply not, bild
//     --json must print the confirmation question on stderr, continue with the unchanged prompt in
//     JSON stdout, include the missing-credential error, and exit 1.
//  7. Results file availability: Given a --save-results path beneath a regular file, bild must exit
//     1, leave stdout empty, and write a nonempty diagnostic to stderr.
//  8. Cancellation of a one-word prompt: Given terminal prompt hellp and a reply of n, N, no, No,
//     or No surrounded by spaces, bild must exit 0 and leave stdout empty. Stderr must contain
//     exactly the formatted confirmation question and its trailing space.
//  9. Early failures with `--print-filename`: With --json and --print-filename, bild must exit 2
//     with both streams empty for an unknown flag or missing prompt. Without JSON, the tested
//     unknown flag must produce empty stdout and usage on stderr.
//  10. Regular expression validation for `search`: Given ( as the --regex value, bild search must
//      exit 2, leave stdout empty, and include both Usage: and the invalid pattern on stderr.
//  11. Configuration warnings during `list`: Given invalid YAML in the user configuration, bild
//      list must exit 0 and include every built-in provider display name in stdout. Stderr must
//      name the invalid configuration file. Given api-keys.notaprovider in the user configuration,
//      bild list must exit 0 and name both notaprovider and the configuration file in stderr.
//  12. Uniform usage paths: bild --bogus, bild with no arguments, and bild list --bogus must exit
//      2, leave stdout empty, and include Usage: without OPTIONS: on stderr. Invalid-flag errors
//      must name bogus and omit the relevant root or command flag listing.
//  13. Ambiguous model information: Given two providers declaring shared-id, running bild info
//      shared-id through Command.Run must return errs.ErrModelResolveConflict.
//  14. JSON combined with `--print-filename`: Given --json, --print-filename, and unknown model
//      no-such-model-jp without --save-results, bild must exit 1 and leave both stdout and stderr
//      empty.
//  15. Help command error identity: Running bild help --nothing through Command.Run must return an
//      error matching errs.ErrCLI.
//  16. Root argument count: Given two positional arguments, bild must exit 2, leave stdout empty,
//      and write nonempty stderr.
//  17. Whitespace-only prompt: Given a whitespace-only prompt, bild must exit 2, leave stdout
//      empty, and print the same stderr as a no-argument invocation.
//  18. Help and version flags under hostile input: bild must accept the listed standalone
//      help/version combinations and reject the listed invalid combinations with text or JSON usage
//      output. If -h is the value of --model, it must exit 1 without usage.
//  19. Application loaded once: With malformed configuration, bild list must name the configuration
//      path once in stderr. Standalone version and a rejected command line must not name it.
//  20. Search terms under hostile input: bild search -x -m must exit 0 with empty stderr and print
//      an indented listing. Repeated -x values ending in openrouter must produce the same nonempty
//      output as -x openrouter alone. search -- -x must exit 0 with both streams empty. Empty
//      terms, missing exclusion values, surplus terms, and exclusion combined with help must exit 2
//      with empty stdout, Usage:, and the specified error text. The two invalid JSON searches must
//      exit 2 with empty stderr and one JSON object containing one error.
//  21. Essential command delivery: With read-only stdout, every tested list, search, info, help,
//      and version form must return errs.ErrOutputFileWrite and exit 1. Writable /dev/null must
//      allow success.
//  22. Early result ownership: Given a registration mismatch, bild must write the failed JSON
//      outcome to the requested file and close it before returning, including in filename mode.
//      Version must still work, and restoring the catalog must allow a later JSON listing to
//      succeed.
//  23. Help topic arity: bild help with no topic or one of list, info, or search must exit 0 with
//      nonempty stdout and empty stderr. bild help list extra must exit 2 with empty stdout and
//      nonempty stderr.
//  24. Explicit command input: Given a supplied strings.Reader despite terminal process input and
//      diagnostics, Command.Run must produce the failed JSON outcome without a confirmation
//      diagnostic.
//  25. Help and version flags alone: For the tested help or version flags combined with positional
//      arguments, bild must exit 2 with empty stdout and print compact usage plus the message
//      naming --help or --version. When combined with another flag or repeated, it must print usage
//      without that message. Invalid combinations with JSON must emit one JSON object containing
//      one error and leave stderr empty. Standalone supported forms must exit 0, and -h -- must
//      print the general help page without diagnostics. Root flags before command words must exit 2
//      and identify the command in the flags-before-command error, using text or JSON as selected.
//      When -h is a flag value or follows -- as a prompt, the run must exit 1 without the
//      combination message.
//  26. Credential configuration guidance: With the selected model's credentials absent, bild must
//      exit 1 in text, JSON, and debug modes. The combined output must name the credential
//      environment variable, api-keys configuration key, and ~/.bildomat/config.yml.
//  27. Short generation flags: Given -i with a missing file, bild must exit 1 and print the
//      formatted missing-input message. Given --json -n p, it must exit 1 after the blocked request
//      and retain p as the prompt. Given -N 0 for a supporting model, it must exit 1 and print the
//      adjustment from 0 to 1. Given -N 2.5, it must exit 2.
//  28. Generation argument parsing: bild must report the expected duration adjustment and exit 1
//      with missing credentials. In the JSON form, -m after -- must remain the prompt.
//  29. Unknown help topic: bild help nonexistent must exit 2, leave stdout empty, and name
//      nonexistent in stderr.
//  30. Independent version requests: Given malformed user configuration, bild --version must
//      succeed with the exact version line and empty stderr. Combining version with a prompt, JSON,
//      or list must exit 2.
//  31. Help page failure: Given malformed root and command help templates, bild -h and bild list -h
//      must each exit 1, leave stdout empty, and write nonempty stderr.
//  32. CLI malformed values: parseCfg must classify unknown Unicode flags, malformed booleans, and
//      numeric overflows as errs.ErrCLIFlagParse without invoking capture.
//  33. CLI output directory flag removed: parseCfg must reject --output-dir and -O with
//      errs.ErrCLIFlagParse without invoking capture.
//  34. Prompt trimming: parseCfg must trim the prompt and reject a whitespace-only prompt with
//      errs.ErrCLIPromptMissing without invoking capture.
//  35. High-volume CLI arguments: Given 100 input-media flags and a 10,000-character prompt,
//      parseCfg must succeed and return 100 sources with the exact prompt.
//  36. CLI error identities: parseCfg must retain the missing-prompt and flag-parse sentinels and
//      avoid invoking capture for those failures.
//  37. Experimental options: bild --help must omit debug, persistence, and reuse options. The
//      tested malformed, unknown, repeated, and unsupported reuse selections must exit 2.
//  38. Classified command dispatch: For list and info with fuzzed tokens, Command.Run must not
//      panic; any error must match a tested command classification or implement cli.ExitCoder.
//  39. Command line under arbitrary help, version, and command-word arguments: For up to four
//      tokens selected from the help/version vocabulary, bild must not panic and must return status
//      0, 1, or 2. Status 2 must produce either empty stdout with Usage: on stderr, or one valid
//      JSON object on stdout with empty stderr. The vocabulary excludes --print-filename.
//  40. Search command under arbitrary terms and flags: For bild search followed by up to four
//      vocabulary tokens, including empty strings, bild must not panic and must return status 0, 1,
//      or 2. Status 0 must leave stderr empty. Status 2 must produce either empty stdout with
//      Usage: on stderr, or one valid JSON object on stdout with empty stderr.
//  41. CLI parsing under arbitrary arguments: For three arbitrary argument strings, testCommand
//      must not panic. If capture runs, Run must succeed with a nonempty prompt; otherwise any
//      error must match a flag-parse, missing-prompt, or argument-count sentinel.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
	providerconfig "github.com/shdeen/bildomat/internal/provider/config"
	tmpl "github.com/shdeen/bildomat/internal/templates"
	"github.com/urfave/cli/v3"
)

// TestCLIInfoArgumentCount verifies invariant #1: Usage failure reporting.
//
// What is being tested:
// Given zero or two info arguments, bild info must exit 2, leave stdout empty, and include Usage:
// on stderr while omitting the full OPTIONS: section.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIInfoArgumentCount(t *testing.T) {
	clearProviderKeys(t)

	shippedProvider := firstStandardProvider(t, shippedCatalog(t))
	if len(shippedProvider.Models) == 0 {
		t.Fatalf("💣 the provider %s declares no model to name as a second argument", shippedProvider.ID)
	}

	for _, args := range [][]string{{"info"}, {"info", shippedProvider.ID, shippedProvider.Models[0].ID}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 {
			t.Errorf("✗ %v: exit %d, want 2", args, code)
		}

		if stdout != "" {
			t.Errorf("✗ %v: stdout = %q, want empty", args, stdout)
		}

		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: compact usage did not render: %q", args, stderr)
		}

		if strings.Contains(stderr, "OPTIONS:") {
			t.Errorf("✗ %v: the full command help page rendered: %q", args, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ info reports a usage error for any argument count but one")
	}
}

// TestCLIPromptMissing verifies invariant #1: Usage failure reporting.
//
// What is being tested:
// Given no prompt, an empty prompt, or an empty prompt followed by another argument, bild must exit
// 2 and leave stdout empty. Stderr must include Usage: and omit OPTIONS: and --aspect-ratio.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIPromptMissing(t *testing.T) {
	cases := [][]string{{}, {""}, {"", "out"}}
	for _, args := range cases {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 {
			t.Errorf("✗ %v: exit %d, want 2", args, code)
		}

		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: compact usage did not render: %q", args, stderr)
		}

		if strings.Contains(stderr, "OPTIONS:") || strings.Contains(stderr, "--aspect-ratio") {
			t.Errorf("✗ %v: the full help page rendered: %q", args, stderr)
		}

		if stdout != "" {
			t.Errorf("✗ %v: stdout = %q, want empty", args, stdout)
		}
	}

	if !t.Failed() {
		t.Log("✓ missing and empty prompts print compact usage on stderr and exit 2")
	}
}

// TestCLIParseError verifies invariant #1: Usage failure reporting.
//
// What is being tested:
// Given --bogus, -m without a value, strength abc, num-images abc, or duration 2.5, bild must exit
// 2 and leave stdout empty. Stderr must name the rejected flag or value, include Usage:, and omit
// OPTIONS: and --aspect-ratio.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIParseError(t *testing.T) {
	cases := []struct {
		args   []string
		needle string
	}{
		{[]string{"--bogus"}, "bogus"},
		{[]string{"-m"}, "-m"},
		{[]string{"--strength", "abc"}, "abc"},
		{[]string{"-N", "abc"}, "abc"},
		{[]string{"-d", "2.5"}, "2.5"},
	}
	for _, c := range cases {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 {
			t.Errorf("✗ %v: exit %d, want 2", c.args, code)
		}

		if !strings.Contains(stderr, c.needle) {
			t.Errorf("✗ %v: stderr does not name %q: %q", c.args, c.needle, stderr)
		}

		if !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: compact usage did not render: %q", c.args, stderr)
		}

		if strings.Contains(stderr, "OPTIONS:") || strings.Contains(stderr, "--aspect-ratio") {
			t.Errorf("✗ %v: the full help page rendered: %q", c.args, stderr)
		}

		if stdout != "" {
			t.Errorf("✗ %v: stdout = %q, want empty", c.args, stdout)
		}
	}

	if !t.Failed() {
		t.Log("✓ parse errors print an error line plus usage on stderr and exit 2")
	}
}

// TestCLIGenerateError verifies invariant #2: Generation failure reporting.
//
// What is being tested:
// Given -m no-such-model-xyz and prompt p, bild must exit 1, leave stdout empty, and name the model
// in stderr without Usage: or OPTIONS: text.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIGenerateError(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "-m", "no-such-model-xyz", "p")
	if code != 1 {
		t.Errorf("✗ exit %d, want 1", code)
	}

	if !strings.Contains(stderr, "no-such-model-xyz") {
		t.Errorf("✗ stderr does not name the model input: %q", stderr)
	}

	if strings.Contains(stderr, "Usage:") || strings.Contains(stderr, "OPTIONS:") {
		t.Errorf("✗ stderr unexpectedly carries usage or help text")
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if !t.Failed() {
		t.Log("✓ a generation error prints its user message on stderr and exits 1")
	}
}

// TestCLIJSONErrors verifies invariant #3: JSON errors.
//
// What is being tested:
// bild --json --bogus and bild list --json extra must exit 2; bild info --json no-such-model-json
// must exit 1. Each must leave stderr empty and emit exactly one JSON object with one error
// identifying the rejected flag, argument count, or model.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIJSONErrors(t *testing.T) {
	clearProviderKeys(t)

	for _, testCase := range []struct {
		name     string
		args     []string
		wantCode int
		wantText string
	}{
		{name: "root parse", args: []string{"--json", "--bogus"}, wantCode: 2, wantText: "bogus"},
		{name: "list arguments", args: []string{"list", "--json", "extra"}, wantCode: 2, wantText: "argument"},
		{name: "info lookup", args: []string{"info", "--json", "no-such-model-json"}, wantCode: 1, wantText: "no-such-model-json"},
	} {
		code, stdout, stderr := captureCLI(t, testCase.args...)
		if code != testCase.wantCode {
			t.Errorf("✗ %s: exit %d, want %d", testCase.name, code, testCase.wantCode)
		}

		if stderr != "" {
			t.Errorf("✗ %s: stderr = %q, want empty", testCase.name, stderr)
		}

		document := decodeJSONObject(t, stdout)

		errors := jsonArrayField(t, document, "errors")
		if len(errors) != 1 || !strings.Contains(fmt.Sprint(errors[0]), testCase.wantText) {
			t.Errorf("✗ %s: errors = %#v, want text %q", testCase.name, errors, testCase.wantText)
		}
	}

	if !t.Failed() {
		t.Log("✓ JSON errors stay on stdout in one document while usage and operational exit statuses remain unchanged")
	}
}

// TestCLIReservedUnquoted verifies invariant #4: Command ownership.
//
// What is being tested:
// bild list extra and bild info followed by a provider ID and extra must each return a nonzero
// status and leave stdout empty.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIReservedUnquoted(t *testing.T) {
	clearProviderKeys(t)

	shippedProvider := firstStandardProvider(t, shippedCatalog(t))

	for _, args := range [][]string{
		{"list", "extra"},
		{"info", shippedProvider.ID, "extra"},
	} {
		code, stdout, _ := captureCLI(t, args...)
		if code == 0 {
			t.Errorf("✗ %v: exit 0, want a failure (unexpected arguments)", args)
		}

		if stdout != "" {
			t.Errorf("✗ %v: stdout = %q, want empty", args, stdout)
		}
	}

	if !t.Failed() {
		t.Log("✓ an unquoted leading reserved word dispatches with the rest as that command's arguments")
	}
}

// TestCLIVersionOnCommand verifies invariant #4: Command ownership.
//
// What is being tested:
// bild list --version must exit 2, leave stdout empty, and name version in stderr without printing
// the provider listing.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIVersionOnCommand(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "list", "--version")

	if code != 2 {
		t.Errorf("✗ exit %d, want 2 (the version flag is the root's alone)", code)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "version") {
		t.Errorf("✗ stderr does not name the rejected flag: %q", stderr)
	}

	if strings.Contains(stdout, shippedCatalog(t).Providers[0].DisplayName) {
		t.Errorf("✗ the listing ran despite --version")
	}

	if !t.Failed() {
		t.Log("✓ the version flag belongs to the root command alone; a subcommand rejects it")
	}
}

// TestCLIInfoBrokenConfig verifies invariant #5: Provider configuration failures in `info`.
//
// What is being tested:
// After replacing the selected provider's configuration with malformed JSON, running bild info for
// that provider through Command.Run must return an error matching errs.ErrProvConfigDecode.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIInfoBrokenConfig(t *testing.T) {
	flags := params.Flags()

	sources := providerRegistrations()
	if len(sources) == 0 {
		t.Fatalf("💣 no registered provider config exists to corrupt")
	}

	sources[0].ConfigBytes = []byte(`{ this is deliberately not JSON`)
	brokenProviderID := sources[0].ProviderID

	loadedCatalog, err := catalog.LoadCatalog(flags, registeredTestSources(t, sources)...)
	if err != nil {
		t.Fatalf("💣 catalog construction failed: %v", err)
	}

	app := fixtureApp(t, loadedCatalog)
	command := quietCommand(app)

	err = command.Run(context.Background(), []string{"bild", "info", brokenProviderID})
	if !errors.Is(err, errs.ErrProvConfigDecode) {
		t.Errorf("✗ info %s on the broken build = %v, want the stored decode error", brokenProviderID, err)
	}

	if !t.Failed() {
		t.Log("✓ info surfaces a broken config's stored decode error rather than falling through")
	}
}

// TestCLISingleWordPromptContinuation verifies invariant #6: Continuation after prompt
// confirmation.
//
// What is being tested:
// Given terminal prompt hellp and reply not, bild --json must print the confirmation question on
// stderr, continue with the unchanged prompt in JSON stdout, include the missing-credential error,
// and exit 1.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLISingleWordPromptContinuation(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureInteractiveCLI(t, "not\n", true, "--json", "--model", qualifiedSpecifier(t, builtinDefaultPair(t, shippedCatalog(t))), "hellp")
	if code != 1 {
		t.Errorf("✗ reply not: exit %d, want 1 at the missing-credential boundary", code)
	}

	if !strings.Contains(stderr, formBody(t, output.SingleWordConfirmation)) {
		t.Errorf("✗ reply not: confirmation is missing: %q", stderr)
	}

	if !strings.Contains(stdout, `"prompt": "hellp"`) {
		t.Errorf("✗ reply not: generation did not continue with the original prompt: %q", stdout)
	}

	if !strings.Contains(stdout, errs.ErrKeyMissing.Error()) {
		t.Errorf("✗ reply not: generation did not reach the credential boundary: %q", stdout)
	}

	if !t.Failed() {
		t.Log("✓ only a complete n or no reply cancels; another reply continues the original request")
	}
}

// TestCLISaveResultsUnopenable verifies invariant #7: Results file availability.
//
// What is being tested:
// Given a --save-results path beneath a regular file, bild must exit 1, leave stdout empty, and
// write a nonempty diagnostic to stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLISaveResultsUnopenable(t *testing.T) {
	blockerPath := filepath.Join(t.TempDir(), "blocker")
	if writeErr := os.WriteFile(blockerPath, []byte("x"), 0o600); writeErr != nil {
		t.Fatalf("💣 the blocker file could not be written: %v", writeErr)
	}

	code, stdout, stderr := captureCLI(t, "--save-results", filepath.Join(blockerPath, "results.txt"), "two words")
	if code != 1 {
		t.Errorf("✗ exit = %d, want 1 for the unopenable results path", code)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if stderr == "" {
		t.Errorf("✗ stderr is empty, want the results-path failure")
	}

	if !t.Failed() {
		t.Log("✓ an unopenable results path fails the run before generation")
	}
}

// TestCLISingleWordPromptConfirmation verifies invariant #8: Cancellation of a one-word prompt.
//
// What is being tested:
// Given terminal prompt hellp and a reply of n, N, no, No, or No surrounded by spaces, bild must
// exit 0 and leave stdout empty. Stderr must contain exactly the formatted confirmation question
// and its trailing space.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLISingleWordPromptConfirmation(t *testing.T) {
	clearProviderKeys(t)

	for _, reply := range []string{"n\n", "N\n", "no\n", "No\n", "  No  \n"} {
		code, stdout, stderr := captureInteractiveCLI(t, reply, true, "hellp")
		if code != 0 {
			t.Errorf("✗ reply %q: exit %d, want 0", reply, code)
		}

		if stdout != "" {
			t.Errorf("✗ reply %q: stdout = %q, want empty", reply, stdout)
		}

		wantPrompt := fmt.Sprintf(output.SingleWordConfirmation, "", "", "", "hellp") + " "
		if stderr != wantPrompt {
			t.Errorf("✗ reply %q: stderr = %q, want %q", reply, stderr, wantPrompt)
		}
	}

	if !t.Failed() {
		t.Log("✓ an interactive single-word prompt is confirmed, and n, N, no, or No cancels with no generation output")
	}
}

// TestCLIPrintFilenameEarlyStops verifies invariant #9: Early failures with `--print-filename`.
//
// What is being tested:
// With --json and --print-filename, bild must exit 2 and leave both streams empty for --bogus or a
// missing prompt. With --print-filename and --bogus alone, it must exit 2, leave stdout empty, and
// include Usage: on stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIPrintFilenameEarlyStops(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "--json", "--print-filename", "--bogus")
	if code != 2 || stdout != "" || stderr != "" {
		t.Errorf("✗ JSON usage error: exit %d stdout %q stderr %q, want 2 with both streams empty", code, stdout, stderr)
	}

	code, stdout, stderr = captureCLI(t, "--json", "--print-filename")
	if code != 2 || stdout != "" || stderr != "" {
		t.Errorf("✗ JSON missing prompt: exit %d stdout %q stderr %q, want 2 with both streams empty", code, stdout, stderr)
	}

	code, stdout, stderr = captureCLI(t, "--print-filename", "--bogus")
	if code != 2 || stdout != "" {
		t.Errorf("✗ plain usage error: exit %d stdout %q, want 2 with empty stdout", code, stdout)
	}

	if !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ plain usage error lost its compact usage: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ early stops under --print-filename keep stdout empty")
	}
}

// TestCLISearchRegexpError verifies invariant #10: Regular expression validation for `search`.
//
// What is being tested:
// Given ( as the --regex value, bild search must exit 2, leave stdout empty, and include both
// Usage: and the invalid pattern on stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLISearchRegexpError(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "search", "--regex", "(")
	if code != 2 {
		t.Errorf("✗ exit %d, want 2", code)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, "(") {
		t.Errorf("✗ stderr lacks the compact usage or the pattern: %q", stderr)
	}

	if !t.Failed() {
		t.Log("✓ a broken pattern is a usage error")
	}
}

// TestListWarnsOnBrokenConfig verifies invariant #11: Configuration warnings during `list`.
//
// What is being tested:
// Given invalid YAML in the user configuration, bild list must exit 0 and include every built-in
// provider display name in stdout. Stderr must name the invalid configuration file.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestListWarnsOnBrokenConfig(t *testing.T) {
	configPath := setUserConfig(t, "default-model: [unclosed\n")

	code, stdout, stderr := captureCLI(t, "list")
	if code != 0 {
		t.Errorf("✗ exit = %d, want 0 (stderr: %q)", code, stderr)
	}

	for _, document := range shippedConfigDocuments(t) {
		displayName, _ := document["displayName"].(string)
		if !strings.Contains(stdout, displayName) {
			t.Errorf("✗ the listing lacks provider %q", displayName)
		}
	}

	if !strings.Contains(stderr, configPath) {
		t.Errorf("✗ stderr %q carries no warning naming the config file %q", stderr, configPath)
	}

	if !t.Failed() {
		t.Log("✓ a broken config file warns on stderr while the listing renders and the run exits 0")
	}
}

// TestListWarnsOnUnknownProviderKey verifies invariant #11: Configuration warnings during `list`.
//
// What is being tested:
// Given api-keys.notaprovider in the user configuration, bild list must exit 0 and name both
// notaprovider and the configuration file in stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestListWarnsOnUnknownProviderKey(t *testing.T) {
	configPath := setUserConfig(t, "api-keys:\n  notaprovider: bogus-value\n")

	code, _, stderr := captureCLI(t, "list")
	if code != 0 {
		t.Errorf("✗ exit = %d, want 0 (stderr: %q)", code, stderr)
	}

	for _, wantValue := range []string{"notaprovider", configPath} {
		if !strings.Contains(stderr, wantValue) {
			t.Errorf("✗ stderr %q carries no warning naming %q", stderr, wantValue)
		}
	}

	if !t.Failed() {
		t.Log("✓ an api-keys entry naming no provider warns on stderr while the listing exits 0")
	}
}

// TestCLIUsagePathsNew verifies invariant #12: Uniform usage paths.
//
// What is being tested:
// bild --bogus, bild with no arguments, and bild list --bogus must exit 2, leave stdout empty, and
// include Usage: without OPTIONS: on stderr. Invalid-flag errors must name bogus and omit the
// relevant root or command flag listing.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIUsagePathsNew(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "--bogus")
	if code != 2 {
		t.Errorf("✗ --bogus: exit %d, want 2", code)
	}

	if !strings.Contains(stderr, "bogus") || !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ --bogus: stderr lacks the error line or compact usage")
	}

	if strings.Contains(stderr, "OPTIONS:") || strings.Contains(stderr, "--aspect-ratio") {
		t.Errorf("✗ --bogus: stderr contains the full help page")
	}

	if stdout != "" {
		t.Errorf("✗ --bogus: stdout = %q, want empty", stdout)
	}

	code, stdout, stderr = captureCLI(t)
	if code != 2 {
		t.Errorf("✗ no args: exit %d, want 2", code)
	}

	if !strings.Contains(stderr, "Usage:") || strings.Contains(stderr, "OPTIONS:") {
		t.Errorf("✗ no args: stderr is not compact usage: %q", stderr)
	}

	if stdout != "" {
		t.Errorf("✗ no args: stdout = %q, want empty", stdout)
	}

	code, stdout, stderr = captureCLI(t, "list", "--bogus")
	if code != 2 {
		t.Errorf("✗ list --bogus: exit %d, want 2", code)
	}

	if !strings.Contains(stderr, "bogus") || !strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ list --bogus: stderr lacks the error line or compact usage")
	}

	if strings.Contains(stderr, "OPTIONS:") || strings.Contains(stderr, "--image") {
		t.Errorf("✗ list --bogus: stderr contains the full command help page")
	}

	if stdout != "" {
		t.Errorf("✗ list --bogus: stdout = %q, want empty", stdout)
	}

	if !t.Failed() {
		t.Log("✓ usage failures return exit 2 and render compact usage only on stderr")
	}
}

// TestCLIInfoAmbiguous verifies invariant #13: Ambiguous model information.
//
// What is being tested:
// Given two providers declaring shared-id, running bild info shared-id through Command.Run must
// return errs.ErrModelResolveConflict.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIInfoAmbiguous(t *testing.T) {
	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags,
		catalog.Source{ProviderID: "prov-a", ConfigBytes: []byte(fixtureProviderCfg(t, "prov-a", "shared-id"))},
		catalog.Source{ProviderID: "prov-b", ConfigBytes: []byte(fixtureProviderCfg(t, "prov-b", "shared-id"))},
	)
	if err != nil {
		t.Fatalf("💣 catalog construction failed: %v", err)
	}

	app := fixtureApp(t, loadedCatalog)
	command := quietCommand(app)

	err = command.Run(context.Background(), []string{"bild", "info", "shared-id"})
	if !errors.Is(err, errs.ErrModelResolveConflict) {
		t.Errorf("✗ info shared-id = %v, want the ambiguity error", err)
	}

	if !t.Failed() {
		t.Log("✓ info returns the ambiguity error for a shared bare id")
	}
}

// TestCLIJSONWithPrintFilename verifies invariant #14: JSON combined with `--print-filename`.
//
// What is being tested:
// Given --json, --print-filename, and unknown model no-such-model-jp without --save-results, bild
// must exit 1 and leave both stdout and stderr empty.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIJSONWithPrintFilename(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "--json", "--print-filename", "--model", "no-such-model-jp", "two words")
	if code != 1 {
		t.Errorf("✗ exit = %d, want 1 for the operational failure", code)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty: the JSON document has no destination", stdout)
	}

	if stderr != "" {
		t.Errorf("✗ stderr = %q, want empty under --json", stderr)
	}

	if !t.Failed() {
		t.Log("✓ --json with --print-filename and no results path emits nothing")
	}
}

// TestHelpUsageErrorClassified verifies invariant #15: Help command error identity.
//
// What is being tested:
// Running bild help --nothing through Command.Run must return an error matching errs.ErrCLI.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestHelpUsageErrorClassified(t *testing.T) {
	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, registeredTestSources(t, providerRegistrations())...)
	if err != nil {
		t.Fatalf("💣 catalog construction failed: %v", err)
	}

	app := fixtureApp(t, loadedCatalog)
	command := quietCommand(app)

	runErr := command.Run(context.Background(), []string{"bild", "help", "--nothing"})
	if !errors.Is(runErr, errs.ErrCLI) {
		t.Errorf("✗ help --nothing = %v, want the usage-error classification", runErr)
	}

	if !t.Failed() {
		t.Log("✓ an invalid flag on the help subcommand classifies as a usage error")
	}
}

// TestRootArgumentCount verifies invariant #16: Root argument count.
//
// What is being tested:
// Given two positional arguments, hello world and second argument, bild must exit 2, leave stdout
// empty, and write nonempty stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestRootArgumentCount(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "hello world", "second argument")
	if code != 2 {
		t.Errorf("✗ exit = %d, want 2 (stderr: %s)", code, stderr)
	}

	if stdout != "" {
		t.Errorf("✗ stdout = %q, want empty", stdout)
	}

	if stderr == "" {
		t.Errorf("✗ stderr is empty, want the usage error")
	}

	if !t.Failed() {
		t.Log("✓ a second positional argument is a usage error")
	}
}

// TestWhitespaceOnlyPrompt verifies invariant #17: Whitespace-only prompt.
//
// What is being tested:
// Given a prompt containing only spaces and a tab, bild must exit 2 and leave stdout empty. Its
// stderr must exactly match the no-argument invocation, which must also exit 2.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestWhitespaceOnlyPrompt(t *testing.T) {
	clearProviderKeys(t)

	codeBlank, stdoutBlank, stderrBlank := captureCLI(t, "  \t ")
	codeMissing, _, stderrMissing := captureCLI(t)

	if codeBlank != 2 || stdoutBlank != "" {
		t.Errorf("✗ a whitespace prompt: exit %d, stdout %q; want exit 2 and empty stdout", codeBlank, stdoutBlank)
	}

	if codeMissing != 2 || stderrBlank != stderrMissing {
		t.Errorf("✗ a whitespace prompt renders:\n%s\nwant the missing-prompt rendering:\n%s", stderrBlank, stderrMissing)
	}

	if !t.Failed() {
		t.Log("✓ a whitespace-only prompt is the missing prompt")
	}
}

// TestHelpVersionFlagsHostile verifies invariant #18: Help and version flags under hostile input.
//
// What is being tested:
// For the accepted help, version, terminator, and debug combinations, bild must exit 0 with empty
// stderr. The listed invalid combinations must exit 2 with empty stdout and Usage: on stderr, or
// with empty stderr and one JSON object containing one error when JSON was parsed. Given --model -h
// and prompt a cat, it must instead exit 1 without Usage: on stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestHelpVersionFlagsHostile(t *testing.T) {
	clearProviderKeys(t)

	for _, args := range [][]string{{"-v", "--"}, {"--help", "--"}, {"list", "-h", "--"}, {"help", "--debug"}, {"list", "--debug", "-m"}, {"search", "--help"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 0 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q; want 0 with an empty stderr", args, code, stderr)
		}
	}

	for _, args := range [][]string{
		{"-h", "--", "list"},
		{"-h", "help"},
		{"help", "-h"},
		{"help", "list", "-h"},
		{"info", "-h", "google"},
		{"list", "--help", "--help"},
		{"list", "-h", "extra"},
		{"--debug", "-h"},
		{"--debug", "list"},
		{"-p", "list"},
		{"--aspect-ratio", "1:1", "info", "google"},
		{"-v", "help"},
	} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: exit %d, stdout %q, stderr %q; want the usage error 2 with compact usage and an empty stdout", args, code, stdout, stderr)
		}
	}

	for _, args := range [][]string{{"-j", "-v"}, {"info", "google", "-j", "-h"}, {"-j", "help"}, {"search", "-j", "-h", "veo"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q; want the usage error 2 with an empty stderr", args, code, stderr)
		}

		if documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors"); len(documentErrors) != 1 {
			t.Errorf("✗ %v: errors = %#v, want one error in the JSON document", args, documentErrors)
		}
	}

	code, _, stderr := captureCLI(t, "--model", "-h", "a cat")
	if code != 1 || strings.Contains(stderr, "Usage:") {
		t.Errorf("✗ [--model -h a cat]: exit %d, stderr %q; want 1, the unknown model, with no usage error", code, stderr)
	}

	if !t.Failed() {
		t.Log("✓ hostile help and version combinations serve the flag's page or fail as usage errors")
	}
}

// TestApplicationLoadedOnce verifies invariant #19: Application loaded once.
//
// What is being tested:
// Given malformed user configuration, bild list -m must exit 0 and name the configuration path
// exactly once in stderr. bild -v must exit 0 without naming that path, and bild --bogus must exit
// 2 without naming it.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestApplicationLoadedOnce(t *testing.T) {
	clearProviderKeys(t)

	configPath := setUserConfig(t, "default-model: [unclosed\n")

	for _, args := range [][]string{{"list", "-m"}, {"-v"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 0 {
			t.Errorf("✗ %v: exit %d, want 0 (stderr: %q)", args, code, stderr)
		}

		requiredWarnings := 1
		if args[0] == "-v" {
			requiredWarnings = 0
		}

		if warnings := strings.Count(stderr, configPath); warnings != requiredWarnings {
			t.Errorf("✗ %v: stderr names the config file %d times, want %d: %q", args, warnings, requiredWarnings, stderr)
		}
	}

	code, _, stderr := captureCLI(t, "--bogus")
	if code != 2 || strings.Contains(stderr, configPath) {
		t.Errorf("✗ [--bogus]: exit %d, stderr %q; want 2 with no user config warning", code, stderr)
	}

	if !t.Failed() {
		t.Log("✓ a run reads the user config once, and a rejected command line reads none")
	}
}

// TestSearchTermsHostile verifies invariant #20: Search terms under hostile input.
//
// What is being tested:
// bild search -x -m must exit 0 with empty stderr and print an indented listing. Repeated -x values
// ending in openrouter must produce the same nonempty output as -x openrouter alone. search -- -x
// must exit 0 with both streams empty. Empty terms, missing exclusion values, surplus terms, and
// exclusion combined with help must exit 2 with empty stdout, Usage:, and the specified error text.
// The two invalid JSON searches must exit 2 with empty stderr and one JSON object containing one
// error.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestSearchTermsHostile(t *testing.T) {
	clearProviderKeys(t)

	code, stdout, stderr := captureCLI(t, "search", "-x", "-m")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "\n  ") {
		t.Errorf("✗ [search -x -m]: exit %d, stderr %q; want 0 and the nested listing", code, stderr)
	}

	_, repeatedExclusion, _ := captureCLI(t, "search", "-m", "-x", "zzz-matches-no-model", "-x", "openrouter")
	_, lastExclusionAlone, _ := captureCLI(t, "search", "-m", "-x", "openrouter")

	if repeatedExclusion != lastExclusionAlone || lastExclusionAlone == "" {
		t.Errorf("✗ a repeated -x does not keep the last value: the two listings differ or are empty")
	}

	code, stdout, stderr = captureCLI(t, "search", "--", "-x")
	if code != 0 || stdout != "" || stderr != "" {
		t.Errorf("✗ [search -- -x]: exit %d, stdout %q, stderr %q; want 0 and nothing", code, stdout, stderr)
	}

	for _, usageCase := range []struct {
		args     []string
		wantText string
	}{
		{[]string{"search", "", "-x", "openrouter"}, SearchTermEmpty},
		{[]string{"search", "openai", "-x", ""}, SearchTermEmpty},
		{[]string{"search", "--exclude="}, SearchTermEmpty},
		{[]string{"search", "flux", "dream", "-x", "openrouter"}, "2"},
		{[]string{"search", "-x", "openrouter", "-h"}, ""},
		{[]string{"search", "-x"}, ""},
	} {
		code, stdout, stderr := captureCLI(t, usageCase.args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, usageCase.wantText) {
			t.Errorf("✗ %q: exit %d, stdout %q, stderr %q; want the usage error 2 carrying %q", usageCase.args, code, stdout, stderr, usageCase.wantText)
		}
	}

	for _, args := range [][]string{{"search", "-j"}, {"search", "-j", "flux", "dream"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stderr != "" {
			t.Errorf("✗ %q: exit %d, stderr %q; want 2 and empty", args, code, stderr)
		}

		if documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors"); len(documentErrors) != 1 {
			t.Errorf("✗ %q: errors = %#v, want one error in the JSON document", args, documentErrors)
		}
	}

	if !t.Failed() {
		t.Log("✓ hostile search terms are taken as values or refused as usage errors")
	}
}

// TestEssentialCommandOutput verifies invariant #21: Essential command delivery.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// With stdout opened read-only, Command.Run for each tested list, search, info, help, or version
// form must return errs.ErrOutputFileWrite and map to exit 1. With writable /dev/null stdout, every
// tested command must return nil.
func TestEssentialCommandOutput(t *testing.T) {
	commands := [][]string{{"list"}, {"search", "google"}, {"info", "google"}, {"info", "google/veo-3.1-fast-generate-preview"}, {"help"}, {"help", "list"}, {"--help"}, {"list", "--help"}, {"--version"}, {"list", "--json"}}
	for _, arguments := range commands {
		t.Run(strings.Join(arguments, " "), func(t *testing.T) {
			for _, writable := range []bool{false, true} {
				openFlags := os.O_RDONLY
				if writable {
					openFlags = os.O_WRONLY
				}

				stream, err := os.OpenFile(os.DevNull, openFlags, 0)
				if err != nil {
					t.Fatalf("💣 open standard output fixture: %v", err)
				}

				originalOutput := os.Stdout
				os.Stdout = stream
				command := createCommand(&bildApp{})
				runErr := command.Run(context.Background(), append([]string{"bild"}, arguments...))
				os.Stdout = originalOutput

				if closeErr := stream.Close(); closeErr != nil {
					t.Errorf("✗ close fixture: %v", closeErr)
				}

				if writable {
					if runErr != nil {
						t.Errorf("✗ writable output failed: %v", runErr)
					}
				} else if runExitCode(runErr) != 1 || !errors.Is(runErr, errs.ErrOutputFileWrite) {
					t.Errorf("✗ read-only stdout returned status %d and %v", runExitCode(runErr), runErr)
				}
			}

			if !t.Failed() {
				t.Log("✓ command status reflects essential output delivery")
			}
		})
	}

	if !t.Failed() {
		t.Log("✓ every essential command output propagates write failures")
	}
}

// TestCatalogFailureResultDestination verifies invariant #22: Early result ownership.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given a provider ID that conflicts with its registration, bild --json --save-results must exit 1,
// leave both streams empty, and write one failed document with one error to the requested file. The
// same must hold with --print-filename, and no descriptor may still hold either file after return.
// Version must still succeed. After restoring the catalog, list --json must succeed with valid JSON
// on stdout and empty stderr.
func TestCatalogFailureResultDestination(t *testing.T) {
	originalDescription := providerconfig.JSONXAI

	var description map[string]any
	if err := json.Unmarshal(originalDescription, &description); err != nil {
		t.Fatalf("💣 decode registered provider: %v", err)
	}

	description["id"] = "mismatched-registration"

	malformedDescription, err := json.Marshal(description)
	if err != nil {
		t.Fatalf("💣 encode registered provider: %v", err)
	}

	providerconfig.JSONXAI = malformedDescription

	t.Cleanup(func() { providerconfig.JSONXAI = originalDescription })

	for _, filenameMode := range []bool{false, true} {
		resultsPath := filepath.Join(t.TempDir(), "results.json")

		arguments := []string{"--json", "--save-results", resultsPath, "a boat"}
		if filenameMode {
			arguments = append([]string{"--print-filename"}, arguments...)
		}

		status, stdout, stderr := captureCLI(t, arguments...)
		if status != 1 || stdout != "" || stderr != "" {
			t.Errorf("✗ early failure routing: status %d, stdout %q, stderr %q", status, stdout, stderr)
		}
		// #nosec G304 -- resultsPath belongs to this test's temporary directory.
		content, readErr := os.ReadFile(resultsPath)

		var failure output.FailureOutcome
		if readErr != nil || json.Unmarshal(content, &failure) != nil || failure.Status != "failed" || len(failure.Errors) != 1 {
			t.Errorf("✗ early failure absent from requested file: %q, %v", content, readErr)
		}

		if readErr == nil {
			assertNoOpenResult(t, resultsPath)
		}
	}

	status, stdout, stderr := captureCLI(t, "--version")
	if status != 0 || stdout != versionText(t) || stderr != "" {
		t.Errorf("✗ version depends on invalid catalog: %d %q %q", status, stdout, stderr)
	}

	providerconfig.JSONXAI = originalDescription

	status, stdout, stderr = captureCLI(t, "list", "--json")
	if status != 0 || !json.Valid([]byte(stdout)) || stderr != "" {
		t.Errorf("✗ later invocation inherited output state: %d %q %q", status, stdout, stderr)
	}

	if !t.Failed() {
		t.Log("✓ early failure routing, closure, and later invocation independence hold")
	}
}

// TestHelpTopicArity verifies invariant #23: Help topic arity.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// bild help with no topic or one of list, info, or search must exit 0 with nonempty stdout and
// empty stderr. bild help list extra must exit 2 with empty stdout and nonempty stderr.
func TestHelpTopicArity(t *testing.T) {
	for _, arguments := range [][]string{{"help"}, {"help", "list"}, {"help", "info"}, {"help", "search"}, {"help", "list", "extra"}} {
		status, stdout, stderr := captureCLI(t, arguments...)
		if len(arguments) > 2 {
			if status != 2 || stdout != "" || stderr == "" {
				t.Errorf("✗ surplus topics: status %d, stdout %q, stderr %q", status, stdout, stderr)
			}
		} else if status != 0 || stdout == "" || stderr != "" {
			t.Errorf("✗ valid help %q: status %d, stdout %q, stderr %q", arguments, status, stdout, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ help accepts at most one topic")
	}
}

// TestCommandReaderOwnsInteraction verifies invariant #24: Explicit command input.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given terminal process input and diagnostics but a supplied strings.Reader, Command.Run for a
// one-word JSON prompt with an unknown model must map to exit 1, write a valid failed outcome to
// results, and leave diagnostics empty.
func TestCommandReaderOwnsInteraction(t *testing.T) {
	useStdin(t, "n\n", true)

	var (
		results bytes.Buffer
		err     error
	)

	diagnostics := captureDiagnosticOutput(t, true, func() {
		command := createCommand(&bildApp{})
		command.Reader = strings.NewReader("NO\n")
		command.Writer = &results
		command.ErrWriter = os.Stderr
		err = command.Run(context.Background(), []string{"bild", "--json", "--model", "invalid-presentation-selection", "boat"})
	})

	var outcome output.GenerationOutcome

	decodeErr := json.Unmarshal(results.Bytes(), &outcome)
	if runExitCode(err) != 1 || decodeErr != nil || outcome.Status != "failed" || len(diagnostics) != 0 {
		t.Errorf("✗ the supplied reader did not govern interaction: %v, JSON error %v, status %q, diagnostics %q", err, decodeErr, outcome.Status, diagnostics)
	}

	if !t.Failed() {
		t.Log("✓ an explicit command reader determines whether prompting is permitted")
	}
}

// TestHelpVersionFlagsAlone verifies invariant #25: Help and version flags alone.
//
// What is being tested:
// For the tested help or version flags combined with positional arguments, bild must exit 2 with
// empty stdout and print compact usage plus the message naming --help or --version. When combined
// with another flag or repeated, it must print usage without that message. Invalid combinations
// with JSON must emit one JSON object containing one error and leave stderr empty. Standalone
// supported forms must exit 0, and -h -- must print the general help page without diagnostics. Root
// flags before command words must exit 2 and identify the command in the flags-before-command
// error, using text or JSON as selected. When -h is a flag value or follows -- as a prompt, the run
// must exit 1 without the combination message.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestHelpVersionFlagsAlone(t *testing.T) {
	clearProviderKeys(t)

	combinationStart, _, _ := strings.Cut(HelpFlagCombinedForm, "%s")

	for _, c := range []struct {
		args      []string
		namedFlag string
	}{
		{[]string{"info", "google", "-h"}, "--help"},
		{[]string{"search", "veo", "--help"}, "--help"},
		{[]string{"a cat", "-h"}, "--help"},
		{[]string{"-h", "a cat"}, "--help"},
		{[]string{"-help", "a cat"}, "--help"},
		{[]string{"-h", "list"}, "--help"},
		{[]string{"-help", "list"}, "--help"},
		{[]string{"-v", "a cat"}, "--version"},
		{[]string{"a cat", "-v"}, "--version"},
	} {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 || stdout != "" {
			t.Errorf("✗ %v: exit %d, stdout %q; want the usage error 2 with an empty stdout", c.args, code, stdout)
		}

		if !strings.Contains(stderr, fmt.Sprintf(HelpFlagCombinedForm, c.namedFlag)) || !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: stderr lacks the combination message naming %s with compact usage: %q", c.args, c.namedFlag, stderr)
		}
	}

	for _, args := range [][]string{
		{"list", "-h", "-p"},
		{"-h", "-v"},
		{"-v", "-h"},
		{"-version", "-m", "gemini"},
		{"--version", "-m", "gemini"},
		{"-h", "-h"},
		{"-h", "--help"},
		{"-v", "-v"},
		{"search", "-m", "-h"},
	} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stdout != "" {
			t.Errorf("✗ %v: exit %d, stdout %q; want the usage error 2 with an empty stdout", args, code, stdout)
		}

		if !strings.Contains(stderr, "Usage:") || strings.Contains(stderr, combinationStart) {
			t.Errorf("✗ %v: stderr = %q, want compact usage without the combination message", args, stderr)
		}
	}

	for _, args := range [][]string{{"--help", "--json"}, {"list", "-j", "-h"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 2 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q; want the usage error 2 with an empty stderr", args, code, stderr)
		}

		if documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors"); len(documentErrors) != 1 {
			t.Errorf("✗ %v: errors = %#v, want one error in the JSON document", args, documentErrors)
		}
	}

	for _, args := range [][]string{{"list", "-h"}, {"help", "list"}, {"-v"}, {"list", "-v"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 0 {
			t.Errorf("✗ %v: exit %d, want 0 (stderr: %q)", args, code, stderr)
		}
	}

	code, stdout, stderr := captureCLI(t, "-h", "--")
	if code != 0 || stderr != "" || beforeTips(t, stdout) != beforeTips(t, helpText(t)) {
		t.Errorf("✗ [-h --]: exit %d, stderr %q; want the general help page on stdout and exit 0", code, stderr)
	}

	// A command's flags follow its word: a flag ahead of the word invokes the root command, and
	// the word is then out of place.
	for _, c := range []struct {
		args        []string
		commandWord string
	}{
		{[]string{"-v", "list"}, "list"},
		{[]string{"-m", "gemini", "info", "google"}, "info"},
	} {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 || stdout != "" || !strings.Contains(stderr, fmt.Sprintf(FlagsBeforeCommandForm, c.commandWord)) || !strings.Contains(stderr, "Usage:") {
			t.Errorf("✗ %v: exit %d, stdout %q, stderr %q; want the flags-before-command usage error naming %s", c.args, code, stdout, stderr, c.commandWord)
		}
	}

	for _, c := range []struct {
		args        []string
		commandWord string
	}{
		{[]string{"-j", "list", "-v"}, "list"},
		{[]string{"--json", "search", "-v", "veo"}, "search"},
	} {
		code, stdout, stderr := captureCLI(t, c.args...)
		if code != 2 || stderr != "" {
			t.Errorf("✗ %v: exit %d, stderr %q; want the usage error 2 with an empty stderr", c.args, code, stderr)
		}

		documentErrors := jsonArrayField(t, decodeJSONObject(t, stdout), "errors")
		if len(documentErrors) != 1 || fmt.Sprint(documentErrors[0]) != fmt.Sprintf(FlagsBeforeCommandForm, c.commandWord) {
			t.Errorf("✗ %v: errors = %#v, want the one flags-before-command message naming %s", c.args, documentErrors, c.commandWord)
		}
	}

	// A help form after the terminator is the prompt, and one given as a flag's value is that
	// value: neither is the help flag, so each run goes on to its own outcome, never the
	// combination error.
	specifier := qualifiedSpecifier(t, builtinDefaultPair(t, shippedCatalog(t)))
	for _, args := range [][]string{{"--model", specifier, "--", "-h"}, {"-m", "-h", "a cat"}} {
		code, _, stderr := captureCLI(t, args...)
		if code != 1 || strings.Contains(stderr, combinationStart) {
			t.Errorf("✗ %v: exit %d, stderr %q; want 1 with no combination error", args, code, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ a help or version flag beside other input is a usage error, and alone it serves its page")
	}
}

// TestCLIAPIKeyConfigGuidance verifies invariant #26: Credential configuration guidance.
//
// What is being tested:
// With the selected model's credentials absent, bild must exit 1 in text, JSON, and debug modes.
// The combined output must name the credential environment variable, api-keys configuration key,
// and ~/.bildomat/config.yml.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIAPIKeyConfigGuidance(t *testing.T) {
	setUserConfig(t, "{}\n")
	clearProviderKeys(t)
	modelPair := builtinDefaultPair(t, shippedCatalog(t))

	for _, outputFlag := range []string{"", "--json", "--debug"} {
		t.Run(outputFlag, func(t *testing.T) {
			t.Chdir(t.TempDir())

			arguments := []string{"--model", qualifiedSpecifier(t, modelPair)}
			if outputFlag != "" {
				arguments = append(arguments, outputFlag)
			}

			arguments = append(arguments, "a blue circle")

			exitCode, stdout, stderr := captureCLI(t, arguments...)
			if exitCode != 1 {
				t.Errorf("✗ missing credentials returned exit %d: %s%s", exitCode, stdout, stderr)
			}

			for _, setting := range []string{modelPair.Provider.APIKeyEnvVar, "api-keys." + modelPair.Provider.ID, "~/.bildomat/config.yml"} {
				if !strings.Contains(stdout+stderr, setting) {
					t.Errorf("✗ credential error omits %q: %s%s", setting, stdout, stderr)
				}
			}

			if !t.Failed() {
				t.Log("✓ the missing-credential error identifies the environment variable and config key")
			}
		})
	}
}

// TestCLIFlagChanges verifies invariant #27: Short generation flags.
//
// What is being tested:
// Given -i with a missing file, bild must exit 1 and print the formatted missing-input message.
// Given --json -n p, it must exit 1 after the blocked request and retain p as the prompt. Given -N
// 0 for a supporting model, it must exit 1 and print the adjustment from 0 to 1. Given -N 2.5, it
// must exit 2.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIFlagChanges(t *testing.T) {
	clearProviderKeys(t)

	// The built-in default model's provider holds a placeholder key, so the request reaches
	// input validation instead of the credential stop.
	defaultKeyVar := builtinDefaultPair(t, shippedCatalog(t)).Provider.APIKeyEnvVar

	missing := filepath.Join(t.TempDir(), "nope.png")
	t.Setenv(defaultKeyVar, "s06-placeholder-never-read")

	code, _, stderr := captureCLI(t, "-i", missing, "p")
	if code != 1 {
		t.Errorf("✗ -i: exit %d, want 1 (input validation failure)", code)
	}

	wantNotFound := fmt.Sprintf(output.InputMediaNotFound, missing)
	if !strings.Contains(stderr, wantNotFound) {
		t.Errorf("✗ -i: the mapped not-found message naming the path is missing: %q", stderr)
	}

	// The placeholder key keeps the default model resolving; the blocked network stops the run
	// at its request instead of reaching a provider.
	blockNetwork(t)

	code, stdout, _ := captureCLI(t, "--json", "-n", "p")
	if code != 1 {
		t.Errorf("✗ -n: exit %d, want 1 (request failure)", code)
	}

	if !strings.Contains(stdout, `"prompt": "p"`) {
		t.Errorf("✗ -n consumed the following argument: %q", stdout)
	}

	numImagesModel := ""

	for _, modelPair := range shippedCatalog(t).ModelDirectory() {
		if modelPair.Model.SupportsParam(params.FlagTypeImageN) {
			numImagesModel = modelPair.Provider.ID + "/" + modelPair.Model.ID

			break
		}
	}

	if numImagesModel == "" {
		t.Fatalf("💣 the shipped catalog has no model supporting --num-images")
	}

	code, _, stderr = captureCLI(t, "--model", numImagesModel, "-N", "0", "p")
	if code != 1 {
		t.Errorf("✗ -N 0: exit %d, want 1 (credential stop)", code)
	}

	wantFloor := fmt.Sprintf(output.FlagBelowLimit, "Images", "0", "1", "1")
	if !strings.Contains(stderr, wantFloor) {
		t.Errorf("✗ -N 0: the num-images floor warning is missing: %q", stderr)
	}

	code, _, _ = captureCLI(t, "-N", "2.5", "p")

	if code != 2 {
		t.Errorf("✗ -N 2.5: exit %d, want 2 (fractional integer)", code)
	}

	if !t.Failed() {
		t.Log("✓ -i, -n, and -N are the short forms of their declared flags")
	}
}

// TestCLIGenerationArmParsing verifies invariant #28: Generation argument parsing.
//
// What is being tested:
// Given a video duration one second above the largest allowed value, bild must exit 1 and print an
// adjustment to that largest value. Given --json and -- followed by -m, it must exit 1 and retain
// -m as the JSON prompt.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIGenerationArmParsing(t *testing.T) {
	clearProviderKeys(t)

	loadedCatalog := shippedCatalog(t)
	videoModel := fixedDurationVideoModel(t, loadedCatalog)
	specifier := qualifiedSpecifier(t, videoModel)
	submittedDuration, snappedDuration := durationOutsideFixedSet(t, videoModel)

	code, _, stderr := captureCLI(t, "--model", specifier, "--duration", submittedDuration, "p")
	if code != 1 {
		t.Errorf("✗ --model %s --duration %s: exit %d, want 1 (credential stop)", specifier, submittedDuration, code)
	}

	wantSnap := fmt.Sprintf(output.FlagAdjusted, flagRecordOf(t, loadedCatalog, params.FlagTypeDuration).FlagName, snappedDuration)
	if !strings.Contains(stderr, wantSnap) {
		t.Errorf("✗ --model %s --duration %s: the adjustment notice is missing: %q", specifier, submittedDuration, stderr)
	}

	code, stdout, _ := captureCLI(t, "--json", "--model", specifier, "--", "-m")
	if code != 1 {
		t.Errorf("✗ -- -m: exit %d, want 1 (credential stop)", code)
	}

	if !strings.Contains(stdout, `"prompt": "-m"`) {
		t.Errorf("✗ -- -m: the dash-led word after the terminator is not the prompt: %q", stdout)
	}

	if !t.Failed() {
		t.Log("✓ the generation arm parses its flags and takes a post-terminator dash-led word as the prompt")
	}
}

// TestCLIHelpUnknownTopic verifies invariant #29: Unknown help topic.
//
// What is being tested:
// bild help nonexistent must exit 2, leave stdout empty, and name nonexistent in stderr.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIHelpUnknownTopic(t *testing.T) {
	code, stdout, stderr := captureCLI(t, "help", "nonexistent")
	if code != 2 {
		t.Errorf("✗ help nonexistent exit %d, want 2", code)
	}

	if stdout != "" {
		t.Errorf("✗ help nonexistent printed on stdout: %q", stdout)
	}

	if !strings.Contains(stderr, "nonexistent") {
		t.Errorf("✗ help nonexistent stderr = %q, want the usage error naming the topic", stderr)
	}

	if !t.Failed() {
		t.Log("✓ an unknown help topic prints its usage error and exits 2")
	}
}

// TestStandaloneVersionIgnoresConfig verifies invariant #30: Independent version requests.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// Given malformed user configuration, bild --version must exit 0, print exactly the version line,
// and leave stderr empty. Combining --version with a prompt, --json, or list must exit 2.
func TestStandaloneVersionIgnoresConfig(t *testing.T) {
	homeDirectory := t.TempDir()
	t.Setenv("HOME", homeDirectory)

	configDirectory := filepath.Join(homeDirectory, ".bildomat")
	if err := os.Mkdir(configDirectory, 0o700); err != nil {
		t.Fatalf("💣 create config directory: %v", err)
	}

	if err := os.WriteFile(filepath.Join(configDirectory, "config.yml"), []byte("api-keys: ["), 0o600); err != nil {
		t.Fatalf("💣 create malformed config: %v", err)
	}

	status, stdout, stderr := captureCLI(t, "--version")
	if status != 0 || stdout != versionText(t) || stderr != "" {
		t.Errorf("✗ standalone version read configuration: status %d, stdout %q, stderr %q", status, stdout, stderr)
	}

	for _, arguments := range [][]string{{"--version", "a prompt"}, {"--version", "--json"}, {"--version", "list"}} {
		status, _, _ := captureCLI(t, arguments...)
		if status != 2 {
			t.Errorf("✗ invalid version combination %q returned %d", arguments, status)
		}
	}

	if !t.Failed() {
		t.Log("✓ version bypasses configuration only for a valid standalone request")
	}
}

// TestHelpPageFailure verifies invariant #31: Help page failure.
//
// What is being tested:
// Given malformed root and command help templates, bild -h and bild list -h must each exit 1, leave
// stdout empty, and write nonempty stderr.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestHelpPageFailure(t *testing.T) {
	clearProviderKeys(t)

	rootTemplate, commandTemplate := tmpl.HelpMainText, tmpl.HelpCommandText

	t.Cleanup(func() { tmpl.HelpMainText, tmpl.HelpCommandText = rootTemplate, commandTemplate })

	tmpl.HelpMainText, tmpl.HelpCommandText = "{{", "{{"

	for _, args := range [][]string{{"-h"}, {"list", "-h"}} {
		code, stdout, stderr := captureCLI(t, args...)
		if code != 1 || stdout != "" || stderr == "" {
			t.Errorf("✗ %v: exit %d, stdout %q, stderr %q; want 1 with the failure on stderr alone", args, code, stdout, stderr)
		}
	}

	if !t.Failed() {
		t.Log("✓ a help page the library cannot print fails the run with its error on stderr")
	}
}

// TestCLIMalformedValues verifies invariant #32: CLI malformed values.
//
// What is being tested:
// Given an unknown Unicode flag, -n=banana, strength 1e309, or an overflowing num-images integer,
// parseCfg must return errs.ErrCLIFlagParse without invoking its capture action.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIMalformedValues(t *testing.T) {
	for _, args := range [][]string{
		{"--模型", "x", "p"},
		{"-n=banana", "p"},
		{"--strength", "1e309", "p"},
		{"-N", "99999999999999999999", "p"},
	} {
		_, called, err := parseCfg(t, args...)
		if !errors.Is(err, errs.ErrCLIFlagParse) {
			t.Errorf("✗ %v: err = %v, want ErrFlagParse", args, err)
		}

		if called {
			t.Errorf("✗ %v: handler ran despite a parse failure", args)
		}
	}

	if !t.Failed() {
		t.Log("✓ unknown unicode flags, bad bools, and numeric overflows all fail as parse errors")
	}
}

// TestCLIOutputDirFlagRemoved verifies invariant #33: CLI output directory flag removed.
//
// What is being tested:
// Given --output-dir or -O, parseCfg must return errs.ErrCLIFlagParse without invoking its capture
// action.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIOutputDirFlagRemoved(t *testing.T) {
	for _, removedFlag := range []string{"--output-dir", "-O"} {
		_, called, err := parseCfg(t, removedFlag, "out", "prompt")
		if !errors.Is(err, errs.ErrCLIFlagParse) {
			t.Errorf("✗ %s error = %v, want ErrCLIFlagParse", removedFlag, err)
		}

		if called {
			t.Errorf("✗ generation ran after the removed %s flag was supplied", removedFlag)
		}
	}

	if !t.Failed() {
		t.Log("✓ the removed output-dir flag fails during parsing and does not start generation")
	}
}

// TestCLIBlankPrompt verifies invariant #34: Prompt trimming.
//
// What is being tested:
// Given a whitespace-only prompt, parseCfg must return errs.ErrCLIPromptMissing without invoking
// capture. Given a cat surrounded by whitespace and --output-path out/, it must succeed and capture
// Prompt=a cat and OutPath=out/ as explicitly set values.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIBlankPrompt(t *testing.T) {
	_, called, err := parseCfg(t, "--output-path", "out/", " ")
	if called || !errors.Is(err, errs.ErrCLIPromptMissing) {
		t.Errorf("✗ a whitespace-only prompt: called=%v err=%v, want the missing-prompt error and no run", called, err)
	}

	cfg, called, err := parseCfg(t, "--output-path", "out/", "  a cat\t")
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	wantSet(t, "Prompt", cfg.genInputs.Prompt, "a cat")
	wantSet(t, "OutPath", cfg.genInputs.OutPath, "out/")

	if !t.Failed() {
		t.Log("✓ the prompt is trimmed, and a whitespace-only prompt is a missing prompt")
	}
}

// TestCLIBulkArgs verifies invariant #35: High-volume CLI arguments.
//
// What is being tested:
// Given 100 input-media flags and a 10,000-character prompt, parseCfg must succeed, invoke capture,
// return 100 media sources, and retain the exact prompt as an explicitly set value.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLIBulkArgs(t *testing.T) {
	long := strings.Repeat("x", 10_000)

	args := make([]string, 0, 202)
	for i := range 100 {
		args = append(args, "-i", "p"+string(rune('0'+i%10))+".png")
	}

	args = append(args, long)

	cfg, called, err := parseCfg(t, args...)
	if err != nil || !called {
		t.Fatalf("💣 run failed: called=%v err=%v", called, err)
	}

	if paths := inputMediaSources(t, cfg.params); len(paths) != 100 {
		t.Errorf("✗ input-media path count = %d, want 100", len(paths))
	}

	wantSet(t, "Prompt", cfg.genInputs.Prompt, long)

	if !t.Failed() {
		t.Log("✓ 100 input images and a 10k-char prompt parse intact")
	}
}

// TestCLISentinels verifies invariant #36: CLI error identities.
//
// What is being tested:
// With no arguments, parseCfg must return errs.ErrCLIPromptMissing. With --bogus, it must return
// errs.ErrCLIFlagParse. Neither case may invoke the capture action.
//
// Test class: Expanded.
// Test layer: Hardening/adversarial.
func TestCLISentinels(t *testing.T) {
	_, called, err := parseCfg(t)
	if !errors.Is(err, errs.ErrCLIPromptMissing) {
		t.Errorf("✗ no-args error = %v, want ErrPromptMissing", err)
	}

	if called {
		t.Errorf("✗ handler ran despite a missing prompt")
	}

	_, called, err = parseCfg(t, "--bogus")
	if !errors.Is(err, errs.ErrCLIFlagParse) {
		t.Errorf("✗ parse error = %v, want ErrFlagParse", err)
	}

	if called {
		t.Errorf("✗ handler ran despite a parse error")
	}

	if !t.Failed() {
		t.Log("✓ prompt-missing and parse failures surface their sentinels without running the handler")
	}
}

// TestExperimentalOptions verifies invariant #37: Experimental options.
// Test class: Expanded.
// Test layer: Hardening/adversarial.
// What is being tested:
// bild --help must exit 0 and omit --debug, --persist-record, and --reuse. Given a reuse selection
// with a missing separator or value, unknown identifier, unsupported provider, or repeated --reuse,
// bild must exit 2.
func TestExperimentalOptions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GOOGLE_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")

	code, stdout, _ := captureCLI(t, "--help")
	if code != 0 {
		t.Errorf("✗ help exit %d", code)
	}

	for _, option := range []string{"--debug", "--persist-record", "--reuse"} {
		if strings.Contains(stdout, option) {
			t.Errorf("✗ experimental option is public: %s", option)
		}
	}

	for _, testCase := range []struct {
		model string
		reuse []string
	}{
		{"google/veo-3.1-generate-preview", []string{"--reuse", "veo-extend"}},
		{"google/veo-3.1-generate-preview", []string{"--reuse", "veo-extend="}},
		{"google/veo-3.1-generate-preview", []string{"--reuse", "unknown=https://example.com/video"}},
		{"openai/gpt-image-1", []string{"--reuse", "veo-extend=https://example.com/video"}},
		{"google/veo-3.1-generate-preview", []string{"--reuse", "veo-extend=https://example.com/a", "--reuse", "veo-extend=https://example.com/b"}},
	} {
		arguments := append([]string{"--model", testCase.model, "--output-path", filepath.Join(t.TempDir(), "boat.mp4"), "--json"}, testCase.reuse...)
		arguments = append(arguments, "a paper boat")

		code, stdout, _ := captureCLI(t, arguments...)
		if code != 2 {
			t.Errorf("✗ invalid reuse exit %d, output %s", code, stdout)
		}
	}

	if !t.Failed() {
		t.Log("✓ experimental options remain hidden and reuse validates early")
	}
}

// FuzzCLIDispatch verifies invariant #38: Classified command dispatch.
//
// What is being tested:
// For list or info followed by three fuzzed tokens processed by containedToken, Command.Run must
// not panic. Any returned error must match errs.ErrCLI, errs.ErrModelResolve, errs.ErrKeyMissing,
// errs.ErrInputMedia, or errs.ErrOutputFile, or implement cli.ExitCoder.
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzCLIDispatch(f *testing.F) {
	flags := params.Flags()

	loadedCatalog, err := catalog.LoadCatalog(flags, registeredTestSources(f, providerRegistrations())...)
	if err != nil {
		f.Fatalf("💣 catalog construction failed: %v", err)
	}

	// The seeds name a registered provider, a video model, and an image model of the catalog.
	if len(loadedCatalog.Providers) == 0 {
		f.Fatalf("💣 the shipped catalog holds no provider to seed")
	}

	videoModel := firstModelOfMedia(f, loadedCatalog, media.Video)
	imageModel := firstModelOfMedia(f, loadedCatalog, media.Image)

	seeds := []struct {
		wordIndex                      uint8
		tokenOne, tokenTwo, tokenThree string
	}{
		{0, "", "", ""},
		{0, "--model", videoModel.Model.ID, "p"},
		{0, "--image", "", ""},
		{0, "--bogus", "", ""},
		{0, "--providers", "--models", ""},
		{0, "--image", "--video", ""},
		{0, "extra", "", ""},
		{1, loadedCatalog.Providers[0].ID, "", ""},
		{1, "bogus-prov", "", ""},
		{1, imageModel.Model.ID, "--help", ""},
		{1, "in", "white", "halter-top"},
		{1, "--version", "", ""},
	}
	for _, seed := range seeds {
		f.Add(seed.wordIndex, seed.tokenOne, seed.tokenTwo, seed.tokenThree)
	}

	// This target exercises list and info dispatch; other targets cover help.
	commandWords := []string{"list", "info"}

	f.Fuzz(func(t *testing.T, wordIndex uint8, tokenOne, tokenTwo, tokenThree string) {
		clearProviderKeys(t)
		t.Chdir(t.TempDir())

		app := testApp(t, loadedCatalog)
		command := quietCommand(app)
		argv := []string{
			"bild",
			commandWords[int(wordIndex)%len(commandWords)],
			containedToken(t, tokenOne),
			containedToken(t, tokenTwo),
			containedToken(t, tokenThree),
		}

		runErr := command.Run(context.Background(), argv)
		if runErr != nil && !classifiedDispatchError(t, runErr) {
			t.Errorf("✗ unclassified dispatch outcome for %q: %v", argv, runErr)
		}

		if !t.Failed() {
			t.Logf("✓ dispatch stayed within the command-boundary classifications")
		}
	})
}

// FuzzHelpVersionArguments verifies invariant #39: Command line under arbitrary help, version, and
// command-word arguments.
//
// What is being tested:
// For up to four tokens selected from the help/version vocabulary, bild must not panic and must
// return status 0, 1, or 2. Status 2 must produce either empty stdout with Usage: on stderr, or one
// valid JSON object on stdout with empty stderr. The vocabulary excludes --print-filename.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzHelpVersionArguments(f *testing.F) {
	vocabulary := []string{
		"", "-h", "--help", "-help", "-v", "--version", "-version", "-j", "--json", "--debug",
		"list", "info", "search", "help", "--", "a cat", "-m", "zzz-not-a-model", "--providers", "google",
	}

	for _, seed := range [][4]uint8{{1, 0, 0, 0}, {1, 14, 0, 0}, {4, 10, 0, 0}, {7, 10, 4, 0}, {10, 1, 18, 0}, {2, 7, 0, 0}, {16, 1, 15, 0}, {12, 16, 1, 0}, {1, 1, 0, 0}, {13, 10, 1, 0}} {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}

	f.Fuzz(func(t *testing.T, first, second, third, fourth uint8) {
		clearProviderKeys(t)
		useStdin(t, "", false)
		t.Chdir(t.TempDir())

		var args []string

		for _, tokenIndex := range []uint8{first, second, third, fourth} {
			if token := vocabulary[int(tokenIndex)%len(vocabulary)]; token != "" {
				args = append(args, token)
			}
		}

		code, stdout, stderr := captureCLI(t, args...)
		if code < 0 || code > 2 {
			t.Errorf("✗ %q: exit %d, want 0, 1, or 2", args, code)
		}

		if code == 2 {
			textForm := stdout == "" && strings.Contains(stderr, "Usage:")
			jsonForm := stderr == "" && json.Valid([]byte(stdout)) && strings.HasPrefix(strings.TrimSpace(stdout), "{")

			if !textForm && !jsonForm {
				t.Errorf("✗ %q: a usage error printed stdout %q and stderr %q; want compact usage alone or one JSON document alone", args, stdout, stderr)
			}
		}

		if !t.Failed() {
			t.Logf("✓ the command line stayed within the usage-error output contract")
		}
	})
}

// FuzzSearchTerms verifies invariant #40: Search command under arbitrary terms and flags.
//
// What is being tested:
// For bild search followed by up to four vocabulary tokens, including empty strings, bild must not
// panic and must return status 0, 1, or 2. Status 0 must leave stderr empty. Status 2 must produce
// either empty stdout with Usage: on stderr, or one valid JSON object on stdout with empty stderr.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzSearchTerms(f *testing.F) {
	const omittedToken = "<omitted>"

	vocabulary := []string{
		omittedToken, "", "openai", "zzz-matches-no-model", "(", ".", "-x", "--exclude", "-x=", "--exclude=openrouter",
		"-r", "-j", "-m", "-p", "-h", "--", "flux dream",
	}

	for _, seed := range [][4]uint8{{2, 6, 3, 0}, {6, 1, 0, 0}, {1, 0, 0, 0}, {10, 5, 6, 4}, {6, 14, 12, 0}, {11, 6, 1, 0}, {2, 2, 0, 0}, {15, 6, 0, 0}, {0, 0, 0, 0}, {8, 0, 0, 0}} {
		f.Add(seed[0], seed[1], seed[2], seed[3])
	}

	f.Fuzz(func(t *testing.T, first, second, third, fourth uint8) {
		clearProviderKeys(t)
		useStdin(t, "", false)

		args := []string{"search"}

		for _, tokenIndex := range []uint8{first, second, third, fourth} {
			if token := vocabulary[int(tokenIndex)%len(vocabulary)]; token != omittedToken {
				args = append(args, token)
			}
		}

		code, stdout, stderr := captureCLI(t, args...)
		if code < 0 || code > 2 {
			t.Errorf("✗ %q: exit %d, want 0, 1, or 2", args, code)
		}

		if code == 0 && stderr != "" {
			t.Errorf("✗ %q: a served search printed on stderr: %q", args, stderr)
		}

		if code == 2 {
			textForm := stdout == "" && strings.Contains(stderr, "Usage:")
			jsonForm := stderr == "" && json.Valid([]byte(stdout)) && strings.HasPrefix(strings.TrimSpace(stdout), "{")

			if !textForm && !jsonForm {
				t.Errorf("✗ %q: a usage error printed stdout %q and stderr %q; want compact usage alone or one JSON document alone", args, stdout, stderr)
			}
		}

		if !t.Failed() {
			t.Logf("✓ the search command stayed within its output contract")
		}
	})
}

// FuzzCLIParse verifies invariant #41: CLI parsing under arbitrary arguments.
//
// What is being tested:
// Given any three argument strings, the command built by testCommand must not panic. If its capture
// action runs, Command.Run must return nil and the captured prompt must be nonempty. Otherwise, any
// error must match errs.ErrCLIFlagParse, errs.ErrCLIPromptMissing, or errs.ErrCLIOneArg.
//
// Test class: Expanded.
// Test layer: Fuzzing.
func FuzzCLIParse(f *testing.F) {
	seeds := [][3]string{
		{"-h", "", ""},
		{"-m", "x", "p"},
		{"--", "-m", "p"},
		{"-i=false", "p", ""},
		{"--strength", "abc", "p"},
		{"-3", "-", ""},
		{"--bogus", "", ""},
		{"-I", "a,b", "p"},
		{"-model=x", "p", "out"},
		{"-v", "-h", ""},
		{"-s", "1280x720", "p"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1], s[2])
	}

	f.Fuzz(func(t *testing.T, a, b, c string) {
		var prompt string

		called := false
		cmd := testCommand(t, func(genInputs RunFlags, _ params.FlagInputs) error {
			called = true
			prompt = genInputs.Prompt.ValOr("")

			return nil
		})
		cmd.Writer = io.Discard
		cmd.ErrWriter = io.Discard

		err := cmd.Run(context.Background(), []string{"bild", a, b, c})
		if called {
			if err != nil {
				t.Errorf("✗ handler ran yet Run errored: %v (args %q %q %q)", err, a, b, c)
			}

			if prompt == "" {
				t.Errorf("✗ handler ran with an empty prompt (args %q %q %q)", a, b, c)
			}
		} else if err != nil && !errors.Is(err, errs.ErrCLIFlagParse) && !errors.Is(err, errs.ErrCLIPromptMissing) && !errors.Is(err, errs.ErrCLIOneArg) {
			t.Errorf("✗ unexpected error category: %v (args %q %q %q)", err, a, b, c)
		}

		if !t.Failed() {
			t.Logf("✓ parsing stayed within the command-boundary contract")
		}
	})
}

// assertNoOpenResult checks that no live descriptor still owns the requested result file.
func assertNoOpenResult(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("💣 inspect result file: %v", err)
	}

	resultStat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("💣 filesystem does not expose file identity")
	}

	descriptors, err := os.ReadDir("/dev/fd")
	if err != nil {
		t.Fatalf("💣 enumerate process descriptors: %v", err)
	}

	for _, descriptor := range descriptors {
		fd, parseErr := strconv.Atoi(descriptor.Name())
		if parseErr != nil {
			continue
		}

		var descriptorStat syscall.Stat_t
		if syscall.Fstat(fd, &descriptorStat) == nil && descriptorStat.Dev == resultStat.Dev && descriptorStat.Ino == resultStat.Ino {
			t.Errorf("✗ result file remains open on descriptor %d", fd)
		}
	}
}

// firstModelOfMedia returns the first model of the catalog, in catalog order, that produces the
// given medium.
func firstModelOfMedia(test testing.TB, loadedCatalog *catalog.Catalog, media media.Kind) catalog.ProvModelPair {
	test.Helper()

	for _, pair := range loadedCatalog.ModelDirectory() {
		if pair.Model.Media == media {
			return pair
		}
	}

	test.Fatalf("💣 the shipped catalog holds no %s model", media)

	return catalog.ProvModelPair{}
}

// containedToken strips leading slash and tilde characters and replaces each .. with . in a fuzzed
// argument.
func containedToken(test testing.TB, token string) string {
	test.Helper()

	token = strings.TrimLeft(token, "/~")

	return strings.ReplaceAll(token, "..", ".")
}

// classifiedDispatchError reports whether a dispatch error belongs to the command boundary's
// documented classifications.
func classifiedDispatchError(test testing.TB, err error) bool {
	test.Helper()

	for _, root := range []error{
		errs.ErrCLI, errs.ErrModelResolve, errs.ErrKeyMissing,
		errs.ErrInputMedia, errs.ErrOutputFile,
	} {
		if errors.Is(err, root) {
			return true
		}
	}
	// The library reports an unusable command word with its own exit-coded error, which the run
	// boundary renders as a usage error.
	var libraryExit cli.ExitCoder

	return errors.As(err, &libraryExit)
}

// fixtureProviderCfg renders a valid provider configuration with the supplied provider and model
// IDs.
func fixtureProviderCfg(test testing.TB, providerID, modelID string) string {
	test.Helper()

	return fmt.Sprintf(`{
	  "id": %[1]q, "displayName": "Fixture", "apiKeyEnvVar": "FIXT_KEY",
	  "models": [
	    {"id": %[2]q, "name": %[2]q, "media": "image", "params": [
	      {"paramID": "aspect_ratio", "flagID": "aspect-ratio"}
	    ]}
	  ],
	  "config": {"imageAPI": {
	    "genURL": "https://example.test/img",
	    "inputMediaPayloadType": "json", "inputMediaProvParam": "image", "inputMediaListProvParam": "images", "inputMediaStyle": %[3]q,
	    "fallbackExt": ".png"
	  }}
	}`, providerID, modelID, catalog.InputMediaSingle)
}

// quietCommand creates a command over the supplied application and disables loading hooks. Command
// writers discard their output.
func quietCommand(app *bildApp) *cli.Command {
	command := createCommand(app)
	command.Writer = io.Discard
	command.ErrWriter = io.Discard
	command.Before = nil

	for _, subCmd := range command.Commands {
		subCmd.Before = nil
	}

	return command
}

// flagRecordOf returns the catalog's flag record declaring the given flag ID, failing the test when
// the catalog declares none.
func flagRecordOf(t *testing.T, loadedCatalog *catalog.Catalog, flagID params.FlagType) params.Flag {
	t.Helper()

	for _, flagRecord := range loadedCatalog.Flags {
		if flagRecord.FlagID == flagID {
			return flagRecord
		}
	}

	t.Fatalf("💣 the catalog declares no flag record for %s", flagID)

	return params.Flag{}
}
