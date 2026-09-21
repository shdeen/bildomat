package metadata

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/shdeen/bildomat/internal/artifact"
	"github.com/shdeen/bildomat/internal/errs"
	"github.com/shdeen/bildomat/internal/media"
)

// Binary capture uses JSON null while file locations are pending, the standard
// data URI delimiter, and explicit HTTP schemes to distinguish remote sources.
const (
	jsonNull             = "null"
	base64Delimiter      = ";base64,"
	dataURIPrefix        = "data:"
	httpSourcePrefix     = "http://"
	httpsSourcePrefix    = "https://"
	audioMIMEPrefix      = "audio/"
	binaryFormat         = "bin"
	multipartContentType = "multipart/form-data"
	contentTypeHeader    = "Content-Type"
	retainedFileField    = "file"
)

// BinaryField identifies a known encoded field by its JSON path. An asterisk
// matches one array index or object property. MIMEField names a sibling field.
type BinaryField struct {
	Path      []string
	MIMEField string
	MIME      string
}

// binaryReference identifies a JSON value removed during binary capture. Only
// these exact locations receive replacement markers when final paths exist.
type binaryReference struct {
	Path      []string
	Digest    [sha256.Size]byte
	PlainPath bool
}

// binaryData owns one unique decoded body until the generation can be saved.
type binaryData struct {
	Bytes []byte
	MIME  string
}

// binaryStore retains unique decoded content and verified local source paths.
// Providers may discard their temporary artifacts without affecting this data.
type binaryStore struct {
	content   map[[sha256.Size]byte]binaryData
	sources   map[[sha256.Size]byte]string
	fileMIMEs map[string]string
}

// normalize preserves unknown JSON values and exact numbers, replacing only
// schema-declared base64 fields and explicit data URIs with pending references.
func (store *binaryStore) normalize(body []byte, fields []BinaryField) (json.RawMessage, []binaryReference, error) {
	return store.normalizeValue(bytes.TrimSpace(body), nil, nil, fields)
}

// normalizeValue recursively processes one value from an already validated JSON
// body. Decoded object members and array elements retain that validity.
func (store *binaryStore) normalizeValue(value json.RawMessage, path []string, parent map[string]json.RawMessage, fields []BinaryField) (json.RawMessage, []binaryReference, error) {
	var (
		references    []binaryReference
		captureErrors []error
	)

	openingByte := value[0] //nolint:nilaway // Both capture boundaries require json.Valid; recursively decoded members are nonempty JSON values.
	switch openingByte {
	case '{':
		var object map[string]json.RawMessage
		if err := json.Unmarshal(value, &object); err != nil {
			return value, nil, fmt.Errorf("%q: %w, %w", strings.Join(path, "/"), errs.ErrJSONDecode, err)
		}

		for _, name := range slices.Sorted(maps.Keys(object)) {
			normalized, childRefs, err := store.normalizeValue(object[name], append(slices.Clone(path), name), object, fields)
			object[name] = normalized

			references = append(references, childRefs...)
			captureErrors = append(captureErrors, err)
		}

		encoded, err := json.Marshal(object)
		captureErrors = append(captureErrors, err)

		return encoded, references, errors.Join(captureErrors...)
	case '[':
		var values []json.RawMessage
		if err := json.Unmarshal(value, &values); err != nil {
			return value, nil, fmt.Errorf("%q: %w, %w", strings.Join(path, "/"), errs.ErrJSONDecode, err)
		}

		for index := range values {
			normalized, childRefs, err := store.normalizeValue(values[index], append(slices.Clone(path), strconv.Itoa(index)), nil, fields)
			values[index] = normalized

			references = append(references, childRefs...)
			captureErrors = append(captureErrors, err)
		}

		encoded, err := json.Marshal(values)
		captureErrors = append(captureErrors, err)

		return encoded, references, errors.Join(captureErrors...)
	case '"':
		return store.normalizeString(value, path, parent, fields)
	}

	return value, nil, nil
}

// normalizeString recognizes one declared binary value without guessing from
// field names or string length. HTTP references remain exactly as submitted.
func (store *binaryStore) normalizeString(value json.RawMessage, path []string, parent map[string]json.RawMessage, fields []BinaryField) (json.RawMessage, []binaryReference, error) {
	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return value, nil, fmt.Errorf("%q: %w, %w", strings.Join(path, "/"), errs.ErrJSONDecode, err)
	}

	encoded, contentType, known := encodedValue(text, path, parent, fields)
	if !known {
		return value, nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		replacement, _ := json.Marshal(BinaryUnavailable) //nolint:errcheck,errchkjson // A string always has a JSON representation.

		return replacement, nil, fmt.Errorf("%q: %w, %w", strings.Join(path, "/"), errs.ErrResponseDecode, err)
	}

	reference := binaryReference{Path: slices.Clone(path), Digest: store.add(decoded, contentType)}

	return json.RawMessage(jsonNull), []binaryReference{reference}, nil
}

