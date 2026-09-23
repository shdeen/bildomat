package output

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/shdeen/bildomat/internal/errs"
	tmpl "github.com/shdeen/bildomat/internal/templates"
)

// pageName names the templates this package renders. Each name indexes one parsed template in
// pageTemplates.
//   - pageProviderInfo: the standard provider page
//   - pageProviderSummary: the aggregator summary
//   - pageModelInfo: the model card
//   - pageHelpTips: the tips section of the general help page
//   - pageListNested: the default listing of providers with their models
//   - pageListProviders: the providers-only listing
//   - pageListModels: the flat model directory
type pageName string

// The page names, one per template.
const (
	pageProviderInfo    pageName = "provider-info"
	pageProviderSummary pageName = "provider-summary"
	pageModelInfo       pageName = "model-info"
	pageHelpTips        pageName = "help-tips"
	pageListNested      pageName = "list-nested"
	pageListProviders   pageName = "list-providers"
	pageListModels      pageName = "list-models"
)

// pageSources maps each page name to the embedded template text it renders.
//
//nolint:gochecknoglobals // a read-only table, written only at package load.
var pageSources = map[pageName]string{
	pageProviderInfo:    tmpl.ProviderInfoText,
	pageProviderSummary: tmpl.ProviderSummaryText,
	pageModelInfo:       tmpl.ModelInfoText,
	pageHelpTips:        tmpl.HelpTipsText,
	pageListNested:      tmpl.ListNestedText,
	pageListProviders:   tmpl.ListProvidersText,
	pageListModels:      tmpl.ListModelsText,
}

// pageTemplates contains the embedded templates parsed at startup. pageParseErr records the first
// parse failure if any template is malformed.
//
//nolint:gochecknoglobals // parsed once at package load from the embedded pages, never written again.
var pageTemplates, pageParseErr = parsePages()

// parsePages parses every embedded page template and returns them by name, or the first parse
// failure.
func parsePages() (map[pageName]*template.Template, error) {
	parsed := map[pageName]*template.Template{}

	for name, source := range pageSources {
		pageTemplate, err := template.New(string(name)).Funcs(pageLayoutFuncs()).Parse(source + tmpl.APIKeyText + tmpl.ProviderHeaderText + tmpl.ExamplesText)
		if err != nil {
			return nil, fmt.Errorf("%q: %w, %w", name, errs.ErrOutputPageParse, err)
		}

		parsed[name] = pageTemplate
	}

	return parsed, nil
}

// The template-function names the page templates call. The command layer's help printer registers
// the first two beside its own.
//   - PageFuncIndent: the indentation function
//   - PageFuncWrap: the paragraph-wrapping function
//   - pageFuncPad: the column-padding function the info pages call
//   - pageFuncModelRow: a model's listing row, which the nested listing calls
//   - pageFuncModelListingLines: the fully qualified model listings, which the flat directory calls
const (
	PageFuncIndent            = "indent"
	PageFuncWrap              = "wrap"
	pageFuncPad               = "pad"
	pageFuncModelRow          = "modelRow"
	pageFuncModelListingLines = "modelListingLines"
)

// pageLayoutFuncs returns the formatting functions available to page templates.
func pageLayoutFuncs() template.FuncMap {
	return template.FuncMap{
		PageFuncIndent:            IndentText,
		PageFuncWrap:              WrapDetails,
		pageFuncPad:               padText,
		pageFuncModelRow:          modelRowText,
		pageFuncModelListingLines: modelListingLines,
	}
}

// renderPage renders the named template, returning an error for unknown pages or template parsing
// and execution failures.
func renderPage(name pageName, data any) (string, error) {
	if pageParseErr != nil {
		return "", pageParseErr
	}

	pageTemplate, known := pageTemplates[name]
	if !known {
		return "", fmt.Errorf("%q: %w", name, errs.ErrOutputPageUnknown)
	}

	var rendered strings.Builder
	if err := pageTemplate.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("%q: %w, %w", name, errs.ErrOutputPageExecute, err)
	}

	return rendered.String(), nil
}

// writePage renders the named page completely before writing it to destination. Parse, execution,
// and delivery failures reach the command unchanged.
func writePage(destination io.Writer, name pageName, data any) error {
	rendered, err := renderPage(name, data)
	if err != nil {
		return err
	}

	return WriteText(destination, "%s", rendered)
}

// PrintTemplate renders a command-supplied template with its required functions and data, and
// writes only a fully rendered page. Every failure is classified.
func PrintTemplate(destination io.Writer, name, source string, data any, functions template.FuncMap) error {
	page, err := template.New(name).Funcs(functions).Parse(source)
	if err != nil {
		return fmt.Errorf("%q: %w, %w", name, errs.ErrOutputPageParse, err)
	}

	var rendered strings.Builder
	if err := page.Execute(&rendered, data); err != nil {
		return fmt.Errorf("%q: %w, %w", name, errs.ErrOutputPageExecute, err)
	}

	return WriteText(destination, "%s", rendered.String())
}
