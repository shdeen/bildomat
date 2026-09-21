package catalog

// InputMediaStyle identifies the request layout used for input media.
type InputMediaStyle string

// Input media request layouts.
//   - InputMediaParts: local images are sent as multipart file parts
//   - InputMediaSingle: one reference uses a single field and multiple references use an array
//   - InputMediaNested: references use nested image_url or video_url objects
//   - InputMediaString: references use URL or data-URI strings directly
const (
	InputMediaParts  InputMediaStyle = "parts"
	InputMediaSingle InputMediaStyle = "single-or-array"
	InputMediaNested InputMediaStyle = "nested"
	InputMediaString InputMediaStyle = "string"
)
