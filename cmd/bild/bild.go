package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"os"
	"os/signal"
	"slices"
	"strings"
	"time"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/catalog"
	userconfig "github.com/shdeen/bildomat/internal/config"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/generation"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
	"github.com/shdeen/bildomat/internal/output"
	"github.com/shdeen/bildomat/internal/params"
	tmpl "github.com/shdeen/bildomat/internal/templates"
	"github.com/shdeen/bildomat/internal/terminal"
	"github.com/urfave/cli/v3"
)

// bildApp owns the loaded settings and command state for each invocation. The selected command
// prepares them after parsing the command line.
//   - invocation: the command's streams, presentation choices, and generation facts
//   - catalog: the decoded provider configs and the parameter flag records
//   - apiKeys: configured API keys owned by the command
//   - defaultModel: the user config's default model, used when --model is omitted; empty when the
//     config names none, in which case the catalog's provider default applies
//   - defaultOutDir: the directory used when no output location is given; empty for the working
//     directory
type bildApp struct {
	invocation    commandInvocation
	catalog       *catalog.Catalog
	apiKeys       map[string]string
	defaultModel  string
	defaultOutDir string
}

// load reads user settings and the provider catalog and applies configured API keys. It prints
// configuration faults as warnings and returns any warning-write or catalog-load error.
func (bild *bildApp) load() error {
	settings, configPath, configFaults := userconfig.Load()
	for _, fault := range configFaults {
		if err := output.PrintUserConfigWarning(bild.invocation.stderr, fault, bild.invocation.diagnosticStyled); err != nil {
			return err
		}
	}

	bild.defaultModel = settings.DefaultModel
	bild.defaultOutDir = settings.DefaultOutputDir
	registrations := providerRegistrations()

	sources := make([]catalog.Source, 0, len(registrations))
	for _, registration := range registrations {
		sources = append(sources, catalog.Source{ProviderID: registration.ProviderID, ConfigBytes: registration.ConfigBytes})
	}

	loadedCatalog, err := catalog.LoadCatalog(params.Flags(), sources...)
	if err != nil {
		return err
	}

	bild.catalog = loadedCatalog
	for _, providerID := range bild.applyAPIKeys(settings.APIKeys) {
		if err := output.PrintUserConfigWarning(bild.invocation.stderr, userconfig.UnknownProviderFault(configPath, providerID), bild.invocation.diagnosticStyled); err != nil {
			return err
		}
	}

	return nil
}

// cliPrepareRoot captures the parsed command streams and output modes. For root requests, it opens
// the requested results file before loading settings and the catalog. Standalone version requests
// skip both loading and file creation; selected subcommands perform their own preparation.
func (bild *bildApp) cliPrepareRoot(ctx context.Context, root *cli.Command) (context.Context, error) {
	bild.invocation = newInvocation(root.Reader, root.Writer, root.ErrWriter)
	selectedCommand := root.Command(root.Args().First())

	flagSource := root
	if selectedCommand != nil {
		flagSource = selectedCommand
	}

	bild.invocation.selectMode(flagSource)

	if root.Bool(HelpFlag) && root.Args().Present() {
		return ctx, bild.invocation.fail(combinedFlagError(HelpFlag))
	}

	if selectedCommand != nil {
		return ctx, nil
	}

	if root.Bool(VersionFlag) {
		if root.Args().Present() {
			return ctx, bild.invocation.fail(combinedFlagError(VersionFlag))
		}

		return ctx, nil
	}

	if err := bild.invocation.openResults(root.String(RunFlagSaveResults)); err != nil {
		return ctx, bild.invocation.reportOutputError(err, "", "")
	}

	if err := bild.load(); err != nil {
		return ctx, bild.invocation.finish(bild.invocation.fail(err))
	}

	return ctx, nil
}

