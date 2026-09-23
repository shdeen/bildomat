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

// The pages' layout measures, counted from the left margin.
//   - infoLabelWidth: the identity block's label field on the provider pages
//   - infoCardLabelWidth: the identity block's label field on the model card
//   - infoFlagWidth: an option's flag field, so its details start at the column after it
//   - infoIDWidth: a roster model's ID field, so aliases start at the column after it
//   - infoIntroColumn: the indentation of a model-group introduction
//   - infoVendorWidth: the summary's vendor field
//   - infoVendorCountWidth: the summary's right-aligned vendor count
const (
	infoLabelWidth       = 10
	infoCardLabelWidth   = 16
	infoFlagWidth        = 32
	infoIDWidth          = 28
	infoIntroColumn      = infoFlagWidth - 8
	infoVendorWidth      = 19
	infoVendorCountWidth = 2
)

// The widths available after the option labels.
//   - infoDetailsWidth: the space for parameter descriptions
//   - infoIntroWidth: the space for model-group introductions
const (
	infoDetailsWidth = pageWidth - infoFlagWidth
	infoIntroWidth   = pageWidth - infoIntroColumn
)

// The quoting characters the pages use.
//   - singleQuote: the quote around a footer pattern carrying an escape, and around a string value
//     in a change notice
//   - patternEscape: the backslash that marks a pattern as needing the quotes
const (
	singleQuote   = "'"
	patternEscape = "\\"
)

// itemJoiner is the separator and the space placed between the items of a list.
const itemJoiner = ListSeparator + " "

// pageStyle contains the ANSI codes used by an info page. Empty codes produce plain text.
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

// apiKeySettings identifies the environment variable and the provider key under api-keys in the
// user's config file.
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
//   - Dim, Reset: ANSI codes for dimmed column headers and rules, and for restoring normal text
//   - Options: rendered lines for each declared parameter option
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

// providerHeader contains the identity, credential guidance, and catalog totals shared by detailed
// provider pages and aggregator summaries.
//   - Name, ID: the published name and catalog identifier
//   - DocsURL: the provider documentation URL
//   - Credentials: the supported credential settings
//   - CatalogMedia: the generated media labels
//   - ModelCount: the number of models before display filtering
//   - LabelWidth: the padded width of identity labels
//   - Aggregator: whether to describe models by vendor
//   - Dim, Reset: the terminal styling sequences, empty for plain output
type providerHeader struct {
	Name         string
	ID           string
	DocsURL      string
	Credentials  apiKeySettings
	CatalogMedia string
	ModelCount   int
	LabelWidth   int
	Aggregator   bool
	Dim          string
	Reset        string
}

