package catalog

// ImageAPI describes the endpoints and wire format for image generation.
//   - GenURL: the image generation endpoint
//   - InputMediaURL: the endpoint used when input media is present
//   - InputMediaListProvParam: the array field for multiple single-or-array references
//   - InputMediaProvParam: the request field for input media
//   - InputMediaPayloadType: the request encoding for input media
//   - InputMediaStyle: the JSON layout for input media
//   - FixedProvFields: the constant request fields
//   - FallbackExt: the file extension used when the response does not identify its media format
//   - RespImageURL: whether response image URLs are downloaded
type ImageAPI struct {
	GenURL                  string            `json:"genURL"`
	InputMediaURL           string            `json:"inputMediaURL"`
	InputMediaProvParam     string            `json:"inputMediaProvParam"`
	InputMediaListProvParam string            `json:"inputMediaListProvParam,omitempty"`
	InputMediaPayloadType   string            `json:"inputMediaPayloadType"`
	InputMediaStyle         InputMediaStyle   `json:"inputMediaStyle"`
	FixedProvFields         map[string]string `json:"fixedProvFields"`
	FallbackExt             string            `json:"fallbackExt"`
	RespImageURL            bool              `json:"respImageURL"`
}