// cliPrepareCommand validates subcommand placement, then loads its settings and catalog using the
// already selected invocation streams.
func (bild *bildApp) cliPrepareCommand(ctx context.Context, command *cli.Command) (context.Context, error) {
	bild.invocation.selectMode(command)

	if command.Bool(HelpFlag) && command.Args().Present() {
		return ctx, bild.invocation.fail(combinedFlagError(HelpFlag))
	}

	if command.Root().NumFlags() > 0 {
		err := &errs.UsageError{Text: fmt.Sprintf(FlagsBeforeCommandForm, command.Name), Cause: errs.ErrCLIFlagsBeforeCommand}

		return ctx, bild.invocation.fail(err)
	}

	if err := bild.load(); err != nil {
		return ctx, bild.invocation.fail(err)
	}

	return ctx, nil
}

// resolveGenerator resolves the model and records its identity before constructing the provider, so
// construction failures retain the provider and model context.
func (bild *bildApp) resolveGenerator(genInputs *RunFlags) (catalog.ProvModelPair, generation.Generator, error) {
	modelInput, supplied := bild.runModelInput(genInputs)
	if !supplied {
		return catalog.ProvModelPair{}, nil, &errs.UsageError{Text: NoDefaultModel, Cause: errs.ErrCLINoDefaultModel}
	}

	pair, err := bild.resolveModelInput(modelInput)
	if err != nil {
		return catalog.ProvModelPair{}, nil, err
	}

	bild.invocation.outcome.Provider = pair.Provider.DisplayName
	bild.invocation.outcome.Model = pair.Model.ID
	generator, err := newGenerator(bild.catalog, pair.Provider.ID, providerRegistrations())

	return pair, generator, err
}

// cliRunGenerate serves standalone help/version requests or one generation. Every generation return
// closes its owned result file and retains all causes.
func (bild *bildApp) cliRunGenerate(ctx context.Context, command *cli.Command) error {
	if command.Bool(HelpFlag) {
		return bild.invocation.finish(bild.showHelpPage(command))
	}

	if command.Bool(VersionFlag) {
		return bild.invocation.reportOutputError(output.WriteText(bild.invocation.results, VersionReport+"\n", command.Name, command.Version), "", "")
	}

	return bild.invocation.finish(bild.runGeneration(ctx, command))
}

// runGeneration validates and confirms the prompt, collects the generation outcome, and reports it.
// When generation runs, final reporting follows artifact saving, record persistence, and cleanup.
func (bild *bildApp) runGeneration(ctx context.Context, command *cli.Command) error {
	if command.Args().Len() > 1 {
		return bild.invocation.fail(argCountError(command.Name, 1, command.Args().Len()))
	}

	genInputs := createRunInputs(command)

	prompt := genInputs.Prompt.ValOr("")
	if prompt == "" && !bild.promptIgnored(&genInputs) {
		return bild.invocation.fail(errs.ErrCLIPromptMissing)
	}

	paramInputs := createParamInputs(command, bild.catalog.Flags)
	startedAt := time.Now()
	bild.invocation.outcome = output.NewGenerationOutcome(startedAt, prompt, getGenFlagsInput(&genInputs, paramInputs))

	canceled, err := terminal.ConfirmOneWordPrompt(bild.invocation.stdin, bild.invocation.stderr, prompt, bild.invocation.interactive, bild.invocation.diagnosticStyled)
	if err == nil && canceled {
		bild.invocation.outcome.Status = output.StatusCanceled
	}

	var pair catalog.ProvModelPair
	if err == nil && !canceled {
		pair, err = bild.generate(ctx, &genInputs, paramInputs)
	}

	return bild.invocation.reportGeneration(startedAt, pair.Provider.DisplayName, pair.Model.Name, err)
}

// cliRunList renders the requested listing or the command's help page.
func (bild *bildApp) cliRunList(_ context.Context, command *cli.Command) error {
	if command.Bool(HelpFlag) {
		return bild.showHelpPage(command)
	}

	if command.Args().Len() != 0 {
		return bild.invocation.fail(argCountError(command.Name, 0, command.Args().Len()))
	}

	return bild.printListing(command, bild.selectedModels(command))
}

