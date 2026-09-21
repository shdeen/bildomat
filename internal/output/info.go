package output

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
)

// File: internal/output/info.go
// The details pages the info command renders: the model card, the standard
// provider page, and the aggregator summary. Each assembles its page data,
// laid out in the pages' columns, and hands it to its template; none holds
// page copy. The JSON document of the same details is the reduced catalog
// (catalog.go), and the constraint sentences the pages share live in
// constraints.go.

// The pages' layout measures, counted from the left margin.
//   - infoContentWidth: the page column that wrapped and flowed content stays within
//   - infoLabelWidth: the identity block's label field on the provider pages
//   - infoCardLabelWidth: the identity block's label field on the model card
//   - infoFlagWidth: an option's flag field, so its details start at the column after it
//   - infoIDWidth: a roster model's ID field, so aliases start at the column after it
//   - infoIntroColumn: a group's intro line, above the details column
//   - infoVendorWidth: the summary's vendor field
//   - infoVendorCountWidth: the summary's right-aligned vendor count
const (
	infoContentWidth     = 80
	infoLabelWidth       = 10
	infoCardLabelWidth   = 16
	infoFlagWidth        = 32
	infoIDWidth          = 28
	infoIntroColumn      = infoFlagWidth - 8
	infoVendorWidth      = 19
	infoVendorCountWidth = 2
)

// The widths the columns leave for their content.
const (
	infoDetailsWidth = infoContentWidth - infoFlagWidth
	infoIntroWidth   = infoContentWidth - infoIntroColumn
)

// The quoting characters the pages use.
//   - singleQuote: the quote around a footer pattern carrying an escape, and around a string value in a change notice
//   - patternEscape: the backslash that marks a pattern as needing the quotes
const (
	singleQuote   = "'"
	patternEscape = "\\"
)

// itemJoiner is the separator and the space placed between the items of a list.
const itemJoiner = ListSeparator + " "

// pageStyle holds the styling codes an info page renders with, each empty off
// a terminal so the page renders plain.
//   - steel: the steel-blue accent, on option headings
//   - clay: the clay accent, on the required classification and on values
//   - dim: the dim shade, on the optional classification, column headers, and rules
//   - reset: the rendition reset
type pageStyle struct {
	steel string
	clay  string
	dim   string
	reset string
}

// apiKeySettings identifies the environment variable and the provider key under
// api-keys in the user's config file.
//   - ProviderID: the key under api-keys
//   - EnvVar: the alternative environment-variable name
type apiKeySettings struct {
	ProviderID string
	EnvVar     string
}

// modelCardPage carries the model card's data.
//   - PromptIgnoredNote: the prompt line's value for a model that ignores the prompt, else empty
//   - DocsURL: the model's documentation address, or the provider's when the model declares none
//   - Name: the model's published name
//   - ID: the model's bare ID
//   - MediaLabel: the medium the model generates, in the card's display form
//   - Key: the fully qualified provider/model key
//   - Aliases: the model's aliases, comma separated, empty where it declares none
//   - ProviderName, ProviderID: the provider's display name and catalog identifier
//   - Credentials: the environment variable and user-config key for the API key
//   - LabelWidth: the column the template pads the labels to
//   - Dim, Reset: the dim shade and the reset the template styles the column header and rule with
//   - Options: one option per declared parameter, each its lines from the page indent
type modelCardPage struct {
	Name              string
	ID                string
	MediaLabel        string
	Key               string
	Aliases           string
	PromptIgnoredNote string
	DocsURL           string
	ProviderName      string
	ProviderID        string
	Credentials       apiKeySettings
	LabelWidth        int
	Dim               string
	Reset             string
	Options           [][]string
}

