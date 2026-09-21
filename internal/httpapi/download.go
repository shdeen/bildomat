package httpapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/metadata"
)

// Fetch streams generated media into an owned temporary file. A failed download
// retains captured bytes independently, cleans its source, and returns no media.
func Fetch(ctx context.Context, endpoint string, credential AuthCredential, fallbackExt string, record *metadata.Record) (artifact.Media, error) {
	response, requestIndex, err := request(ctx, http.MethodGet, endpoint, credential, nil, nil, record)
	if err != nil {
		return artifact.Media{}, err
	}
	defer response.Body.Close() //nolint:errcheck // Reading or streaming reports transfer failures; closing this response only releases its connection.

	if response.StatusCode/100 != 2 {
		snippet, readErr := readFirstBytes(response.Body, maxRespBytes)
		record.Receive(requestIndex, metadata.Synchronous, response.StatusCode, response.Header.Get(headerContentType), snippet, nil, readErr)
		statusContext := fmt.Sprintf(StatusExcerptForm, endpoint, response.StatusCode, errs.Excerpt(snippet))
		statusErr := fmt.Errorf("%q: %w", statusContext, errs.ErrTransportStatus)

		return artifact.Media{}, requestFailure(ctx, endpoint, errors.Join(statusErr, readErr))
	}

	return stream(ctx, response, endpoint, fallbackExt, record, requestIndex)
}

// stream saves a complete nonempty response, counting its signature and remainder.
// Response retention precedes artifact cleanup on every unsuccessful download.
func stream(ctx context.Context, response *http.Response, endpoint, fallbackExt string, record *metadata.Record, requestIndex int) (artifact.Media, error) {
	temporaryFile, err := os.CreateTemp("", tempDownloadPattern)
	if err != nil {
		record.ReceiveFile(requestIndex, response.StatusCode, response.Header.Get(headerContentType), "", err)

		return artifact.Media{}, requestFailure(ctx, endpoint, errors.Join(errs.ErrTransportDownload, err))
	}

	signature, written, copyErr := copyDownload(temporaryFile, response.Body)
	downloadErr := errors.Join(copyErr, temporaryFile.Close())
	contentType := response.Header.Get(headerContentType)
	record.ReceiveFile(requestIndex, response.StatusCode, contentType, temporaryFile.Name(), downloadErr)

	if written == 0 && downloadErr == nil {
		downloadErr = errors.Join(downloadErr, errs.ErrResponseNoData)
	}

	if downloadErr != nil {
		cleanupErr := artifact.Cleanup([]artifact.Media{{TmpPath: temporaryFile.Name()}})

		return artifact.Media{}, requestFailure(ctx, endpoint, errors.Join(errs.ErrTransportDownload, downloadErr, cleanupErr))
	}

	return artifact.Media{TmpPath: temporaryFile.Name(), FileExt: media.ExtForData(contentType, signature, fallbackExt)}, nil
}

// copyDownload retains the signature while streaming all remaining bytes. It
// preserves a partial initial read and does not continue after a read or write failure.
func copyDownload(file *os.File, body io.Reader) (signature []byte, written int64, copyErr error) {
	signature, readErr := io.ReadAll(io.LimitReader(body, mediaTypeDetectBytes))

	prefixBytes, writeErr := file.Write(signature)
	if writeErr == nil && prefixBytes != len(signature) {
		writeErr = io.ErrShortWrite
	}

	if readErr != nil || writeErr != nil {
		return signature, int64(prefixBytes), errors.Join(readErr, writeErr)
	}

	remainingBytes, copyErr := io.Copy(file, body)

	return signature, int64(prefixBytes) + remainingBytes, copyErr //nolint:wrapcheck // stream adds the endpoint and download classification to this I/O cause.
}