// cliRunSearch filters models by media and search terms, then renders the selected listing form. It
// returns validation, search, and output errors after reporting them.
func (bild *bildApp) cliRunSearch(_ context.Context, command *cli.Command) error {
	if command.Bool(HelpFlag) {
		return bild.showHelpPage(command)
	}

	if err := searchTermsError(command); err != nil {
		return bild.invocation.fail(err)
	}

	matches, err := catalog.SearchDirectory(bild.selectedModels(command), command.Args().Get(0), command.String(FilterFlagExclude), command.Bool(FilterFlagRegex))
	if err != nil {
		return bild.invocation.fail(err)
	}

	return bild.printListing(command, matches)
}

// cliRunInfo resolves provider or model details and propagates every rendering failure. Provider
// configuration failures retain their original identity.
func (bild *bildApp) cliRunInfo(_ context.Context, command *cli.Command) error {
	if command.Bool(HelpFlag) {
		return bild.showHelpPage(command)
	}

	if command.Args().Len() != 1 {
		return bild.invocation.fail(argCountError(command.Name, 1, command.Args().Len()))
	}

	modelInput := command.Args().Get(0)
	if provider, found := bild.catalog.Provider(modelInput); found {
		var providerModels []catalog.ProvModelPair

		selectedModels := bild.selectedModels(command)

		for index := range selectedModels {
			if selectedModels[index].Provider.ID == provider.ID {
				providerModels = append(providerModels, selectedModels[index])
			}
		}

		var reportErr error
		if bild.invocation.jsonOutput {
			reportErr = output.PrintJSON(bild.invocation.results, output.ProviderInfoPage(&provider, providerModels, bild.catalog.Flags))
		} else {
			reportErr = output.PrintProviderInfo(bild.invocation.results, &provider, providerModels, bild.catalog.Flags, mediaFilterFlagNames, bild.invocation.styled)
		}

		return bild.invocation.reportOutputError(reportErr, "", "")
	}

	if err := bild.catalog.ConfigError(modelInput); err != nil {
		return bild.invocation.fail(err)
	}

	pairs, err := bild.catalog.ResolveModelInput(modelInput)
	if err != nil {
		return bild.invocation.fail(err)
	}

	if len(pairs) > 1 {
		reportErr := output.PrintAmbiguity(bild.invocation.stderr, modelInput, pairs, bild.invocation.diagnosticStyled)

		return bild.invocation.fail(errors.Join(fmt.Errorf("%q, %w", modelInput, errs.ErrModelResolveConflict), reportErr))
	}

	if bild.invocation.jsonOutput {
		return bild.invocation.reportOutputError(output.PrintJSON(bild.invocation.results, output.ModelInfoPage(&pairs[0], bild.catalog.Flags)), "", "")
	}

	return bild.invocation.reportOutputError(output.PrintModelInfo(bild.invocation.results, &pairs[0], bild.catalog.Flags, bild.invocation.styled), "", "")
}

// promptIgnored reports whether the selected or default model requires no prompt. Unresolved or
// ambiguous model input returns false.
func (bild *bildApp) promptIgnored(genInputs *RunFlags) bool {
	modelInput, ok := bild.runModelInput(genInputs)
	if !ok {
		return false
	}

	provModelPairs, err := bild.catalog.ResolveModelInput(modelInput)

	return err == nil && len(provModelPairs) == 1 && provModelPairs[0].Model.PromptIgnored
}

// runModelInput returns the supplied or default model and whether one is available.
func (bild *bildApp) runModelInput(genInputs *RunFlags) (string, bool) {
	if modelInput, supplied := genInputs.Model.ValIf(); supplied {
		return modelInput, true
	}

	return bild.defaultModelKey()
}

// defaultModelKey prefers the configured default, then a catalog default with available
// credentials.
func (bild *bildApp) defaultModelKey() (string, bool) {
	if bild.defaultModel != "" {
		return bild.defaultModel, true
	}

	availableProviders := make(map[string]bool, len(bild.catalog.Providers))
	for providerIndex := range bild.catalog.Providers {
		description := &bild.catalog.Providers[providerIndex]
		_, credentialErr := bild.apiKey(description)
		availableProviders[description.ID] = credentialErr == nil
	}

	return bild.catalog.DefaultModelKey(availableProviders)
}