// providerPage carries the standard provider page's data.
//   - Name: the provider's display name
//   - ID: the provider's catalog identifier
//   - DocsURL: the provider's documentation address, empty when it declares none
//   - Credentials: the environment variable and user-config key for the API key
//   - CatalogMedia: the model counts per medium
//   - ModelCount: the provider's model count
//   - LabelWidth, IDWidth: the columns the template pads to
//   - Dim, Reset: the dim shade and the reset the template styles the column headers and rules with
//   - Sections: one section per selected medium the provider declares a model for
type providerPage struct {
	Name         string
	ID           string
	DocsURL      string
	Credentials  apiKeySettings
	CatalogMedia string
	ModelCount   int
	LabelWidth   int
	IDWidth      int
	Dim          string
	Reset        string
	Sections     []providerSection
}

// vendorExample carries one summary search example naming a medium's top vendor.
//   - Filter: the dashed long name of the medium's filter flag
//   - Vendor: the medium's top vendor
type vendorExample struct {
	Filter string
	Vendor string
}

// providerSection carries one medium's part of the standard provider page.
//   - Media: the medium the section covers
//   - ModelCount: the section's model count
//   - HasAliases: whether any model of the section declares aliases
//   - Roster: the section's models, each its ID with its aliases, from the page indent
//   - Options: the section's options, each its lines from the page indent
type providerSection struct {
	Media      media.Kind
	ModelCount int
	HasAliases bool
	Roster     []string
	Options    [][]string
}

// summaryPage carries the aggregator summary's data.
//   - Name, ID, DocsURL, Credentials: as on the provider page
//   - LabelWidth: the columns the template pads to
//   - Dim, Reset: the dim shade and the reset the template styles the rules with
//   - CatalogMedia, ModelCount: as on the provider page
//   - Vendors: per selected medium, the vendor counts
//   - SharedFlags: per selected medium, the dashed flag names flowed into lines
//   - Footer: the footer's examples, drawn from the models the summary covers
type summaryPage struct {
	Name         string
	ID           string
	DocsURL      string
	Credentials  apiKeySettings
	CatalogMedia string
	ModelCount   int
	LabelWidth   int
	Dim          string
	Reset        string
	Vendors      []summaryMediaLines
	SharedFlags  []summaryMediaLines
	Footer       pageFooter
}

// summaryMediaLines carries one medium's lines on the summary.
//   - Media: the medium
//   - Lines: the lines
type summaryMediaLines struct {
	Media media.Kind
	Lines []string
}

// declarationGroup holds the models declaring one flag the same way.
//   - declaration: the declaration the models share
//   - modelIDs: the identifiers of the models sharing it
type declarationGroup struct {
	declaration params.Definition
	modelIDs    []string
}

// vendorCount holds one vendor's model count within a medium.
//   - vendor: the first slash-delimited token of the models' bare IDs
//   - count: the number of models
type vendorCount struct {
	vendor string
	count  int
}

// PrintModelInfo writes the model card with the requested terminal styling.
// Template and delivery failures are returned to the command.
func PrintModelInfo(destination io.Writer, provModelPair *catalog.ProvModelPair, paramFlags []params.Flag, styled bool) error {
	return writePage(destination, pageModelInfo, modelCardData(provModelPair, paramFlags, pageStyleValues(styled)))
}

// PrintProviderInfo writes the selected provider's detailed page or aggregator
// summary. It returns template and delivery failures to the command.
func PrintProviderInfo(destination io.Writer, prov *catalog.Provider, paramFlags []params.Flag, printImage, printVideo bool, mediaFilterFlags map[media.Kind]string, styled bool) error {
	if prov.Aggregator {
		return writePage(destination, pageProviderSummary, summaryData(prov, paramFlags, printImage, printVideo, pageStyleValues(styled), mediaFilterFlags))
	}

	return writePage(destination, pageProviderInfo, providerPageData(prov, paramFlags, printImage, printVideo, pageStyleValues(styled)))
}

// pageStyleValues returns the selected info-page colors, or empty strings for plain text.
func pageStyleValues(styled bool) pageStyle {
	if styled {
		return pageStyle{steel: ansiSteelBlue, clay: ansiClay, dim: ansiDim, reset: ansiReset}
	}

	return pageStyle{}
}

