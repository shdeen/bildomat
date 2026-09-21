package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/provider"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
)

// The Google adapter's tokens.
//   - filesPathSegment: the path segment naming the Files resource, which also opens a resource name
//   - fileIDStopChars: the characters a bare file ID never contains
//   - downloadQuerySuffix: the file-download query
//   - stateResponseContext: names the file state response in a failed-decode context
//   - fileStateProcessing: the file state while the file is still being processed
//   - fileStateActive: the file state once the file is ready
//   - fileStateFailed: the file state after processing failed
const (
	filesPathSegment     = "files"
	fileIDStopChars      = "/:?"
	downloadQuerySuffix  = ":download?alt=media"
	stateResponseContext = "state response"
	fileStateProcessing  = "PROCESSING"
	fileStateActive      = "ACTIVE"
	fileStateFailed      = "FAILED"
)

// canonicalFileID returns the canonical files/<id> resource represented by a file reference.
func canonicalFileID(fileRef string) (string, error) {
	if id, ok := strings.CutPrefix(fileRef, filesPathSegment+"/"); ok && id != "" && !strings.ContainsAny(id, fileIDStopChars) {
		return fileRef, nil
	}

	if id := fileIDFromURI(fileRef); id != "" {
		return filesPathSegment + "/" + id, nil
	}

	return "", fmt.Errorf("%q, %w", fileRef, errs.ErrResponseDecode)
}

// fileIDFromURI returns the file identifier in an absolute Files download URI.
func fileIDFromURI(ref string) string {
	u, err := url.Parse(ref)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return ""
	}

	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i, seg := range segs {
		if seg != filesPathSegment {
			continue
		}

		if i != len(segs)-2 || slices.Contains(segs[i+1:], filesPathSegment) {
			return ""
		}

		id := segs[i+1]
		if j := strings.IndexByte(id, ':'); j >= 0 {
			id = id[:j]
		}

		return id
	}

	return ""
}

// fileReadyProbe holds the values needed to inspect a file resource.
//   - apiBase: the base endpoint for file status requests
//   - credential: the credential sent with each status request
//   - fileResource: the canonical file resource name
//   - record: optional retention of the file-status requests and responses
type fileReadyProbe struct {
	apiBase      string
	credential   httpapi.AuthCredential
	fileResource string
	record       *metadata.Record
}

// Poll checks the file resource and reports whether it is ready for download.
func (probe *fileReadyProbe) Poll(ctx context.Context) (complete bool, err error) {
	status, body, err := httpapi.GetAuth(ctx, probe.apiBase+"/"+probe.fileResource, probe.credential, metadata.Asynchronous, probe.record)
	if err := provider.PollResponseError(probe.fileResource, status, body, err); err != nil {
		return false, err
	}

	return checkFileReady(probe.fileResource, body)
}

// checkFileReady returns whether a file response reports a ready state.
func checkFileReady(fileResource string, body []byte) (bool, error) {
	var fileState struct {
		State string `json:"state"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &fileState); err != nil {
		return false, fmt.Errorf("%q, %w, %w", fileResource+": "+stateResponseContext, errs.ErrResponseDecode, err)
	}

	switch fileState.State {
	case fileStateProcessing:
		return false, nil
	case fileStateActive:
		return true, nil
	case fileStateFailed:
		msg := fileResource
		if fileState.Error.Message != "" {
			msg += ": " + fileState.Error.Message
		}

		return false, &errs.ProviderError{Message: msg, Cause: errs.ErrResponseGen}
	}

	obs := fileState.State
	if obs == "" {
		obs = provider.MissingResponseValue
	}

	return false, fmt.Errorf("%q, %w", fmt.Sprintf("%s (%s)", fileResource, obs), errs.ErrResponseUnknown)
}
