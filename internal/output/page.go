package output

// File: internal/output/page.go
// The template machinery every page in this package renders through: the
// parsed page templates, the layout functions they call, and the one execution
// path that writes a rendered page. Page copy lives in internal/templates, not
// here, so a wording change never reaches rendering code.

import (
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/shdeen/bildomat/internal/errs"
	tmpl "github.com/shdeen/bildomat/internal/templates"
)

// Page names the templates this package renders. Each name indexes one parsed
// template in pageTemplates.
//   - pageProviderInfo: the standard provider page
//   - pageProviderSummary: the aggregator summary
//   - pageModelInfo: the model card
//   - pageHelpTips: the tips section of the general help page
//   - pageListNested: the default listing of providers with their models
//   - pageListProviders: the providers-only listing
//   - pageListModels: the flat model directory
type Page string

// The page names, one per template.
const (
	pageProviderInfo    Page = "provider-info"
	pageProviderSummary Page = "provider-summary"
	pageModelInfo       Page = "model-info"
	pageHelpTips        Page = "help-tips"
	pageListNested      Page = "list-nested"
	pageListProviders   Page = "list-providers"
	pageListModels      Page = "list-models"
)

// pageSources maps each page name to the embedded template text it renders.
//
//nolint:gochecknoglobals // a read-only table, written only at package load.
var pageSources = map[Page]string{
	pageProviderInfo:    tmpl.ProviderInfoText,
	pageProviderSummary: tmpl.ProviderSummaryText,
	pageModelInfo:       tmpl.ModelInfoText,
	pageHelpTips:        tmpl.HelpTipsText,
	pageListNested:      tmpl.ListNestedText,
	pageListProviders:   tmpl.ListProvidersText,
	pageListModels:      tmpl.ListModelsText,
}

// pageTemplates holds every page template, parsed once at startup, and
// pageParseErr holds the first parse failure where the embedded copy is
// malformed.
//
//nolint:gochecknoglobals // parsed once at package load from the embedded pages, never written again.
var pageTemplates, pageParseErr = parsePages()

// parsePages parses every embedded page template and returns them by name, or
// the first parse failure.
func parsePages() (map[Page]*template.Template, error) {
	parsed := map[Page]*template.Template{}

	for name, source := range pageSources {
		pageTemplate, err := template.New(string(name)).Funcs(pageLayoutFuncs()).Parse(source + tmpl.APIKeyText)
		if err != nil {
			return nil, fmt.Errorf("%q: %w, %w", name, errs.ErrOutputPageParse, err)
		}

		parsed[name] = pageTemplate
	}

	return parsed, nil
}

// The template-function names the page templates call. The command layer's
// help printer registers the first two beside its own.
//   - PageFuncIndent: the indentation function
//   - PageFuncWrap: the paragraph-wrapping function
//   - pageFuncPad: the column-padding function the info pages call
//   - pageFuncMediaModels: the models of one medium, which the nested listing calls
//   - pageFuncModelRow: a model's listing row, which the nested listing calls
//   - pageFuncModelListingLines: the fully qualified model listings, which the flat directory calls
const (
	PageFuncIndent            = "indent"
	PageFuncWrap              = "wrap"
	pageFuncPad               = "pad"
	pageFuncMediaModels       = "mediaModels"
	pageFuncModelRow          = "modelRow"
	pageFuncModelListingLines = "modelListingLines"
)

// pageLayoutFuncs returns the functions the page templates call, so a
// template controls its own indentation, wrapping, column padding, and the
// selection and the models it lists.
func pageLayoutFuncs() template.FuncMap {
	return template.FuncMap{
		PageFuncIndent:            IndentText,
		PageFuncWrap:              WrapDetails,
		pageFuncPad:               padText,
		pageFuncMediaModels:       modelsOfMedia,
		pageFuncModelRow:          modelNotice,
		pageFuncModelListingLines: modelListingLines,
	}
}

// renderPage takes a page name and its data, and returns the rendered page. It
// returns the template failure where the embedded copy is malformed.
func renderPage(name Page, data any) (string, error) {
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

// writePage renders the named page completely before writing it to destination.
// Parse, execution, and delivery failures reach the command unchanged.
func writePage(destination io.Writer, name Page, data any) error {
	rendered, err := renderPage(name, data)
	if err != nil {
		return err
	}

	return WriteText(destination, "%s", rendered)
}

// PrintTemplate renders a command-supplied template with its required functions
// and data, and writes only a fully rendered page. Every failure is classified.
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