// generate resolves inputs, calls the provider, writes artifacts, and optionally saves a record. It
// returns any resolved provider and model even on failure for reporting context.
//
//nolint:funlen // Keep the ordered generation stages together; extracting the media-read error check creates a single-use pass-through.
func (bild *bildApp) generate(ctx context.Context, genInputs *RunFlags, userInputs params.FlagInputs) (catalog.ProvModelPair, error) {
	started := time.Now()

	if err := params.CheckFlagInputTypes(userInputs, bild.catalog.Flags); err != nil {
		return catalog.ProvModelPair{}, err
	}

	provModelPair, generator, err := bild.resolveGenerator(genInputs)
	if err != nil {
		return provModelPair, err
	}

	inputMediaSources, err := params.Value[[]string](userInputs, params.FlagTypeInputMedia)
	if err != nil {
		return provModelPair, err
	}

	inputMedia, err := media.ReadInputs(generation.SelectSources(inputMediaSources, &provModelPair.Model))
	if err != nil {
		return provModelPair, err
	}

	var suppliedJSON json.RawMessage
	if genInputs.PersistRecord {
		suppliedJSON, err = json.Marshal(userInputs)
		if err != nil {
			return provModelPair, fmt.Errorf("%q: %w, %w", provModelPair.Model.ID, errs.ErrJSONEncode, err)
		}
	}

	outPath, pathRecords, err := resolveOutputTarget(genInputs, &provModelPair.Model, userInputs, bild.defaultOutDir)
	if err != nil {
		return provModelPair, err
	}

	stem := outPath.Stem
	if stem == "" {
		stem = artifact.DefaultStem(provModelPair.Model.Media)
	}

	var record *metadata.Record
	if genInputs.PersistRecord {
		record = metadata.New(provModelPair.Provider.ID, provModelPair.Model.ID, appVersion(), genInputs.Prompt.ValOr(""), started)
		record.Describe(suppliedJSON, inputMediaSources)
	}

	// Keep interrupt handling installed through record persistence. Cancellation stops network
	// work but must still finalize responses already received.
	generationCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	finalStem, savedFiles, generationErr := bild.executeGeneration(generationCtx, generator, &provModelPair, genInputs, userInputs, inputMedia, outPath, pathRecords, record)
	if finalStem == "" {
		finalStem = stem
	}

	_, persistErr := record.Save(outPath.Dir, finalStem, savedFiles, generationErr)
	if persistErr == nil {
		return provModelPair, generationErr
	}

	if generationErr == nil {
		return provModelPair, persistErr
	}

	return provModelPair, errors.Join(generationErr, persistErr)
}

// cliRenderFlagEntry renders a flag with catalog support and default-model details. Flags without
// documentation produce no entry.
func (bild *bildApp) cliRenderFlagEntry(flag cli.Flag) string {
	flagDoc, ok := flag.(cli.DocGenerationFlag)
	if !ok {
		return ""
	}

	flagNames := flag.Names()
	if len(flagNames) == 0 {
		return output.FlagEntry(flagNames, "", flagDoc.GetUsage())
	}

	flagDetailText := flagDoc.GetUsage()

	// Only a generation parameter can have a support note: no model declares any other flag.
	if supportNote := output.FlagSupportNote(params.FlagType(flagNames[0]), bild.catalog.ModelDirectory()); supportNote != "" {
		flagDetailText += " " + fmt.Sprintf(SupportedBySentence, supportNote)
	}

	defaultText := ""
	if flagDoc.IsDefaultVisible() {
		defaultText = flagDoc.GetDefaultText()
	}

	if flagNames[0] == RunFlagModel {
		defaultText, _ = bild.defaultModelKey()
	}

	if defaultText != "" {
		flagDetailText += " " + fmt.Sprintf(FlagDefaultSuffix, defaultText)
	}

	valueHint := ""
	if flagDoc.TakesValue() {
		valueHint = output.FlagValueHint(flagNames[0], bild.catalog.Flags, runFlagHints)
	}

	return output.FlagEntry(flagNames, valueHint, flagDetailText)
}

