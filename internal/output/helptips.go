package output

import "github.com/shdeen/bildomat/internal/catalog"

// File: internal/output/helptips.go
// The tips section of the general help page: the fully qualified model form
// and the info and search examples, drawn from one provider's catalog by the
// same example logic the aggregator summary's footer uses (examples.go).

// helpTipsPage carries the data the tips section of the general help page
// names.
//   - Heading: the heading that opens the section
//   - ID: the catalog identifier of the provider the examples draw from
//   - Dim, Reset: the dim shade and the reset the template styles the rule with
//   - Footer: the examples, drawn from the provider's models
type helpTipsPage struct {
	Heading string
	ID      string
	Dim     string
	Reset   string
	Footer  pageFooter
}

// HelpTips takes the provider whose models the examples draw from and returns
// the tips section of the general help page: the fully qualified model form
// and the info and search examples. It returns the template failure where the
// embedded copy is malformed.
func HelpTips(prov *catalog.Provider, styled bool) (string, error) {
	style := pageStyleValues(styled)

	return renderPage(pageHelpTips, helpTipsPage{Heading: HelpTipsHeading, ID: prov.ID, Dim: style.dim, Reset: style.reset, Footer: footerData(prov.ID, prov.Models)})
}
