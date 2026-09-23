// Package media owns media sources, formats, and local image transformations.
package media

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/shdeen/bildomat/internal/errs"
)

// Remote media source syntax.
//   - schemeHTTP: unencrypted HTTP
//   - schemeHTTPS: encrypted HTTP
//   - schemeSeparator: the delimiter between scheme and address
const (
	schemeHTTP      = "http"
	schemeHTTPS     = "https"
	schemeSeparator = "://"
)

// Frame anchors identify the requested position in a generated video.
//   - FrameFirst: the opening frame
//   - FrameLast: the closing frame
const (
	FrameFirst = "first"
	FrameLast  = "last"
)

// inputLimit is the maximum accepted local input size, in bytes.
const inputLimit = 64 << 20

// Input contains local bytes or a remote URL and an optional frame request. Copies share immutable
// bytes and time values; transformations replace them.
//   - Bytes: local or downloaded media content
//   - MIME: the identified media type, or empty while unresolved
//   - Filepath: the local path or source URL retained after download
//   - URL: the remote source while it remains a URL reference
//   - Time: numeric frame seconds; nil means absent and zero requests the opening time
//   - FrameAnchor: first or last; takes precedence over Time when formatting a source
type Input struct {
	Bytes       []byte
	MIME        string
	Filepath    string
	URL         string
	Time        *float64
	FrameAnchor string
}

// Kind returns the kind established by the MIME type, or empty when unresolved.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) Kind() Kind {
	return MIMEKind(mediaInput.MIME)
}

// HasFrame reports whether a numeric time or anchor was supplied.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) HasFrame() bool {
	return mediaInput.Time != nil || mediaInput.FrameAnchor != ""
}

// FrameTime returns the numeric frame time and distinguishes absence from zero.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) FrameTime() (float64, bool) {
	if mediaInput.Time == nil {
		return 0, false
	}

	return *mediaInput.Time, true
}

// SourceName returns the bare local path or remote URL.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) SourceName() string {
	if mediaInput.URL != "" {
		return mediaInput.URL
	}

	return mediaInput.Filepath
}

// Source returns the source including its optional frame prefix.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) Source() string {
	if mediaInput.FrameAnchor != "" {
		return mediaInput.FrameAnchor + ":" + mediaInput.SourceName()
	}

	if mediaInput.Time != nil {
		return fmt.Sprintf("%g:%s", *mediaInput.Time, mediaInput.SourceName())
	}

	return mediaInput.SourceName()
}

// DataURI returns a remote URL unchanged or local bytes encoded with their MIME type. Without a
// MIME type, a local input has no data URI.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) DataURI() string {
	if mediaInput.URL != "" {
		return mediaInput.URL
	}

	if mediaInput.MIME == "" {
		return ""
	}

	return "data:" + mediaInput.MIME + ";base64," + base64.StdEncoding.EncodeToString(mediaInput.Bytes)
}

// URLOrBase64 returns a remote URL unchanged or local bytes as raw Base64.
//
//nolint:gocritic // Input is an immutable value; the receiver does not copy its backing bytes.
func (mediaInput Input) URLOrBase64() string {
	if mediaInput.URL != "" {
		return mediaInput.URL
	}

	return base64.StdEncoding.EncodeToString(mediaInput.Bytes)
}

// ReadInputs reads local media and retains validated HTTP(S) sources in supplied order. Remote
// sources remain unresolved; any invalid source fails the entire read.
func ReadInputs(sources []string) ([]Input, error) {
	mediaInputs := make([]Input, 0, len(sources))
	for _, source := range sources {
		mediaInput, err := readSource(source)
		if err != nil {
			return nil, err
		}

		mediaInputs = append(mediaInputs, mediaInput)
	}

	return mediaInputs, nil
}

