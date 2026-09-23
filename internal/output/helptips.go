package output

import "github.com/shdeen/bildomat/internal/catalog"

// helpTipsPage contains the heading, provider ID, examples, and styling for general-help tips.
//   - Heading: the heading that opens the section
//   - ID: the catalog identifier of the provider the examples draw from
//   - Dim, Reset: ANSI codes for dimming the rule and restoring normal text
//   - Footer: the examples, drawn from the provider's models
type helpTipsPage struct {
	Heading string
	ID      string
	Dim     string
	Reset   string
	Footer  pageFooter
}

// HelpTips renders general-help examples drawn from the supplied provider's models.
func HelpTips(prov *catalog.Provider, styled bool) (string, error) {
	style := pageStyleValues(styled)

	return renderPage(pageHelpTips, helpTipsPage{Heading: HelpTipsHeading, ID: prov.ID, Dim: style.dim, Reset: style.reset, Footer: footerData(prov.ID, prov.Models)})
}
