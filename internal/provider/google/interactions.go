package google

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/shdeen/bildomat/internal/catalog"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
	"github.com/shdeen/bildomat/internal/params"
	"github.com/shdeen/bildomat/internal/provider"
)

// The Interactions content fields, text content type, and error status.
const (
	interactionFieldMIMEType = "mime_type"
	interactionFieldData     = "data"
	interactionFieldType     = "type"
	interactionFieldText     = "text"
	interactionFieldModel    = "model"
	interactionTextType      = "text" //nolint:goconst // The content type and text field name are independent protocol values.
	interactionStatusError   = "error"
)

// interactionResponse contains the steps returned for an interaction.
type interactionResponse struct {
	// InteractionSteps contains the interaction's step history.
	InteractionSteps []interactionStep `json:"steps"`
}

// interactionStep contains one interaction step and its content.
type interactionStep struct {
	// Type identifies the step kind.
	Type string `json:"type"`
	// Status identifies the step's completion state.
	Status string `json:"status"`
	// Error contains the step's failure details.
	Error *interactionStepError `json:"error"`
	// Content contains the step's response blocks.
	Content []interactionBlock `json:"content"`
	// Summary contains the step's thought summary blocks.
	Summary []interactionBlock `json:"summary"`
}

// interactionStepError contains the diagnostic values for a failed interaction step.
type interactionStepError struct {
	// Code is the provider's error code.
	Code any `json:"code"`
	// Message is the provider's error message.
	Message string `json:"message"`
	// Details contains additional provider error data.
	Details json.RawMessage `json:"details"`
}

// interactionBlock contains one typed media or text block.
type interactionBlock struct {
	// Type identifies the block kind.
	Type string `json:"type"`
	// MimeType identifies the media data's MIME type.
	MimeType string `json:"mime_type"`
	// Data contains base64-encoded inline media.
	Data string `json:"data"`
	// Text contains the block's text.
	Text string `json:"text"`
	// URI identifies a file-backed media resource.
	URI string `json:"uri"`
}

// The Interactions API's tokens; the type, MIME type, data, text, model,
// and error fields and values are the shared words.
//   - interactionsPathSuffix: the request path under the API base
//   - wireKeyURI: the block field carrying a file resource URI
//   - wireKeyInput: the request field carrying the input blocks
//   - wireKeyResponseFormat: the request field describing the wanted response
//   - wireKeyGenerationConfig: the request field carrying generation settings
//   - wireKeyThinkingSummaries: the generation setting selecting thought summaries
//   - wireKeyDelivery: the response-format field selecting file delivery
//   - wireKeyVideoConfig: the generation setting carrying video settings
//   - wireKeyTask: the video setting naming the task
//   - wireValueThought: the step type carrying thought summaries
//   - wireValueModelOutput: the step type carrying the model's output
//   - taskImageToVideo: the video task animating one reference image
//   - taskReferenceToVideo: the video task drawing on several reference images
//   - thinkingSummariesAuto: the thought-summary selection value
const (
	interactionsPathSuffix = "/interactions"

	wireKeyURI               = "uri"
	wireKeyInput             = "input"
	wireKeyResponseFormat    = "response_format"
	wireKeyGenerationConfig  = "generation_config"
	wireKeyThinkingSummaries = "thinking_summaries"
	wireKeyDelivery          = "delivery"
	wireKeyVideoConfig       = "video_config"
	wireKeyTask              = "task"

	wireValueThought     = "thought"
	wireValueModelOutput = "model_output"

	taskImageToVideo     = "image_to_video"
	taskReferenceToVideo = "reference_to_video"

	thinkingSummariesAuto = "auto"
)

// The labels appended to the model, after a colon and a space, in the interaction
// diagnostics' contexts.
//   - interactionResponseContext: a response that does not decode
//   - interactionErrorContext: a failed step reporting no error record
//   - blockDataContext: a media block whose inline data does not decode, after the block type
const (
	interactionResponseContext = "interaction response"
	interactionErrorContext    = "interaction error"
	blockDataContext           = "block data"
)

// interactionInput returns the model and ordered input blocks for an Interactions request.
func interactionInput(modelID, prompt string, mediaInputs []media.Input) map[string]any {
	input := make([]map[string]any, 0, len(mediaInputs)+1)
	for _, mediaInput := range mediaInputs {
		mediaKind := mediaInput.Kind()
		if mediaKind == "" {
			mediaKind = media.Image
		}

		block := map[string]any{interactionFieldType: string(mediaKind), interactionFieldMIMEType: mediaInput.MIME}
		if mediaInput.URL != "" {
			block[wireKeyURI] = mediaInput.URL
		} else {
			block[interactionFieldData] = base64.StdEncoding.EncodeToString(mediaInput.Bytes)
		}

		input = append(input, block)
	}

	input = append(input, map[string]any{interactionFieldType: interactionTextType, interactionFieldText: prompt})

	return map[string]any{interactionFieldModel: modelID, wireKeyInput: input}
}