// apiKey returns the provider's nonempty configured credential, falling back to its environment
// variable. If neither supplies a value, it returns a CredentialError naming both settings.
func (bild *bildApp) apiKey(description *catalog.Provider) (string, error) {
	if key := bild.apiKeys[description.ID]; key != "" {
		return key, nil
	}

	if key := os.Getenv(description.APIKeyEnvVar); key != "" {
		return key, nil
	}

	return "", &errs.CredentialError{EnvVar: description.APIKeyEnvVar, ProviderID: description.ID}
}

// applyAPIKeys stores configured credentials in the command and reports unknown providers.
func (bild *bildApp) applyAPIKeys(keys map[string]string) []string {
	bild.apiKeys = maps.Clone(keys)

	var unknownIDs []string

	for providerID := range keys {
		if _, loaded := bild.catalog.Provider(providerID); !loaded {
			unknownIDs = append(unknownIDs, providerID)
		}
	}

	slices.Sort(unknownIDs)

	return unknownIDs
}

// identifyInputMedia resolves media types and updates inputMedia in place when credentials exist.
// It may fetch remote sources. Without credentials, it returns nil without fetching so the later
// generation step can report the missing key.
func (bild *bildApp) identifyInputMedia(ctx context.Context, pair *catalog.ProvModelPair, inputMedia []media.Input) error {
	if _, credentialErr := bild.apiKey(&pair.Provider); credentialErr == nil {
		resolvedMedia, err := httpapi.ResolveInputMediaTypes(ctx, inputMedia)
		if err != nil {
			return err
		}

		copy(inputMedia, resolvedMedia)
	}

	return nil
}

// runGenerator assigns credentials and invokes the provider, recording its completion. Any progress
// animation stops before the result returns for persistence and reporting.
func (bild *bildApp) runGenerator(ctx context.Context, generator generation.Generator, run *generation.Generation) (generation.Result, error) {
	var spinner *terminal.Spinner
	if bild.invocation.animate {
		spinner = terminal.StartSpinner(ctx, bild.invocation.results, run.Model.Media)
	}

	result := generation.Result{Preparation: run.Clone()}

	apiKey, err := bild.apiKey(&run.Provider)
	if err == nil {
		run.APIKey = apiKey
		result, err = generator.Generate(ctx, run)
	}

	if run.Record != nil && run.Record.ProviderStatus == "" {
		run.Record.ProviderFinished(err)
	}

	if spinner != nil {
		elapsed := spinner.Finish()
		if err == nil {
			bild.invocation.generationElapsed = elapsed
		}
	}

	return result, err
}

// executeGeneration resolves any reuse selection, prepares provider inputs, and runs generation. It
// saves generated artifacts and returns their final stem and completed file records, including
// those saved before a later failure.
func (bild *bildApp) executeGeneration(ctx context.Context, generator generation.Generator, pair *catalog.ProvModelPair, flags *RunFlags, inputs params.FlagInputs, inputMedia []media.Input, outPath artifact.Location, pathChanges []params.Adjustment, record *metadata.Record) (finalStem string, completedFiles []output.SavedFile, resultErr error) {
	var reuse *metadata.Reuse

	if selection, supplied := flags.Reuse.ValIf(); supplied {
		var err error

		reuse, err = parseReuse(selection, pair.Provider.ID)
		if err != nil {
			return outPath.Stem, nil, err
		}
	}

	if !bild.invocation.jsonOutput {
		if err := output.PrintRunDetails(bild.invocation.results, pair.Provider.DisplayName, pair.Model.Name); err != nil {
			return outPath.Stem, nil, err
		}
	}

	if reuse == nil {
		if err := bild.identifyInputMedia(ctx, pair, inputMedia); err != nil {
			return outPath.Stem, nil, err
		}
	}

	preparedGeneration, adjustmentErr := adjustRequest(generator, pair, inputs, inputMedia, pathChanges, bild.catalog.Flags, bild.invocation.outcome, record, reuse)
	if err := errors.Join(adjustmentErr, bild.invocation.printAdjustments()); err != nil {
		return outPath.Stem, nil, err
	}

	run := generation.Generation{ProvModelPair: *pair, Preparation: preparedGeneration, Prompt: flags.Prompt.ValOr(""), Record: record}

	result, generationErr := bild.runGenerator(ctx, generator, &run)
	if generationErr == nil {
		defer cleanupGeneration(result.Artifacts, &resultErr)
	}

	recordErr := bild.applyFinalPreparation(&run, &result.Preparation)

	if err := errors.Join(generationErr, recordErr); err != nil {
		return outPath.Stem, nil, err
	}

	extension, err := landingExt(outPath, run.Params)
	if err != nil {
		return outPath.Stem, nil, err
	}

	finalStem, completedFiles, resultErr = writeRunResults(&result, outPath.Dir, outPath.Stem, extension, &run, bild.catalog.Flags, bild.invocation.outcome)
	cleanupGeneration(result.Artifacts, &resultErr)

	return finalStem, completedFiles, resultErr
}