// readSource reads local media or preserves a validated, unresolved HTTP(S) URL.
func readSource(source string) (Input, error) {
	bareSource, frameTime, anchor, err := splitFramePrefix(source)
	if err != nil {
		return Input{}, err
	}

	var mediaInput Input

	switch {
	case isURL(bareSource):
		parsedURL, parseErr := url.Parse(bareSource)
		if parseErr != nil {
			return Input{}, &errs.MediaError{Source: bareSource, Cause: errors.Join(errs.ErrInputMediaSource, parseErr)}
		}

		if parsedURL.Host == "" {
			return Input{}, &errs.MediaError{Source: bareSource, Cause: errs.ErrInputMediaSource}
		}

		mediaInput.URL = bareSource
	case strings.Contains(bareSource, schemeSeparator):
		return Input{}, &errs.MediaError{Source: bareSource, Cause: errs.ErrInputMediaSource}
	default:
		mediaInput, err = readLocal(bareSource)
		if err != nil {
			return Input{}, err
		}
	}

	mediaInput.Time, mediaInput.FrameAnchor = frameTime, anchor

	return mediaInput, nil
}

// readLocal reads and validates local image or MP4 bytes within the input limit.
func readLocal(path string) (Input, error) {
	// #nosec G304 -- path is the user's requested local media source.
	file, err := os.Open(path)
	if err != nil {
		classification := errs.ErrInputMediaRead
		if errors.Is(err, os.ErrNotExist) {
			classification = errs.ErrInputMediaNotFound
		}

		return Input{}, &errs.MediaError{Source: path, Cause: errors.Join(classification, err)}
	}
	defer file.Close() //nolint:errcheck // closing a read-only input cannot alter the read result

	data, err := io.ReadAll(io.LimitReader(file, inputLimit+1))
	if err != nil {
		return Input{}, &errs.MediaError{Source: path, Cause: errors.Join(errs.ErrInputMediaRead, err)}
	}

	if len(data) > inputLimit {
		return Input{}, &errs.MediaError{Source: path, Cause: fmt.Errorf("%d: %w, %w", inputLimit, errs.ErrInputMediaRead, errs.ErrInputMediaSize)}
	}

	mimeType := http.DetectContentType(data)

	format, known := formatForMIME(mimeType)
	if !known || !format.localInput {
		return Input{}, &errs.MediaError{Source: path, Problem: mimeType, Cause: errs.ErrInputMediaMIME}
	}

	return Input{Bytes: data, MIME: mimeType, Filepath: path}, nil
}

// IsURLSource recognizes an HTTP(S) source even when its optional frame prefix is invalid.
func IsURLSource(source string) bool {
	bareSource, _, _, _ := splitFramePrefix(source) //nolint:dogsled,errcheck // Recognition needs only the source; readSource reports errors after the CLI preserves the complete value.

	return isURL(bareSource)
}

// isURL recognizes the two supported schemes without interpreting remote content.
func isURL(source string) bool {
	lowerSource := strings.ToLower(source)

	return strings.HasPrefix(lowerSource, schemeHTTP+schemeSeparator) || strings.HasPrefix(lowerSource, schemeHTTPS+schemeSeparator)
}

// splitFramePrefix separates a source from its numeric time or first/last anchor. Invalid numeric
// times retain their source and classified error. Ordinary filenames, including colons, pass
// through when no frame prefix is present.
func splitFramePrefix(source string) (bareSource string, frameTime *float64, anchor string, err error) {
	prefix, remainder, separated := strings.Cut(source, ":")
	if !separated || remainder == "" {
		return source, nil, "", nil
	}

	if strings.EqualFold(prefix, FrameFirst) {
		return remainder, nil, FrameFirst, nil
	}

	if strings.EqualFold(prefix, FrameLast) {
		return remainder, nil, FrameLast, nil
	}

	seconds, err := strconv.ParseFloat(prefix, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		if isURL(remainder) {
			return remainder, nil, "", &errs.MediaError{Source: source, Cause: errs.ErrInputMediaSource}
		}

		return source, nil, "", nil
	}

	if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return remainder, nil, "", &errs.MediaError{Problem: fmt.Sprintf(FrameTimeInvalid, source), Cause: errs.ErrInputMediaTime}
	}

	return remainder, &seconds, "", nil
}

// SplitInputs groups video references separately, retaining the relative order of each group.
// Unresolved media keeps the image treatment used by adapters.
func SplitInputs(inputs []Input) (imageInputs, videoInputs []Input) {
	for _, input := range inputs {
		if input.Kind() == Video {
			videoInputs = append(videoInputs, input)
		} else {
			imageInputs = append(imageInputs, input)
		}
	}

	return imageInputs, videoInputs
}
