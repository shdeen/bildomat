// Package output renders terminal messages, command results, and sidecars.
package output

import (
	"cmp"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shdeen/bildomat/internal/media"
)

// PrintRunDetails writes the provider and model headers to destination.
func PrintRunDetails(destination io.Writer, providerDisplayName, modelName string) error {
	return WriteText(destination, ProviderHeader+"\n"+ModelHeader+"\n", cmp.Or(providerDisplayName, ProviderFallbackAsName), modelName)
}

// joinQuotedMediaSources returns quoted input sources separated by commas.
func joinQuotedMediaSources(inputMedia []media.Input) string {
	quotedPaths := make([]string, 0, len(inputMedia))
	for mediaIndex := range inputMedia {
		quotedPaths = append(quotedPaths, strconv.Quote(inputMedia[mediaIndex].Source()))
	}

	return strings.Join(quotedPaths, ", ")
}

// escapeForTerminal quotes text containing a control or replacement rune.
func escapeForTerminal(text string) string {
	if needsEscaping(text) {
		return strconv.Quote(text)
	}

	return text
}

// needsEscaping reports whether text contains a control or replacement rune.
func needsEscaping(text string) bool {
	for _, r := range text {
		if unicode.IsControl(r) || r == utf8.RuneError {
			return true
		}
	}

	return false
}