// applyFinalPreparation replaces the run's preparation and updates its optional record. It appends
// newly reported changes to the invocation outcome, including after generation failure.
func (bild *bildApp) applyFinalPreparation(run *generation.Generation, preparedGeneration *generation.Preparation) error {
	earlyChangeCount := len(run.Changes)
	run.Preparation = *preparedGeneration
	recordErr := recordPreparation(&run.Preparation, run.Record)

	lateChanges := run.Changes
	if earlyChangeCount <= len(lateChanges) {
		lateChanges = lateChanges[earlyChangeCount:]
	}

	bild.invocation.outcome.Adjustments = append(bild.invocation.outcome.Adjustments, output.AdjustmentRecords(bild.catalog.Flags, lateChanges, runFlagNames)...)

	return recordErr
}

// tipsHelpText renders general help tips with examples from an available provider.
func (bild *bildApp) tipsHelpText() (string, error) {
	return output.HelpTips(bild.helpExampleProvider(), bild.invocation.styled)
}

// helpExampleProvider chooses a loaded example provider at random from the rotation. It falls back
// to the first provider with models, or an empty provider when none qualify.
func (bild *bildApp) helpExampleProvider() *catalog.Provider {
	var rotation []*catalog.Provider

	for i := range bild.catalog.Providers {
		prov := &bild.catalog.Providers[i]
		if slices.Contains(helpExampleProviderIDs, prov.ID) && len(prov.Models) > 0 {
			rotation = append(rotation, prov)
		}
	}

	if len(rotation) > 0 {
		// #nosec G404 -- the examples need variety, not unpredictability.
		return rotation[rand.IntN(len(rotation))]
	}

	for i := range bild.catalog.Providers {
		prov := &bild.catalog.Providers[i]
		if len(prov.Models) > 0 {
			return prov
		}
	}

	return &catalog.Provider{}
}

// selectedModels returns catalog models matching the image and video flags. It selects both media
// kinds when neither flag or both flags are set.
func (bild *bildApp) selectedModels(command *cli.Command) []catalog.ProvModelPair {
	imageSelected, videoSelected := inclusivePair(command.Bool(FilterFlagImage), command.Bool(FilterFlagVideo))

	return catalog.SelectMedia(bild.catalog.ModelDirectory(), imageSelected, videoSelected)
}

// printListing renders provider, model, media, alias, and JSON choices for list or search. It
// reports output failures on the invocation's diagnostic stream and returns their causes.
func (bild *bildApp) printListing(command *cli.Command, pairs []catalog.ProvModelPair) error {
	providersSelected, modelsSelected := inclusivePair(command.Bool(FilterFlagProviders), command.Bool(FilterFlagModels))
	page := output.ListingPage(pairs, modelsSelected || !providersSelected, command.Bool(ListingFlagAliases))

	return bild.invocation.reportOutputError(output.PrintListing(bild.invocation.results, page, providersSelected, modelsSelected, bild.invocation.jsonOutput), "", "")
}

