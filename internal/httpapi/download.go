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

// Fetch streams generated media into a temporary file that the caller owns. On failure it removes
// that file and returns no media. A nonnil record retains received bytes.
func Fetch(ctx context.Context, endpoint string, credential AuthCredential, fallbackExt string, record *metadata.Record) (artifact.Media, error) {
	response, requestIndex, err := request(ctx, http.MethodGet, endpoint, credential, nil, nil, record)
	if err != nil {
		return artifact.Media{}, err
	}
	defer response.Body.Close() //nolint:errcheck // Reading or streaming reports transfer failures; closing this response only releases its connection.

	if response.StatusCode/100 != 2 {
		snippet, readErr := readFirstBytes(response.Body, maxRespBytes)
		record.Receive(requestIndex, metadata.Synchronous, response.StatusCode, response.Header.Get(headerContentType), snippet, readErr)
		statusContext := fmt.Sprintf(StatusExcerptForm, endpoint, response.StatusCode, errs.Excerpt(snippet))
		statusErr := fmt.Errorf("%q: %w", statusContext, errs.ErrTransportStatus)

		return artifact.Media{}, requestFailure(ctx, endpoint, errors.Join(statusErr, readErr))
	}

	return stream(ctx, response, endpoint, fallbackExt, record, requestIndex)
}

// stream writes a complete nonempty response to a temporary artifact file. A nonnil record retains
// the response before a failed download's temporary file is removed.
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

// copyDownload streams the body into file, returning its leading signature and written byte count.
// It retains a partial initial read and stops on any read or write failure.
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
