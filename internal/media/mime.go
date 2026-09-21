package media

import (
	"net/http"
	"slices"
	"strings"
)

// Kind identifies an image or video. An empty kind has not been established.
type Kind string

// Image and Video are the supported media kinds.
const (
	Image Kind = "image"
	Video Kind = "video"
)

// Known MIME values for supported local inputs and generated SVG output.
const (
	MimePNG  = "image/png"
	MimeJPEG = "image/jpeg"
	MimeWebP = "image/webp"
	MimeMP4  = "video/mp4"
	mimeSVG  = "image/svg+xml"
)

// Format tokens used in provider requests and file extensions.
const (
	FormatPNG  = "png"
	FormatJPEG = "jpeg"
	FormatJPG  = "jpg"
	FormatWebP = "webp"
	FormatMP4  = "mp4"
	FormatSVG  = "svg"
)

// format describes a known encoding, its canonical extension, and input support.
type format struct {
	mime       string
	extension  string
	aliases    []string
	localInput bool
}

// mimeJPG is the alternate JPEG MIME spelling accepted by the format lookup.
const mimeJPG = "image/jpg"

// formats is the shared description of recognized encodings and MIME aliases.
//
//nolint:gochecknoglobals // immutable format descriptions, initialized once
var formats = []format{
	{mime: MimePNG, extension: FormatPNG, localInput: true},
	{mime: MimeJPEG, extension: FormatJPG, aliases: []string{mimeJPG}, localInput: true},
	{mime: MimeWebP, extension: FormatWebP, localInput: true},
	{mime: MimeMP4, extension: FormatMP4, localInput: true},
	{mime: mimeSVG, extension: FormatSVG},
}

// MIMEKind returns the image or video category of a MIME value, or empty when unresolved.
func MIMEKind(mimeType string) Kind {
	category, subtype, valid := splitMIME(mimeType)
	if !valid || subtype == "" {
		return ""
	}

	return Kind(category)
}

// formatForMIME returns the known encoding identified by a MIME value or alias.
func formatForMIME(mimeType string) (format, bool) {
	category, subtype, valid := splitMIME(mimeType)
	if !valid {
		return format{}, false
	}

	normalized := category + "/" + subtype
	for _, encoding := range formats {
		if normalized == encoding.mime || slices.Contains(encoding.aliases, normalized) {
			return encoding, true
		}
	}

	return format{}, false
}

// extForMime returns a canonical media extension or a bare dot when unavailable.
// Valid unknown image/video subtypes retain their existing extension fallback.
func extForMime(mimeType string) string {
	_, subtype, valid := splitMIME(mimeType)
	if !valid || subtype == "" {
		return "."
	}

	if encoding, known := formatForMIME(mimeType); known {
		return "." + encoding.extension
	}

	return "." + subtype
}

// ExtForMimeOr returns the MIME extension or the supplied fallback extension.
func ExtForMimeOr(mimeType, fallback string) string {
	if extension := extForMime(mimeType); extension != "." {
		return extension
	}

	return fallback
}

// splitMIME normalizes a media category and its leading subtype token.
func splitMIME(mimeType string) (category, subtype string, valid bool) {
	category, subtype, separated := strings.Cut(strings.ToLower(strings.TrimSpace(mimeType)), "/")
	if !separated || (category != string(Image) && category != string(Video)) {
		return "", "", false
	}

	return category, mimeSubtype(subtype), true
}

// mimeSubtype returns the leading valid subtype token, excluding parameters.
func mimeSubtype(mimeValue string) string {
	mimeValue = strings.TrimLeft(mimeValue, " \t")
	for index := range len(mimeValue) {
		character := mimeValue[index]
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '+' && character != '-' && character != '.' {
			return mimeValue[:index]
		}
	}

	return mimeValue
}

// ExtForData returns an extension inferred from a MIME type, leading bytes, or a fallback extension in that order.
func ExtForData(mime string, byteHead []byte, fallbackExt string) string {
	if ext := ExtForMimeOr(mime, ""); ext != "" {
		return ext
	}

	if ext := ExtForMimeOr(http.DetectContentType(byteHead), ""); ext != "" {
		return ext
	}

	return fallbackExt
}
