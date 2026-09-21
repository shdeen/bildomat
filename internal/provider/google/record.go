package google

import (
	"encoding/json"
	"fmt"

	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/metadata"
)

// retainedOperation is Google's provisional return-values shape. It preserves
// original provider references independently of downloaded artifact paths.
type retainedOperation struct {
	Operation string          `json:"operation"`
	Videos    []retainedVideo `json:"videos,omitempty"`
}

// retainedVideo identifies one original generated video for a follow-up request.
type retainedVideo struct {
	URI string `json:"uri"`
}

// retainOperation captures an acknowledgement or completed operation before
// later polling or artifact work can fail. The retained shape is experimental
// and subject to change with the provisional generation-record format.
func retainOperation(record *metadata.Record, model string, completed operation) error {
	if record == nil {
		return nil
	}

	retained := retainedOperation{Operation: completed.Name}
	if completed.Response != nil && completed.Response.GenerateVideoResponse != nil {
		for _, sample := range completed.Response.GenerateVideoResponse.GeneratedSamples {
			if sample.Video != nil && sample.Video.URI != "" {
				retained.Videos = append(retained.Videos, retainedVideo{URI: sample.Video.URI})
			}
		}
	}

	encoded, err := json.Marshal(retained)
	if err != nil {
		return fmt.Errorf("%q: %w, %w", completed.Name, errs.ErrJSONEncode, err)
	}

	record.Retain(ProviderID, model, encoded)

	return nil
}
