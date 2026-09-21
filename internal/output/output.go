// Package output renders terminal messages, command results, and sidecars.
package output

import (
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shdeen/bildomat/internal/media"
)

// PrintRunDetails writes the provider and model headers and returns any delivery
// failure. The prompt remains in the request, JSON outcome, and thoughts sidecar.
func PrintRunDetails(destination io.Writer, providerDisplayName, modelName string) error {
	return WriteText(destination, ProviderHeader+"\n"+ModelHeader+"\n", providerName(providerDisplayName), modelName)
}

// joinQuotedMediaSources takes input media and returns their quoted sources separated by commas.
func joinQuotedMediaSources(inputMedia []media.Input) string {
	quotedPaths := make([]string, 0, len(inputMedia))
	for mediaIndex := range inputMedia {
		quotedPaths = append(quotedPaths, strconv.Quote(inputMedia[mediaIndex].Source()))
	}

	return strings.Join(quotedPaths, ", ")
}

// oneNotice takes text and returns it unchanged unless it contains a control or replacement
// rune, in which case it returns a quoted form.
func oneNotice(text string) string {
	if needsEscaping(text) {
		return strconv.Quote(text)
	}

	return text
}

// needsEscaping takes text and reports whether it contains a control rune or replacement rune.
func needsEscaping(text string) bool {
	for _, r := range text {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return true
		}
	}

	return false
}

// providerDisplay takes a provider display name and returns it, or the
// in-sentence fallback when the name is empty.
func providerDisplay(providerDisplayName string) string {
	if providerDisplayName == "" {
		return ProviderFallbackInSentence
	}

	return providerDisplayName
}

// providerName takes a provider display name and returns it, or the name-form
// fallback when the name is empty.
func providerName(providerDisplayName string) string {
	if providerDisplayName == "" {
		return ProviderFallbackAsName
	}

	return providerDisplayName
}