// modelCardData takes a provider-model pair, the parameter enumeration, and
// the page style, and returns the model card's page data.
func modelCardData(provModelPair *catalog.ProvModelPair, paramFlags []params.Flag, style pageStyle) modelCardPage {
	model := &provModelPair.Model
	page := modelCardPage{
		Name:         model.Name,
		ID:           model.ID,
		MediaLabel:   mediaLabels[model.Media],
		Key:          provModelPair.Provider.ID + catalog.KeySeparator + model.ID,
		Aliases:      strings.Join(model.Aliases, itemJoiner),
		DocsURL:      cmp.Or(model.DocsURL, provModelPair.Provider.DocsURL),
		ProviderName: provModelPair.Provider.DisplayName,
		ProviderID:   provModelPair.Provider.ID,
		Credentials:  apiKeySettings{ProviderID: provModelPair.Provider.ID, EnvVar: provModelPair.Provider.APIKeyEnvVar},
		LabelWidth:   infoCardLabelWidth,
		Dim:          style.dim,
		Reset:        style.reset,
	}

	if model.PromptIgnored {
		page.PromptIgnoredNote = CardPromptIgnored
	}

	for i := range paramFlags {
		paramFlag := &paramFlags[i]

		paramCfg, declared := model.Param(paramFlag.FlagID)
		if !declared {
			continue
		}

		details := slices.Concat(flagGuidanceLines(paramFlag, requirementSentence(&paramCfg, style), style), wrapSentences(constraintPhrases(paramFlag, &paramCfg, style)), wrapWords(paramCfg.ModelInfoComment, infoDetailsWidth))
		page.Options = append(page.Options, headedLines(optionHeading(paramFlag), details, style))
	}

	return page
}

// providerPageData takes a provider, the parameter enumeration, the media
// selections, and the page style, and returns the standard provider page's
// data.
func providerPageData(prov *catalog.Provider, paramFlags []params.Flag, printImage, printVideo bool, style pageStyle) providerPage {
	page := providerPage{
		Name:         prov.DisplayName,
		ID:           prov.ID,
		DocsURL:      prov.DocsURL,
		Credentials:  apiKeySettings{ProviderID: prov.ID, EnvVar: prov.APIKeyEnvVar},
		CatalogMedia: catalogMediaCounts(prov.Models),
		ModelCount:   len(prov.Models),
		LabelWidth:   infoLabelWidth,
		IDWidth:      infoIDWidth,
		Dim:          style.dim,
		Reset:        style.reset,
	}

	for _, media := range selectedMediaOrder(printImage, printVideo) {
		mediaModels := modelsOfMedia(prov.Models, media)
		if len(mediaModels) == 0 {
			continue
		}

		page.Sections = append(page.Sections, providerSectionData(media, mediaModels, paramFlags, style))
	}

	return page
}

// providerSectionData takes a medium, the provider's models of that medium,
// the parameter enumeration, and the page style, and returns the medium's
// section: the model roster and the options.
func providerSectionData(mediaKind media.Kind, models []catalog.Model, paramFlags []params.Flag, style pageStyle) providerSection {
	section := providerSection{Media: mediaKind, ModelCount: len(models)}

	for i := range models {
		model := &models[i]
		section.HasAliases = section.HasAliases || len(model.Aliases) > 0
		section.Roster = append(section.Roster, modelRosterText(model)...)
	}

	section.Options = sectionOptions(models, paramFlags, style)

	return section
}

