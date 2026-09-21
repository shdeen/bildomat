package main

import (
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/metadata"
)

// reuseFlag is experimental, hidden from public help, and subject to change.
const reuseFlag = "reuse"

// reuseURIScheme distinguishes a supplied provider reference from a record path.
const reuseURIScheme = "https"

// parseReuse reads one experimental provider reuse selection. The option and
// generation-record format are provisional and subject to change.
func parseReuse(selection, providerID string) (*metadata.Reuse, error) {
	identifier, value, separated := strings.Cut(selection, "=")
	if !separated || identifier == "" || value == "" {
		return nil, fmt.Errorf("%q: %w", selection, errs.ErrReuseSyntax)
	}

	registered := false

	for _, provider := range providerRegistrations() {
		if provider.ProviderID == providerID {
			registered = slices.Contains(provider.ReuseIDs, identifier)

			break
		}
	}

	if !registered {
		return nil, fmt.Errorf("%q: %w", providerID+"/"+identifier, errs.ErrReuseProvider)
	}

	reference, parseErr := url.Parse(value)
	if parseErr == nil && reference.Scheme == reuseURIScheme && reference.Host != "" {
		return &metadata.Reuse{URI: value}, nil
	}

	expandedPath, err := artifact.ExpandHome(value)
	if err != nil {
		return nil, fmt.Errorf("%q: %w, %w", value, errs.ErrRecordRead, err)
	}

	recordPath, err := filepath.Abs(expandedPath)
	if err != nil {
		return nil, fmt.Errorf("%q: %w, %w", value, errs.ErrRecordRead, err)
	}

	record, err := metadata.Read(recordPath)
	if err != nil {
		return nil, err
	}

	return &metadata.Reuse{Record: record}, nil
}