// interactionImageBody returns the request fields for an image interaction: the input blocks,
// the fixed response-format type, every declared parameter at its declared path, and the
// thought-summary selection. It returns the parameter error when a stored thoughts value has
// another type, and the unplaced-parameter error when a supplied parameter has no declared
// path.
func interactionImageBody(model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input) (map[string]any, error) {
	body := interactionInput(model.ID, prompt, mediaInputs)
	body[wireKeyResponseFormat] = map[string]any{interactionFieldType: string(media.Image)}

	if err := placeDeclaredParams(body, model, parameterValues); err != nil {
		return nil, err
	}

	return body, nil
}

// interactionVideoBody returns the request fields for a video interaction: the input blocks,
// the fixed response-format type and delivery, the task the input images select, every
// declared parameter at its declared path, and the thought-summary selection. A video input
// selects no task. It returns the errors interactionImageBody returns.
func interactionVideoBody(model *catalog.Model, prompt string, parameterValues params.Values, mediaInputs []media.Input) (map[string]any, error) {
	body := interactionInput(model.ID, prompt, mediaInputs)
	body[wireKeyResponseFormat] = map[string]any{interactionFieldType: string(media.Video), wireKeyDelivery: wireKeyURI}

	imageInputs, _ := media.SplitInputs(mediaInputs)
	if task := interactionVideoTask(len(imageInputs)); task != "" {
		body[wireKeyGenerationConfig] = map[string]any{wireKeyVideoConfig: map[string]any{wireKeyTask: task}}
	}

	if err := placeDeclaredParams(body, model, parameterValues); err != nil {
		return nil, err
	}

	return body, nil
}

// placeDeclaredParams merges validated parameter paths and adds the explicitly
// requested thought summary. Input blocks consume input media separately.
func placeDeclaredParams(body map[string]any, model *catalog.Model, parameterValues params.Values) error {
	mergeFields(body, provider.WireParamValues(model, parameterValues))

	thoughtsVal, err := params.Value[bool](parameterValues, params.FlagTypeThoughts)
	if err != nil {
		return err
	}

	if thoughtsVal {
		mergeFields(body, map[string]any{wireKeyGenerationConfig: map[string]any{wireKeyThinkingSummaries: thinkingSummariesAuto}})
	}

	for _, flag := range slices.Sorted(maps.Keys(parameterValues)) {
		if flag == params.FlagTypeThoughts || flag == params.FlagTypeInputMedia {
			continue
		}

		paramCfg, declared := model.Param(flag)
		if !declared || paramCfg.ParamID == "" {
			return &errs.ConfigError{Provider: ProviderID, Setting: string(flag), Problem: model.ID + ": " + string(flag), Cause: errs.ErrProvConfigParamUnplaced}
		}
	}

	return nil
}

// mergeFields takes destination and source request fields and writes each source field
// into the destination: a field whose value is an object on both sides merges one level
// down, and any other field replaces the destination's.
func mergeFields(dst, src map[string]any) {
	for key, value := range src {
		nestedSrc, srcIsObject := value.(map[string]any)
		nestedDst, dstIsObject := dst[key].(map[string]any)

		if srcIsObject && dstIsObject {
			mergeFields(nestedDst, nestedSrc)

			continue
		}

		dst[key] = value
	}
}

// interactionVideoTask returns the video task represented by the number of input image references.
func interactionVideoTask(refs int) string {
	switch refs {
	case 0:
		return ""
	case 1:
		return taskImageToVideo
	}

	return taskReferenceToVideo
}

// findMediaBlock returns the first usable block of the requested media type and the first text block encountered.
func findMediaBlock(blocks []interactionBlock, mediaBlockType string, mediaBlock *interactionBlock, firstTextBlock string) (selectedMediaBlock *interactionBlock, textBlock string) {
	for i := range blocks {
		block := &blocks[i]
		if block.Type == mediaBlockType && mediaBlock == nil && (block.Data != "" || block.URI != "") {
			mediaBlock = block
		}

		if block.Type == interactionTextType && block.Text != "" && firstTextBlock == "" {
			firstTextBlock = block.Text
		}
	}

	return mediaBlock, firstTextBlock
}

// newInteractionStepError returns a generation error containing the interaction step's diagnostic values.
func newInteractionStepError(model string, e *interactionStepError) error {
	msg := model + ": " + interactionErrorContext
	if e != nil {
		msg = fmt.Sprintf(ErrorCodeForm, model, e.Code, e.Message)
		if len(e.Details) > 0 {
			msg += " " + string(e.Details)
		}
	}

	return &errs.ProviderError{Message: msg, Cause: errs.ErrResponseGen}
}