// summaryData takes a provider, the parameter enumeration, the media
// selections, and the page style, and returns the aggregator summary's
// data.
func summaryData(prov *catalog.Provider, paramFlags []params.Flag, printImage, printVideo bool, style pageStyle, mediaFilterFlags map[media.Kind]string) summaryPage {
	page := summaryPage{
		Name:         prov.DisplayName,
		ID:           prov.ID,
		DocsURL:      prov.DocsURL,
		Credentials:  apiKeySettings{ProviderID: prov.ID, EnvVar: prov.APIKeyEnvVar},
		LabelWidth:   infoLabelWidth,
		Dim:          style.dim,
		Reset:        style.reset,
		CatalogMedia: catalogMediaCounts(prov.Models),
		ModelCount:   len(prov.Models),
	}

	var shown []catalog.Model

	for _, media := range selectedMediaOrder(printImage, printVideo) {
		mediaModels := modelsOfMedia(prov.Models, media)
		if len(mediaModels) == 0 {
			continue
		}

		shown = append(shown, mediaModels...)
		vendors := vendorCounts(mediaModels)

		vendorCounts := make([]string, 0, len(vendors))
		for _, vendor := range vendors {
			vendorCounts = append(vendorCounts, padText(vendor.vendor, infoVendorWidth)+fmt.Sprintf("%*d", infoVendorCountWidth, vendor.count))
		}

		page.Vendors = append(page.Vendors, summaryMediaLines{Media: media, Lines: vendorCounts})
		page.SharedFlags = append(page.SharedFlags, summaryMediaLines{Media: media, Lines: flowPhrases(sharedFlagNames(mediaModels, paramFlags), " ", infoContentWidth)})
		page.Footer.VendorExamples = append(page.Footer.VendorExamples, vendorExample{Filter: DashedFlagNames([]string{mediaFilterFlags[media]}, ""), Vendor: vendors[0].vendor})
	}

	vendorExamples := page.Footer.VendorExamples
	page.Footer = footerData(prov.ID, shown)
	page.Footer.VendorExamples = vendorExamples

	return page
}

// selectedMediaOrder takes the media selections and returns the selected
// media in page order, image before video.
func selectedMediaOrder(printImage, printVideo bool) []media.Kind {
	var selectedMedia []media.Kind

	if printImage {
		selectedMedia = append(selectedMedia, media.Image)
	}

	if printVideo {
		selectedMedia = append(selectedMedia, media.Video)
	}

	return selectedMedia
}

// catalogMediaCounts takes a provider's models and returns its model counts
// per medium, omitting a medium with no model.
func catalogMediaCounts(models []catalog.Model) string {
	var counts []string

	for _, media := range []media.Kind{media.Image, media.Video} {
		count := len(modelsOfMedia(models, media))
		switch {
		case count == 1:
			counts = append(counts, fmt.Sprintf(CatalogCountFormOne, count, string(media)))
		case count > 1:
			counts = append(counts, fmt.Sprintf(CatalogCountForm, count, string(media)))
		}
	}

	return strings.Join(counts, " "+CatalogSeparator+" ")
}

// modelRosterText takes a model and returns its roster text on the provider page: its
// bare ID in the ID column with its aliases beside it, flowed within the page
// width and continued at the alias column, or, for an ID filling the column,
// the ID alone with every alias line at the alias column.
func modelRosterText(model *catalog.Model) []string {
	if len(model.Aliases) == 0 {
		return []string{model.ID}
	}

	aliasLines := flowPhrases(model.Aliases, itemJoiner, infoContentWidth-infoIDWidth)

	modelText := []string{model.ID}
	if len(aliasLines) > 0 && textWidth(model.ID) < infoIDWidth {
		modelText[0] = padText(model.ID, infoIDWidth) + aliasLines[0]
		aliasLines = aliasLines[1:]
	}

	for _, aliasLine := range aliasLines {
		modelText = append(modelText, indentTo(infoIDWidth, aliasLine))
	}

	return modelText
}