// resolveModelInput returns the uniquely matching provider and model. For ambiguous input, it shows
// candidates unless JSON output is noninteractive, asks terminal users for a replacement, and
// retries. It returns resolution, prompt, or diagnostic-write errors when correction cannot
// continue.
func (bild *bildApp) resolveModelInput(modelInput string) (catalog.ProvModelPair, error) {
	for {
		pairs, err := bild.catalog.ResolveModelInput(modelInput)
		if err != nil {
			return catalog.ProvModelPair{}, err
		}

		if len(pairs) == 1 {
			return pairs[0], nil
		}

		if !bild.invocation.jsonOutput || bild.invocation.interactive {
			if err := output.PrintAmbiguity(bild.invocation.stderr, modelInput, pairs, bild.invocation.diagnosticStyled); err != nil {
				return catalog.ProvModelPair{}, errors.Join(fmt.Errorf("%q, %w", modelInput, errs.ErrModelResolveConflict), err)
			}
		}

		reply, asked, err := terminal.RepromptModel(bild.invocation.stdin, bild.invocation.stderr, bild.invocation.interactive, bild.invocation.diagnosticStyled)
		if err != nil {
			return catalog.ProvModelPair{}, err
		}

		if !asked {
			return catalog.ProvModelPair{}, fmt.Errorf("%q, %w", modelInput, errs.ErrModelResolveConflict)
		}

		if reply == "" {
			return catalog.ProvModelPair{}, fmt.Errorf("%w, %q, %w", errs.ErrCLINoModelInput, modelInput, errs.ErrModelResolveConflict)
		}

		modelInput = reply
	}
}

// showHelpPage renders a complete root or command page before delivering it.
func (bild *bildApp) showHelpPage(command *cli.Command) error {
	source := tmpl.HelpCommandText
	if command.Root() == command {
		source = tmpl.HelpMainText
	}

	return bild.invocation.reportOutputError(output.PrintTemplate(bild.invocation.results, command.Name, source, command, pageFuncs(bild)), "", "")
}

// cliRunHelp serves general or named command help and classifies unknown topics.
func (bild *bildApp) cliRunHelp(_ context.Context, command *cli.Command) error {
	if command.Args().Len() > 1 {
		return bild.invocation.fail(fmt.Errorf(ArgCountForm, command.Name, errs.ErrCLIOneArgMax, command.Args().Len()))
	}

	topic := command.Args().Get(0)
	if topic == "" {
		return bild.showHelpPage(command.Root())
	}

	if selected := command.Root().Command(topic); selected != nil {
		return bild.showHelpPage(selected)
	}

	return bild.invocation.fail(fmt.Errorf("%q, %w", topic, errs.ErrCLIUnknownHelpTopic))
}

// optionsHelpText renders visible flags using this application's loaded catalog.
func (bild *bildApp) optionsHelpText(command *cli.Command) string {
	flags := command.VisibleFlags()

	rendered := make([]string, 0, len(flags))
	for _, flag := range flags {
		rendered = append(rendered, bild.cliRenderFlagEntry(flag))
	}

	return strings.Join(rendered, "\n\n")
}

// cliClassifyUsageError reports parser failures using the output modes already parsed. For root
// invocations it opens any requested results file, reports the classified usage error, and closes
// the file while preserving output and parsing failures.
func (bild *bildApp) cliClassifyUsageError(_ context.Context, command *cli.Command, err error, isSubcommand bool) error {
	bild.invocation = newInvocation(command.Reader, command.Writer, command.ErrWriter)
	bild.invocation.selectMode(command)

	var destinationErr error

	if !isSubcommand {
		if openErr := bild.invocation.openResults(command.String(RunFlagSaveResults)); openErr != nil {
			destinationErr = bild.invocation.reportOutputError(openErr, "", "")
			bild.invocation.results = io.Discard
		}
	}

	classified := &errs.UsageError{Text: err.Error(), Cause: errors.Join(errs.ErrCLIFlagParse, err)}

	return bild.invocation.finish(errors.Join(destinationErr, bild.invocation.fail(classified)))
}