// providerPage contains a common header and the selected model and option sections.
//   - providerHeader: the shared identity and catalog totals
//   - IDWidth: the model identifier column width
//   - Sections: the selected models and options grouped by medium
type providerPage struct {
	providerHeader

	IDWidth  int
	Sections []providerSection
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
//   - Roster: rendered model IDs and aliases for the section
//   - Options: rendered lines for each option in the section
type providerSection struct {
	Media      media.Kind
	ModelCount int
	HasAliases bool
	Roster     []string
	Options    [][]string
}

// summaryPage contains a common provider header and the selected aggregator summary.
//   - providerHeader: the shared identity and catalog totals
//   - Vendors: the vendor counts grouped by medium
//   - SharedFlags: flags declared by at least one selected model, grouped by medium
//   - Footer: the model and search examples
type summaryPage struct {
	providerHeader

	Vendors     []summaryMediaLines
	SharedFlags []summaryMediaLines
	Footer      pageFooter
}

// summaryMediaLines carries one medium's lines on the summary.
//   - Media: the medium
//   - Lines: the rendered vendor counts or shared parameter labels
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

// PrintModelInfo writes the model card with the requested terminal styling. Template and delivery
// failures are returned to the command.
func PrintModelInfo(destination io.Writer, provModelPair *catalog.ProvModelPair, paramFlags []params.Flag, styled bool) error {
	return writePage(destination, pageModelInfo, modelCardData(provModelPair, paramFlags, pageStyleValues(styled)))
}

// PrintProviderInfo writes the selected provider's detailed page or aggregator summary. It returns
// template and delivery failures to the command.
func PrintProviderInfo(destination io.Writer, prov *catalog.Provider, pairs []catalog.ProvModelPair, paramFlags []params.Flag, mediaFilterFlags map[media.Kind]string, styled bool) error {
	models := make([]catalog.Model, 0, len(pairs))
	for index := range pairs {
		models = append(models, pairs[index].Model)
	}

	if prov.Aggregator {
		return writePage(destination, pageProviderSummary, summaryData(prov, models, paramFlags, pageStyleValues(styled), mediaFilterFlags))
	}

	return writePage(destination, pageProviderInfo, providerPageData(prov, models, paramFlags, pageStyleValues(styled)))
}

// pageStyleValues returns the selected info-page colors, or empty strings for plain text.
func pageStyleValues(styled bool) pageStyle {
	if styled {
		return pageStyle{steel: ansiSteelBlue, clay: ansiClay, dim: ansiDim, reset: ansiReset}
	}

	return pageStyle{}
}

// modelCardData prepares a model card with its identity and declared parameter details.
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

// providerPageData prepares the standard provider page from its selected models, parameter
// declarations, and page style.
func providerPageData(prov *catalog.Provider, models []catalog.Model, paramFlags []params.Flag, style pageStyle) providerPage {
	page := providerPage{providerHeader: providerHeaderData(prov, style), IDWidth: infoIDWidth}

	for _, media := range []media.Kind{media.Image, media.Video} {
		mediaModels := modelsOfMedia(models, media)
		if len(mediaModels) == 0 {
			continue
		}

		page.Sections = append(page.Sections, providerSectionData(media, mediaModels, paramFlags, style))
	}

	return page
}

// providerSectionData prepares one medium's model roster and option descriptions.
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

// summaryData prepares the aggregator summary from its selected models, parameter declarations,
// page style, and command filter names.
func summaryData(prov *catalog.Provider, models []catalog.Model, paramFlags []params.Flag, style pageStyle, mediaFilterFlags map[media.Kind]string) summaryPage {
	page := summaryPage{providerHeader: providerHeaderData(prov, style)}

	var vendorExamples []vendorExample

	var shown []catalog.Model

	for _, media := range []media.Kind{media.Image, media.Video} {
		mediaModels := modelsOfMedia(models, media)
		if len(mediaModels) == 0 {
			continue
		}

		shown = append(shown, mediaModels...)
		vendors := vendorCounts(mediaModels)

		vendorLines := make([]string, 0, len(vendors))
		for _, vendor := range vendors {
			vendorLines = append(vendorLines, padText(vendor.vendor, infoVendorWidth)+fmt.Sprintf("%*d", infoVendorCountWidth, vendor.count))
		}

		page.Vendors = append(page.Vendors, summaryMediaLines{Media: media, Lines: vendorLines})
		page.SharedFlags = append(page.SharedFlags, summaryMediaLines{Media: media, Lines: flowPhrases(sharedFlagNames(mediaModels, paramFlags), " ", pageWidth)})
		vendorExamples = append(vendorExamples, vendorExample{Filter: dashedFlagNames([]string{mediaFilterFlags[media]}, ""), Vendor: vendors[0].vendor})
	}

	page.Footer = footerData(prov.ID, shown)
	page.Footer.VendorExamples = vendorExamples

	return page
}

// providerHeaderData prepares the common identity and full-catalog totals.
func providerHeaderData(prov *catalog.Provider, style pageStyle) providerHeader {
	return providerHeader{
		Name: prov.DisplayName, ID: prov.ID, DocsURL: prov.DocsURL,
		Credentials:  apiKeySettings{ProviderID: prov.ID, EnvVar: prov.APIKeyEnvVar},
		CatalogMedia: catalogMediaCounts(prov.Models), ModelCount: len(prov.Models),
		LabelWidth: infoLabelWidth, Aggregator: prov.Aggregator,
		Dim: style.dim, Reset: style.reset,
	}
}

// catalogMediaCounts formats the number of image and video models, omitting any medium with no
// models.
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

// modelRosterText formats a model ID and its aliases within the roster columns. Aliases continue on
// indented lines when they exceed the available width.
func modelRosterText(model *catalog.Model) []string {
	if len(model.Aliases) == 0 {
		return []string{model.ID}
	}

	aliasLines := flowPhrases(model.Aliases, itemJoiner, pageWidth-infoIDWidth)

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

// sectionOptions returns options in flag-record order and groups differing model constraints. When
// some models lack an option, its scope note names either the supporting models or the models that
// lack it, whichever list is shorter.
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

// introducedGroup pairs a declaration group with the intro line that opens it on the page.
//   - group: the declaration group
//   - intro: the line introducing it: the remainder form, or the form naming its models
type introducedGroup struct {
	group *declarationGroup
	intro string
}

// introducedGroups places a uniquely largest model group first with a remainder heading. Other
// groups retain their order and name their models; tied largest groups are also named.
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

// leadingGroupIndex returns the uniquely largest group's index, or -1 for a tie or no groups.
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

// scopeNote describes partial flag support using the shorter model list. Ties name supporting
// models; universal support needs no note.
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

// joinNames formats names as an English list using the supplied two-name and final-name forms.
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

// declarationGroups groups models with identical flag constraints in first-occurrence order.
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

// sameDeclaration compares display constraints, requirements, and rule descriptions. Provider
// parameter names and model-specific comments do not affect equality.
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

// sameSizeBounds reports whether two optional size constraints are both absent or equal.
func sameSizeBounds(first, second *params.SizeBounds) bool {
	if first == nil || second == nil {
		return first == second
	}

	return *first == *second
}

// optionHeading formats a parameter's short and long flags with its value hint.
func optionHeading(paramFlag *params.Flag) string {
	return dashedFlagNames(append([]string{string(paramFlag.FlagID)}, paramFlag.Aliases...), paramFlag.TextHint)
}

// headedLines aligns option details beside a styled heading. A heading that fills the column stands
// alone above its details.
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

// groupLines formats a model group's introduction and constraint details after a blank line.
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

// flagGuidanceLines wraps a flag's description, optional comment, and example values. A nonempty
// opening precedes the description.
func flagGuidanceLines(paramFlag *params.Flag, opening string, style pageStyle) []string {
	sentences := []string{strings.TrimSpace(opening + " " + paramFlag.Description), paramFlag.Comment}
	if len(paramFlag.ExampleValues) > 0 {
		sentences = append(sentences, fmt.Sprintf(ExamplesSentence, accentValues(paramFlag.ExampleValues, itemJoiner, style)))
	}

	return wrapSentences(sentences)
}

// requirementSentence returns the styled requirement label, or empty text for an optional
// parameter.
func requirementSentence(paramCfg *params.Definition, style pageStyle) string {
	if paramCfg.Required {
		return style.clay + RequiredLabel + style.reset + "."
	}

	return ""
}

// wrapSentences wraps each nonempty sentence separately within the details column.
func wrapSentences(sentences []string) []string {
	lines := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		lines = append(lines, wrapWords(sentence, infoDetailsWidth)...)
	}

	return lines
}

// accentValue surrounds a value with the selected accent and reset codes.
func accentValue(value string, style pageStyle) string {
	return style.clay + value + style.reset
}

// accentValues joins accented values with an unstyled separator.
func accentValues(values []string, separator string, style pageStyle) string {
	accented := make([]string, 0, len(values))
	for _, value := range values {
		accented = append(accented, accentValue(value, style))
	}

	return strings.Join(accented, separator)
}

// vendorCounts counts models by the first slash-delimited part of their IDs. Results sort by
// descending count, then ascending vendor name.
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

// compareVendorCounts orders vendors by descending model count, then ascending name.
func compareVendorCounts(first, second vendorCount) int {
	if first.count != second.count {
		return second.count - first.count
	}

	return strings.Compare(first.vendor, second.vendor)
}

// sharedFlagNames returns each flag declared by at least one model, in flag declaration order.
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