// sectionOptions takes the models of one medium, the parameter enumeration,
// and the page style, and returns one option per flag at least one model
// declares, in the enumeration's order: a single option where the declaring
// models declare the flag alike, and otherwise the heading over the flag's
// description, comment, and examples, then one introduced group per
// declaration; either form ends with the scope note when models of the
// medium lack the flag.
func sectionOptions(models []catalog.Model, paramFlags []params.Flag, style pageStyle) [][]string {
	var options [][]string

	for i := range paramFlags {
		paramFlag := &paramFlags[i]

		groups := declarationGroups(models, paramFlag.FlagID)
		if len(groups) == 0 {
			continue
		}

		var optionText []string

		if len(groups) == 1 {
			details := slices.Concat(flagGuidanceLines(paramFlag, requirementSentence(&groups[0].declaration, style), style), wrapSentences(constraintPhrases(paramFlag, &groups[0].declaration, style)))
			optionText = headedLines(optionHeading(paramFlag), details, style)
		} else {
			optionText = headedLines(optionHeading(paramFlag), flagGuidanceLines(paramFlag, "", style), style)
			for _, group := range introducedGroups(groups) {
				optionText = append(optionText, groupLines(group, paramFlag, style)...)
			}
		}

		if note := scopeNote(models, groups); note != "" {
			optionText = append(optionText, "")
			for _, noteLine := range wrapWords(note, infoDetailsWidth) {
				optionText = append(optionText, indentTo(infoFlagWidth, noteLine))
			}
		}

		options = append(options, optionText)
	}

	return options
}

// introducedGroup pairs a declaration group with the intro line that opens
// it on the page.
//   - group: the declaration group
//   - intro: the line introducing it: the remainder form, or the form naming its models
type introducedGroup struct {
	group *declarationGroup
	intro string
}

// introducedGroups takes the declaration groups of one flag and returns them
// with their intros in page order: the group holding more models than any
// other first, in the remainder form, then the rest in their given order,
// each naming its models; groups tied for the largest are all named.
func introducedGroups(groups []declarationGroup) []introducedGroup {
	leading := leadingGroupIndex(groups)
	ordered := make([]introducedGroup, 0, len(groups))

	if leading >= 0 {
		ordered = append(ordered, introducedGroup{group: &groups[leading], intro: GroupIntroRemainder + ":"})
	}

	for i := range groups {
		if i != leading {
			ordered = append(ordered, introducedGroup{group: &groups[i], intro: fmt.Sprintf(GroupIntro, joinNames(groups[i].modelIDs, ListPairAnd, ListLastAnd)) + ":"})
		}
	}

	return ordered
}

// leadingGroupIndex takes declaration groups and returns the index of the
// one holding more models than any other, or -1 when the largest groups tie.
func leadingGroupIndex(groups []declarationGroup) int {
	leading := -1
	tied := false

	for i := range groups {
		switch {
		case leading < 0 || len(groups[i].modelIDs) > len(groups[leading].modelIDs):
			leading, tied = i, false
		case len(groups[i].modelIDs) == len(groups[leading].modelIDs):
			tied = true
		}
	}

	if tied {
		return -1
	}

	return leading
}

// scopeNote takes the models of a medium and the declaration groups of one
// flag, and returns the sentence naming the shorter list when not every
// model declares the flag: the models not declaring it when they are fewer
// than the declaring ones, otherwise the models declaring it; an empty
// string when every model declares it.
func scopeNote(models []catalog.Model, groups []declarationGroup) string {
	var declaring, missing []string

	for i := range models {
		declared := slices.ContainsFunc(groups, func(group declarationGroup) bool { return slices.Contains(group.modelIDs, models[i].ID) })
		if declared {
			declaring = append(declaring, models[i].ID)
		} else {
			missing = append(missing, models[i].ID)
		}
	}

	switch {
	case len(missing) == 0:
		return ""
	case len(missing) < len(declaring):
		return fmt.Sprintf(NotUsedBy, joinNames(missing, ListPairOr, ListLastOr))
	}

	return fmt.Sprintf(OnlyForModels, joinNames(declaring, ListPairAnd, ListLastAnd))
}

// joinNames takes names and the catalog's pair and last forms, and returns
// the names as an English list: one alone, two in the pair form, and more
// separated up to the last two, which take the last form.
func joinNames(names []string, pairForm, lastForm string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return fmt.Sprintf(pairForm, names[0], names[1])
	}

	return fmt.Sprintf(lastForm, strings.Join(names[:len(names)-1], itemJoiner), names[len(names)-1])
}

