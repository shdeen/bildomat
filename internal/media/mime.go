package media

import (
	"net/http"
	"slices"
	"strings"
)

// Kind identifies an image or video. An empty kind has not been established.
type Kind string

// Supported media kinds.
//   - Image: still images
//   - Video: moving images
const (
	Image Kind = "image"
	Video Kind = "video"
)

// MIME types recognized for local input or generated output.
//   - mimePNG: PNG images
//   - mimeJPEG: JPEG images
//   - mimeWebP: WebP images
//   - mimeMP4: MP4 video
//   - mimeSVG: SVG output, which is not accepted as local input
const (
	mimePNG  = "image/png"
	mimeJPEG = "image/jpeg"
	mimeWebP = "image/webp"
	mimeMP4  = "video/mp4"
	mimeSVG  = "image/svg+xml"
)

// Format names used in provider requests and file extensions.
//   - FormatPNG: PNG encoding
//   - FormatJPEG: JPEG encoding
//   - FormatJPG: the canonical JPEG extension
//   - FormatWebP: WebP encoding
//   - FormatMP4: MP4 encoding
//   - formatSVG: SVG encoding
const (
	FormatPNG  = "png"
	FormatJPEG = "jpeg"
	FormatJPG  = "jpg"
	FormatWebP = "webp"
	FormatMP4  = "mp4"
	formatSVG  = "svg"
)

// format describes a recognized media encoding.
//   - mime: the canonical MIME type
//   - extension: the canonical extension without a leading dot
//   - aliases: accepted alternate MIME types
//   - localInput: whether local files of this type are accepted
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
	{mime: mimePNG, extension: FormatPNG, localInput: true},
	{mime: mimeJPEG, extension: FormatJPG, aliases: []string{mimeJPG}, localInput: true},
	{mime: mimeWebP, extension: FormatWebP, localInput: true},
	{mime: mimeMP4, extension: FormatMP4, localInput: true},
	{mime: mimeSVG, extension: formatSVG},
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

// extForMime returns a known encoding's canonical extension or a parsed image/video subtype. It
// returns a bare dot when no usable extension is present.
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

// splitMIME normalizes an image/video category and its leading subtype token. The boolean validates
// the category; the subtype may still be empty.
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

// ExtForData returns an extension inferred from a MIME type, leading bytes, or a fallback extension
// in that order.
func ExtForData(mime string, byteHead []byte, fallbackExt string) string {
	if ext := ExtForMimeOr(mime, ""); ext != "" {
		return ext
	}

	if ext := ExtForMimeOr(http.DetectContentType(byteHead), ""); ext != "" {
		return ext
	}

	return fallbackExt
}
