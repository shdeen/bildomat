package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
)

// mimeOctetStream denotes content whose media type is unresolved.
const mimeOctetStream = "application/octet-stream"

// headerRange requests only the prefix needed to identify a source's media type.
const headerRange = "Range"

// bytePrefixRangeForm expresses an inclusive HTTP byte range starting at zero.
const bytePrefixRangeForm = "bytes=0-%d"

// ResolveInputMediaTypes returns copies with unresolved remote media identified. It tries HTTP
// metadata, then a bounded content sample, without provider credentials. Source URLs, existing
// metadata, bytes, and frame information remain unchanged. A source that cannot be read or
// identified returns a classified input error.
func ResolveInputMediaTypes(ctx context.Context, mediaInputs []media.Input) ([]media.Input, error) {
	resolvedInputs := slices.Clone(mediaInputs)
	for inputIndex, mediaInput := range resolvedInputs {
		if mediaInput.URL == "" || mediaInput.Kind() != "" {
			continue
		}

		mediaType, err := requestMediaType(ctx, mediaInput.URL, http.MethodHead)
		if (err != nil || mediaType == "") && ctx.Err() == nil {
			mediaType, err = requestMediaType(ctx, mediaInput.URL, http.MethodGet)
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			err = errors.Join(err, ctxErr)
		}

		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				err = fmt.Errorf("%w, %w", errs.ErrCanceled, err)
			}

			return nil, &errs.MediaError{Source: mediaInput.URL, Cause: errors.Join(errs.ErrInputMediaRead, err)}
		}

		if mediaType == "" {
			return nil, &errs.MediaError{Source: mediaInput.URL, Cause: errors.Join(errs.ErrInputMediaRead, errs.ErrInputMediaMIME)}
		}

		resolvedInputs[inputIndex].MIME = mediaType
	}

	return resolvedInputs, nil
}

// requestMediaType inspects successful source metadata and, for GET, a bounded prefix. HEAD servers
// may omit metadata or reject the method; the caller decides whether to retry using GET. Even
// servers that ignore Range are read only to the limit.
func requestMediaType(ctx context.Context, sourceURL, method string) (string, error) {
	headers := make(http.Header)
	if method == http.MethodGet {
		headers.Set(headerRange, fmt.Sprintf(bytePrefixRangeForm, mediaTypeDetectBytes-1))
	}

	response, _, err := request(ctx, method, sourceURL, AuthCredential{}, headers, nil, nil)
	if err != nil {
		return "", err
	}
	defer response.Body.Close() //nolint:errcheck // A read-only response close cannot alter the identified media type.

	if response.StatusCode/100 != 2 {
		statusContext := fmt.Sprintf(errs.StatusContextForm, sourceURL, response.StatusCode)

		return "", fmt.Errorf("%q, %w", statusContext, errs.ErrTransportStatus)
	}

	mediaType := usableMediaType(response.Header.Get(headerContentType))
	if mediaType != "" || method == http.MethodHead {
		return mediaType, nil
	}

	contentPrefix, err := io.ReadAll(io.LimitReader(response.Body, mediaTypeDetectBytes))
	if err != nil {
		return "", fmt.Errorf("%q, %w, %w", sourceURL, errs.ErrTransportRead, err)
	}

	return usableMediaType(http.DetectContentType(contentPrefix)), nil
}

// usableMediaType returns the normalized image or video MIME type of a valid header.
func usableMediaType(contentType string) string {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || media.MIMEKind(mediaType) == "" {
		return ""
	}

	return mediaType
}

// DownloadInputMedia downloads URL inputs without credentials into a copied media slice. It
// preserves source order and leaves format compatibility to the receiving provider. On failure it
// returns completed downloads and untouched remaining inputs without mutating the caller's slice.
func DownloadInputMedia(ctx context.Context, mediaInputs []media.Input) ([]media.Input, error) {
	downloadedInputs := slices.Clone(mediaInputs)
	for inputIndex, mediaInput := range mediaInputs {
		if mediaInput.URL == "" {
			continue
		}

		status, data, err := GetAuth(ctx, mediaInput.URL, AuthCredential{}, metadata.Synchronous, nil)
		if err != nil {
			return downloadedInputs, &errs.MediaError{Source: mediaInput.URL, Cause: errors.Join(errs.ErrInputMediaRead, err)}
		}

		if status/100 != 2 {
			return downloadedInputs, &errs.MediaError{Source: mediaInput.URL, Cause: fmt.Errorf("%d: %w, %w", status, errs.ErrInputMediaRead, errs.ErrTransportStatus)}
		}

		detectedMIME := http.DetectContentType(data)
		if detectedMIME != mimeOctetStream || mediaInput.MIME == "" {
			mediaInput.MIME = detectedMIME
		}

		mediaInput.Bytes = data
		mediaInput.Filepath = mediaInput.URL
		mediaInput.URL = ""
		downloadedInputs[inputIndex] = mediaInput
	}

	return downloadedInputs, nil
}