// declarationGroups takes models and a flag and returns one group per
// distinct declaration of the flag among the models declaring it, in the
// models' first-occurrence order; no group when no model declares it.
func declarationGroups(models []catalog.Model, flagID params.FlagType) []declarationGroup {
	var groups []declarationGroup

	for i := range models {
		model := &models[i]

		paramCfg, declared := model.Param(flagID)
		if !declared {
			continue
		}

		grouped := false

		for groupIndex := range groups {
			if sameDeclaration(&groups[groupIndex].declaration, &paramCfg) {
				groups[groupIndex].modelIDs = append(groups[groupIndex].modelIDs, model.ID)
				grouped = true

				break
			}
		}

		if !grouped {
			groups = append(groups, declarationGroup{declaration: paramCfg, modelIDs: []string{model.ID}})
		}
	}

	return groups
}

// sameDeclaration takes two parameter declarations and reports whether they
// state the same constraints: allowed values, range bounds, repeat maximum,
// size bounds, rule description, and requirement. A model's expanded
// comment does not count.
func sameDeclaration(first, second *params.Definition) bool {
	firstMin, firstHasMin := first.MinValue.ValIf()
	secondMin, secondHasMin := second.MinValue.ValIf()
	firstMax, firstHasMax := first.MaxValue.ValIf()
	secondMax, secondHasMax := second.MaxValue.ValIf()

	return slices.Equal(first.AllowedValues, second.AllowedValues) &&
		firstHasMin == secondHasMin && firstMin == secondMin &&
		firstHasMax == secondHasMax && firstMax == secondMax &&
		first.MaxMultiple == second.MaxMultiple &&
		sameSizeBounds(first.CustomSize, second.CustomSize) &&
		first.RuleDescription == second.RuleDescription &&
		first.Required == second.Required
}

// sameSizeBounds takes two optional size bounds and reports whether both are
// absent or both state the same bounds.
func sameSizeBounds(first, second *params.SizeBounds) bool {
	if first == nil || second == nil {
		return first == second
	}

	return *first == *second
}

// optionHeading takes a flag record and returns the option heading: the flag's
// dashed short and long forms with its value hint, as the help page heads
// the flag.
func optionHeading(paramFlag *params.Flag) string {
	return DashedFlagNames(append([]string{string(paramFlag.FlagID)}, paramFlag.Aliases...), paramFlag.TextHint)
}

// headedLines takes an option heading, the detail lines beneath it, and the
// page style, and returns the option's lines: the heading, in the steel-blue
// accent, in the flag field with the first detail beside it, or alone when
// the heading fills the field, then the remaining details at the details
// column.
func headedLines(heading string, details []string, style pageStyle) []string {
	styledHeading := style.steel + heading + style.reset

	optionText := make([]string, 0, 1+len(details))
	if textWidth(heading) >= infoFlagWidth || len(details) == 0 {
		optionText = append(optionText, styledHeading)
		for _, detail := range details {
			optionText = append(optionText, indentTo(infoFlagWidth, detail))
		}

		return optionText
	}

	optionText = append(optionText, styledHeading+strings.Repeat(" ", infoFlagWidth-textWidth(heading))+details[0])
	for _, detail := range details[1:] {
		optionText = append(optionText, indentTo(infoFlagWidth, detail))
	}

	return optionText
}

// groupLines takes an introduced group, its flag record, and the page
// style, and returns the group's lines: a blank line, the intro wrapped
// from the intro column, then the group's classification sentence opening
// its first constraint sentence and the remaining constraint sentences at
// the details column.
func groupLines(introduced introducedGroup, paramFlag *params.Flag, style pageStyle) []string {
	introLines := wrapWords(introduced.intro, infoIntroWidth)

	sentences := constraintPhrases(paramFlag, &introduced.group.declaration, style)
	if len(sentences) == 0 {
		sentences = []string{strings.TrimSpace(requirementSentence(&introduced.group.declaration, style) + " " + NoAdditionalLimits)}
	} else {
		sentences[0] = requirementSentence(&introduced.group.declaration, style) + " " + sentences[0]
	}

	detailLines := wrapSentences(sentences)

	groupText := make([]string, 0, 1+len(introLines)+len(detailLines))
	groupText = append(groupText, "")

	for _, introLine := range introLines {
		groupText = append(groupText, indentTo(infoIntroColumn, introLine))
	}

	for _, detail := range detailLines {
		groupText = append(groupText, indentTo(infoFlagWidth, detail))
	}

	return groupText
}

