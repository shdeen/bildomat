package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/terminal"
	"github.com/urfave/cli/v3"
)

// commandInvocation owns the streams, result file, presentation choices, and completed generation
// facts of one command.
//   - stdin, stdout, stderr: the caller-owned process streams
//   - results: the current destination for ordinary or JSON results
//   - file: the results file owned and closed by this invocation, if any
//   - outcome: accumulated generation facts for final reporting
//   - generationElapsed: successful generation duration reported by the spinner
//   - reportedAdjustments: number of adjustment notices already delivered
//   - jsonOutput, printFilename, debug: selected presentation modes
//   - styled, diagnosticStyled: whether result and diagnostic streams support terminal styling
//   - interactive: whether stdin and stderr are terminals
//   - animate: whether generation progress may be shown
type commandInvocation struct {
	stdin               io.Reader
	stdout              io.Writer
	stderr              io.Writer
	results             io.Writer
	file                *os.File
	outcome             *output.GenerationOutcome
	generationElapsed   time.Duration
	reportedAdjustments int
	jsonOutput          bool
	printFilename       bool
	debug               bool
	styled              bool
	diagnosticStyled    bool
	interactive         bool
	animate             bool
}

// newInvocation stores the supplied streams and detects terminal support for each one. It enables
// interaction only when both input and diagnostics are terminals.
func newInvocation(stdin io.Reader, stdout, stderr io.Writer) commandInvocation {
	return commandInvocation{
		stdin: stdin, stdout: stdout, stderr: stderr, results: stdout,
		interactive:      terminal.IsTerminal(stdin) && terminal.IsTerminal(stderr),
		styled:           terminal.IsTerminal(stdout),
		diagnosticStyled: terminal.IsTerminal(stderr),
	}
}

// selectMode stores parsed presentation choices without opening a results file.
func (invocation *commandInvocation) selectMode(command *cli.Command) {
	root := command.Root()
	invocation.debug = command.Bool(RunFlagDebug) || root.Bool(RunFlagDebug)
	invocation.jsonOutput = command.Bool(RunFlagJSON) || root.Bool(RunFlagJSON)
	invocation.printFilename = command.Bool(RunFlagPrintFilename)
	invocation.animate = invocation.styled && !invocation.jsonOutput && !invocation.printFilename
}

// openResults creates missing parent directories, then creates or truncates the requested results
// file. Without a path, filename mode discards ordinary results; other modes keep stdout.
func (invocation *commandInvocation) openResults(path string) error {
	if path == "" {
		if invocation.printFilename {
			invocation.results = io.Discard
			invocation.styled = false
		}

		return nil
	}

	expanded, err := artifact.ExpandHome(path)
	if err != nil {
		return err
	}
	// #nosec G301 -- user-selected result directories follow the user's umask.
	if err := os.MkdirAll(filepath.Dir(expanded), 0o755); err != nil {
		return errs.FileError(errs.FileOpMkdir, expanded, errs.ErrOutputFileMkdir, err)
	}
	// #nosec G302 G304 -- the user explicitly selected this writable result path.
	file, err := os.OpenFile(expanded, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return errs.FileError(errs.FileOpCreate, expanded, errs.ErrOutputFileCreate, err)
	}

	invocation.file, invocation.results = file, file
	invocation.styled, invocation.animate = false, false

	return nil
}

// finish closes the owned results file and restores stdout as the result destination. It joins
// close and diagnostic-delivery failures with commandErr without rewriting results.
func (invocation *commandInvocation) finish(commandErr error) error {
	file := invocation.file
	invocation.file = nil

	invocation.results = invocation.stdout

	if file == nil {
		return commandErr
	}

	if err := file.Close(); err != nil {
		closeErr := errs.FileError(errs.FileOpClose, file.Name(), errs.ErrOutputFileClose, err)
		diagnosticErr := output.PrintError(invocation.stderr, closeErr, "", "", invocation.debug, invocation.diagnosticStyled)
		commandErr = errors.Join(commandErr, closeErr, diagnosticErr)
	}

	return commandErr
}

// fail renders a command failure before a generation has resolved its provider. The returned error
// retains the command cause and any failed delivery.
func (invocation *commandInvocation) fail(err error) error {
	return errors.Join(err, invocation.printFailure(err, "", ""))
}

