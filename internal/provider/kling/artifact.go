package kling

import (
	"context"
	"errors"
	"fmt"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/httpapi"
	"github.com/shdeen/bildomat/internal/metadata"
)

// downloadImages downloads every completed image in provider index order.
// A failed download removes the temporary files from prior successful downloads.
func downloadImages(ctx context.Context, providerModelName string, resultURLs []string, fallbackExtension string, record *metadata.Record) ([]artifact.Media, error) {
	artifacts := make([]artifact.Media, 0, len(resultURLs))
	for _, resultURL := range resultURLs {
		generatedMedia, err := httpapi.Fetch(ctx, resultURL, httpapi.AuthCredential{}, fallbackExtension, record)
		if err != nil {
			cleanupErr := artifact.Cleanup(artifacts)

			return nil, fmt.Errorf("%q: %w, %w", providerModelName, errs.ErrTransportDownload, errors.Join(err, cleanupErr))
		}

		artifacts = append(artifacts, generatedMedia)
	}

	return artifacts, nil
}