// flagGuidanceLines takes a flag record, the sentence opening its
// description (empty for none), and the page style, and returns the record's
// help-page guidance wrapped within the details column: the opened
// description, the comment where declared, and the examples sentence where
// the record declares example values, each example in the value accent.
func flagGuidanceLines(paramFlag *params.Flag, opening string, style pageStyle) []string {
	sentences := []string{strings.TrimSpace(opening + " " + paramFlag.Description), paramFlag.Comment}
	if len(paramFlag.ExampleValues) > 0 {
		sentences = append(sentences, fmt.Sprintf(ExamplesSentence, accentValues(paramFlag.ExampleValues, itemJoiner, style)))
	}

	return wrapSentences(sentences)
}

// requirementSentence takes a declaration and the page style and returns the
// Required classification as a sentence, in the clay accent with the sentence
// end, for a required parameter. An optional parameter is the default and
// carries no classification, so the sentence is empty.
func requirementSentence(paramCfg *params.Definition, style pageStyle) string {
	if paramCfg.Required {
		return style.clay + RequiredLabel + style.reset + "."
	}

	return ""
}

// wrapSentences takes sentences and returns them wrapped within the details
// column, each sentence starting a line; an empty sentence yields no line.
func wrapSentences(sentences []string) []string {
	lines := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		lines = append(lines, wrapWords(sentence, infoDetailsWidth)...)
	}

	return lines
}

// accentValue takes one value and the page style and returns the value in
// the style's value accent.
func accentValue(value string, style pageStyle) string {
	return style.clay + value + style.reset
}

// accentValues takes values, their separator, and the page style, and returns
// the values joined by the separator, each in the value accent and the
// separator plain.
func accentValues(values []string, separator string, style pageStyle) string {
	accented := make([]string, 0, len(values))
	for _, value := range values {
		accented = append(accented, accentValue(value, style))
	}

	return strings.Join(accented, separator)
}

// vendorCounts takes the models of one medium and returns each vendor with
// its model count, ordered by count descending and then vendor ascending. A
// model's vendor is the first slash-delimited token of its bare ID.
func vendorCounts(models []catalog.Model) []vendorCount {
	counts := map[string]int{}

	for i := range models {
		vendor, _, _ := strings.Cut(models[i].ID, catalog.KeySeparator)
		counts[vendor]++
	}

	vendors := make([]vendorCount, 0, len(counts))
	for vendor, count := range counts {
		vendors = append(vendors, vendorCount{vendor: vendor, count: count})
	}

	slices.SortFunc(vendors, compareVendorCounts)

	return vendors
}

// compareVendorCounts orders two vendor counts by count descending and then
// vendor ascending, for the summary's vendor list.
func compareVendorCounts(first, second vendorCount) int {
	if first.count != second.count {
		return second.count - first.count
	}

	return strings.Compare(first.vendor, second.vendor)
}

// sharedFlagNames takes the models of one medium and the parameter
// enumeration, and returns the dashed long name of every flag any of the
// models declares, in the enumeration's order.
func sharedFlagNames(models []catalog.Model, paramFlags []params.Flag) []string {
	var names []string

	for i := range paramFlags {
		for modelIndex := range models {
			if models[modelIndex].SupportsParam(paramFlags[i].FlagID) {
				names = append(names, dashedFlagName(string(paramFlags[i].FlagID)))

				break
			}
		}
	}

	return names
}