// printFailure renders a failure with its available provider and model context. It returns delivery
// errors separately from the primary failure being described.
func (invocation *commandInvocation) printFailure(err error, providerName, modelName string) error {
	usageError := errors.Is(err, errs.ErrCLI)
	switch {
	case invocation.jsonOutput:
		reportErr := output.PrintJSON(invocation.results, output.NewFailureOutcome(err, providerName, modelName, invocation.debug, usageError))
		if reportErr != nil {
			return errors.Join(reportErr, output.PrintError(invocation.stderr, errors.Join(err, reportErr), providerName, modelName, invocation.debug, invocation.diagnosticStyled))
		}

		return nil
	case usageError:
		return output.PrintUsageError(invocation.stderr, err, invocation.debug, invocation.diagnosticStyled)
	default:
		return output.PrintError(invocation.stderr, err, providerName, modelName, invocation.debug, invocation.diagnosticStyled)
	}
}

// reportOutputError writes the output error and any completed file reports to diagnostics. It
// returns the original error joined with failures from either diagnostic write.
func (invocation *commandInvocation) reportOutputError(err error, providerName, modelName string) error {
	if err == nil {
		return nil
	}

	diagnosticErr := output.PrintError(invocation.stderr, err, providerName, modelName, invocation.debug, invocation.diagnosticStyled)

	var savedFilesErr error
	if invocation.outcome != nil {
		savedFilesErr = invocation.printSavedFiles(invocation.stderr, false)
	}

	return errors.Join(err, diagnosticErr, savedFilesErr)
}

// printSavedFiles reports the invocation's complete saved-file facts to one destination.
func (invocation *commandInvocation) printSavedFiles(destination io.Writer, styled bool) error {
	for _, savedFile := range invocation.outcome.Artifacts {
		if err := output.PrintSavedFile(destination, savedFile, styled); err != nil {
			return err
		}
	}

	return nil
}

// reportGeneration completes the outcome and writes its text or JSON report and requested
// filenames. If reporting fails, it reports that error on diagnostics and joins it with the
// generation error.
func (invocation *commandInvocation) reportGeneration(startedAt time.Time, providerName, modelName string, generationErr error) error {
	invocation.outcome.Complete(startedAt, providerName, modelName, generationErr, invocation.debug)

	var reportErr error
	if invocation.jsonOutput {
		reportErr = output.PrintJSON(invocation.results, invocation.outcome)
	} else {
		reportErr = invocation.reportText(providerName, modelName, generationErr)
	}

	if invocation.printFilename {
		for _, savedFile := range invocation.outcome.Artifacts {
			if err := output.WriteText(invocation.stdout, "%s\n", savedFile.Path); err != nil {
				reportErr = errors.Join(reportErr, err)

				break
			}
		}
	}

	if reportErr != nil {
		return invocation.reportOutputError(errors.Join(generationErr, reportErr), providerName, modelName)
	}

	return generationErr
}

// reportText writes adjustments and file notices, the available elapsed time, saved-file reports,
// and any generation error. It attempts each report even if an earlier write fails.
func (invocation *commandInvocation) reportText(providerName, modelName string, generationErr error) error {
	reportErr := invocation.printAdjustments()
	if len(invocation.outcome.Notices) > 0 {
		reportErr = errors.Join(reportErr, output.WriteText(invocation.stderr, "%s\n", strings.Join(invocation.outcome.Notices, "\n")))
	}

	if invocation.generationElapsed > 0 {
		reportErr = errors.Join(reportErr, output.PrintGenerationCompleted(invocation.results, output.ElapsedText(invocation.generationElapsed)))
	}

	reportErr = errors.Join(reportErr, invocation.printSavedFiles(invocation.results, invocation.styled))
	if generationErr != nil {
		reportErr = errors.Join(reportErr, invocation.printFailure(generationErr, providerName, modelName))
	}

	return reportErr
}

// printAdjustments delivers the text notices not yet reported. Successful delivery advances the
// count so final reporting prints only changes discovered during generation.
func (invocation *commandInvocation) printAdjustments() error {
	if invocation.jsonOutput || invocation.reportedAdjustments == len(invocation.outcome.Adjustments) {
		return nil
	}

	adjustments := invocation.outcome.Adjustments[invocation.reportedAdjustments:]

	notices := make([]string, 0, len(adjustments))
	for _, adjustment := range adjustments {
		notices = append(notices, adjustment.Notice)
	}

	if err := output.WriteText(invocation.stderr, "%s\n", strings.Join(notices, "\n")); err != nil {
		return err
	}

	invocation.reportedAdjustments = len(invocation.outcome.Adjustments)

	return nil
}