// encodedValue recognizes explicit data URIs or provider-declared base64 values.
func encodedValue(text string, path []string, parent map[string]json.RawMessage, fields []BinaryField) (encoded, contentType string, known bool) {
	if prefix, encoded, ok := strings.Cut(text, base64Delimiter); ok && strings.HasPrefix(prefix, dataURIPrefix) {
		return encoded, strings.TrimPrefix(prefix, dataURIPrefix), true
	}

	if text == "" || strings.HasPrefix(text, httpsSourcePrefix) || strings.HasPrefix(text, httpSourcePrefix) {
		return "", "", false
	}

	for _, field := range fields {
		if !matchesPath(field.Path, path) {
			continue
		}

		contentType := field.MIME
		if field.MIMEField != "" {
			var declared string
			if json.Unmarshal(parent[field.MIMEField], &declared) == nil && declared != "" {
				contentType = declared
			}
		}

		return text, contentType, true
	}

	return "", "", false
}

// matchesPath matches one schema path without recursively matching unrelated data.
func matchesPath(pattern, path []string) bool {
	if len(pattern) != len(path) {
		return false
	}

	for index, segment := range pattern {
		if segment != "*" && segment != path[index] {
			return false
		}
	}

	return true
}

// add retains the first copy of identical content and prefers declared MIME data.
func (store *binaryStore) add(data []byte, contentType string) [sha256.Size]byte {
	digest := sha256.Sum256(data)

	if store.content == nil {
		store.content = make(map[[sha256.Size]byte]binaryData)
	}

	retained, exists := store.content[digest]
	if !exists {
		store.content[digest] = binaryData{Bytes: data, MIME: contentType}
	} else if retained.MIME == "" && contentType != "" {
		retained.MIME = contentType
		store.content[digest] = retained
	}

	return digest
}

// source registers only an unchanged local file as a candidate for reuse.
func (store *binaryStore) source(input *media.Input) {
	if input.Filepath == "" || len(input.Bytes) == 0 {
		return
	}

	path, err := filepath.Abs(input.Filepath)
	if err != nil {
		return
	}
	// #nosec G304 -- path is the input selected by the user.
	original, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(original, input.Bytes) {
		return
	}

	if store.sources == nil {
		store.sources = make(map[[sha256.Size]byte]string)
	}

	store.sources[sha256.Sum256(original)] = path
}

// copyFile retains downloaded bytes before provider cleanup can remove them.
func (store *binaryStore) copyFile(path, contentType string) (binaryReference, error) {
	// #nosec G304 -- path is the downloaded temporary artifact.
	content, err := os.ReadFile(path)
	if err != nil {
		return binaryReference{}, &errs.MediaError{Source: path, Cause: errors.Join(errs.ErrInputMediaRead, err)}
	}

	return binaryReference{Digest: store.add(content, contentType)}, nil
}

// persist matches completed artifacts first, then unchanged inputs, and finally
// writes only binary content that needs its own retained file.
func (store *binaryStore) persist(dir, stem string, files []artifact.SavedFile) (paths map[[sha256.Size]byte]string, failures map[[sha256.Size]byte]error, artifactErr error) {
	paths, artifactErr = store.matchArtifacts(files)
	failures = make(map[[sha256.Size]byte]error)

	for digest, path := range store.sources {
		if paths[digest] != "" {
			continue
		}
		// #nosec G304 -- path is the original user-selected input, rechecked before reference.
		data, err := os.ReadFile(path)
		if err == nil && sha256.Sum256(data) == digest {
			paths[digest] = path
		}
	}

	for _, digest := range sortedDigests(store.content) {
		if paths[digest] != "" {
			continue
		}

		content := store.content[digest]
		extension := binaryExtension(content.MIME, content.Bytes)

		file, err := artifact.Write(dir, stem+".bild-data", extension, content.Bytes)
		if err != nil {
			failures[digest] = err

			continue
		}

		paths[digest] = file.Path
	}

	return paths, failures, artifactErr
}

// matchArtifacts identifies completed media by its actual bytes and retains the
// provider-declared MIME type when matching captured content supplies it.
func (store *binaryStore) matchArtifacts(files []artifact.SavedFile) (map[[sha256.Size]byte]string, error) {
	paths := make(map[[sha256.Size]byte]string)
	store.fileMIMEs = make(map[string]string)

	var readErrors []error

	for _, file := range files {
		// #nosec G304 -- file.Path is a successfully closed generated artifact.
		data, err := os.ReadFile(file.Path)
		if err != nil {
			readErrors = append(readErrors, &errs.MediaError{Source: file.Path, Cause: errors.Join(errs.ErrInputMediaRead, err)})

			continue
		}

		digest := sha256.Sum256(data)
		paths[digest] = file.Path

		contentType := store.content[digest].MIME
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}

		store.fileMIMEs[file.Path] = contentType
	}

	return paths, errors.Join(readErrors...)
}

// sortedDigests gives retained files a stable content order.
func sortedDigests(content map[[sha256.Size]byte]binaryData) [][sha256.Size]byte {
	digests := make([][sha256.Size]byte, 0, len(content))
	for digest := range content {
		digests = append(digests, digest)
	}

	slices.SortFunc(digests, compareDigests)

	return digests
}

// compareDigests orders binary fingerprints lexicographically.
func compareDigests(left, right [sha256.Size]byte) int { return bytes.Compare(left[:], right[:]) }

// binaryExtension prefers a declared format, then detected bytes, and otherwise
// uses a binary extension instead of inventing an image format.
func binaryExtension(contentType string, data []byte) string {
	if extension := media.ExtForMimeOr(contentType, ""); extension != "" {
		return extension
	}

	if extension := media.ExtForMimeOr(http.DetectContentType(data), ""); extension != "" {
		return extension
	}

	if strings.HasPrefix(contentType, audioMIMEPrefix) {
		extensions, err := mime.ExtensionsByType(contentType)
		if err == nil && len(extensions) > 0 {
			slices.Sort(extensions)

			return extensions[0]
		}
	}

	return "." + binaryFormat
}

// request captures JSON, multipart fields with repeated names, or an opaque
// request body. It parses the actual body that transport will send.
func (store *binaryStore) request(body []byte, contentType string, fields []BinaryField) (json.RawMessage, []binaryReference, error) {
	if len(body) == 0 {
		return json.RawMessage(jsonNull), nil, nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType == multipartContentType {
		return store.multipart(body, params["boundary"])
	}

	if json.Valid(body) {
		return store.normalize(body, fields)
	}

	encoded, err := json.Marshal(string(body))
	if err != nil {
		return nil, nil, fmt.Errorf("%q: %w, %w", contentType, errs.ErrJSONEncode, err)
	}

	return encoded, nil, nil
}

// formPart represents one submitted multipart part, preserving repeated names.
//
//nolint:tagliatelle // Keep the provisional record's hyphenated JSON keys.
type formPart struct {
	Name        string  `json:"name"`
	Filename    string  `json:"filename,omitempty"`
	ContentType string  `json:"content-type,omitempty"`
	Value       string  `json:"value,omitempty"`
	File        *string `json:"file,omitempty"`
}

// multipart records text parts and retained binary references in their sent order.
func (store *binaryStore) multipart(body []byte, boundary string) (json.RawMessage, []binaryReference, error) {
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	parts := []formPart{}

	var references []binaryReference

	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, nil, fmt.Errorf("%q: %w, %w", boundary, errs.ErrTransportMultipart, err)
		}

		data, readErr := io.ReadAll(part)

		closeErr := part.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return nil, nil, fmt.Errorf("%q: %w, %w", part.FormName(), errs.ErrTransportMultipart, err)
		}

		description := formPart{Name: part.FormName(), Filename: part.FileName(), ContentType: part.Header.Get(contentTypeHeader)}
		if part.FileName() == "" {
			description.Value = string(data)
		} else {
			description.File = new(string)
			references = append(references, binaryReference{Path: []string{strconv.Itoa(len(parts)), retainedFileField}, Digest: store.add(data, description.ContentType), PlainPath: true})
		}

		parts = append(parts, description)
	}

	encoded, err := json.Marshal(parts)
	if err != nil {
		return nil, nil, fmt.Errorf("%q: %w, %w", boundary, errs.ErrJSONEncode, err)
	}

	return encoded, references, nil
}

// resolveReferences replaces only recorded binary locations with truthful paths.
func resolveReferences(body json.RawMessage, references []binaryReference, paths map[[sha256.Size]byte]string, failures map[[sha256.Size]byte]error) (json.RawMessage, error) {
	var fieldErrors []error

	for _, reference := range references {
		if err := failures[reference.Digest]; err != nil {
			fieldErrors = append(fieldErrors, fmt.Errorf("%q: %w", strings.Join(reference.Path, "/"), err))
		}

		replacement := BinaryUnavailable
		if path := paths[reference.Digest]; path != "" {
			replacement = fmt.Sprintf(BinarySaved, path)
			if reference.PlainPath {
				replacement = path
			}
		}

		value, _ := json.Marshal(replacement) //nolint:errcheck,errchkjson // A string always has a JSON representation.
		body = replaceJSON(body, reference.Path, value)
	}

	return body, errors.Join(fieldErrors...)
}

// replaceJSON updates a known JSON location while preserving every other value.
func replaceJSON(body json.RawMessage, path []string, value json.RawMessage) json.RawMessage {
	if len(path) == 0 {
		return value
	}

	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return body
	}

	switch body[0] {
	case '{':
		var object map[string]json.RawMessage
		if json.Unmarshal(body, &object) != nil {
			return body
		}

		object[path[0]] = replaceJSON(object[path[0]], path[1:], value) //nolint:nilaway // Successful decoding of a body beginning with '{' initializes the object map.

		encoded, err := json.Marshal(object)
		if err == nil {
			return encoded
		}
	case '[':
		var values []json.RawMessage
		if json.Unmarshal(body, &values) != nil {
			return body
		}

		index, err := strconv.Atoi(path[0])
		if err != nil || index < 0 || index >= len(values) {
			return body
		}

		values[index] = replaceJSON(values[index], path[1:], value)

		encoded, err := json.Marshal(values)
		if err == nil {
			return encoded
		}
	}

	return body
}
