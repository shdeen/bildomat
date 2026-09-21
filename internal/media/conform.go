package media

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"

	_ "image/jpeg" // registers the JPEG decoder

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder

	"github.com/shdeen/bildomat/internal/errs"
)

// dimensionsForm renders a width and height as a dimension value.
const dimensionsForm = "%dx%d"

// Resize center-crops local images and returns replacement PNG bytes at concrete dimensions.
// The original byte buffers and optional timestamps remain unchanged.
func Resize(width, height int, mediaInputs []Input) ([]Input, error) {
	if len(mediaInputs) == 0 {
		return mediaInputs, nil
	}

	if width <= 0 || height <= 0 {
		return nil, &errs.MediaError{Problem: fmt.Sprintf(dimensionsForm, width, height), Cause: errs.ErrInputMediaSize}
	}

	resizedInputs := make([]Input, 0, len(mediaInputs))
	for inputIndex, mediaInput := range mediaInputs {
		if mediaInput.URL != "" || mediaInput.Kind() == Video {
			return nil, &errs.MediaError{Problem: fmt.Sprintf(IndexForm, inputIndex+1), Cause: errs.ErrInputMediaSource}
		}

		resizedInput, err := resizeItem(&mediaInput, width, height, fmt.Sprintf(ImageIndexForm, inputIndex+1))
		if err != nil {
			return nil, err
		}

		resizedInputs = append(resizedInputs, resizedInput)
	}

	return resizedInputs, nil
}

// resizeItem returns a replacement image with the requested centered crop and dimensions.
func resizeItem(mediaInput *Input, width, height int, position string) (Input, error) {
	sourceImage, _, err := image.Decode(bytes.NewReader(mediaInput.Bytes))
	if err != nil {
		return Input{}, &errs.MediaError{Problem: position, Cause: errors.Join(errs.ErrInputMediaDecode, err)}
	}

	sourceCrop := centerCrop(sourceImage.Bounds(), width, height)
	resizedImage := image.NewRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(resizedImage, resizedImage.Bounds(), sourceImage, sourceCrop, xdraw.Over, nil)

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, resizedImage); err != nil {
		return Input{}, &errs.MediaError{Problem: position, Cause: errors.Join(errs.ErrInputMediaEncode, err)}
	}

	return Input{Bytes: encoded.Bytes(), MIME: MimePNG, Filepath: mediaInput.Filepath, Time: mediaInput.Time, FrameAnchor: mediaInput.FrameAnchor}, nil
}

// centerCrop returns the largest centered source rectangle at the target ratio.
func centerCrop(bounds image.Rectangle, width, height int) image.Rectangle {
	sourceWidth, sourceHeight := bounds.Dx(), bounds.Dy()
	targetRatio := float64(width) / float64(height)

	sourceRatio := float64(sourceWidth) / float64(sourceHeight)
	if sourceRatio > targetRatio {
		croppedWidth := int(float64(sourceHeight) * targetRatio)
		cropLeft := bounds.Min.X + (sourceWidth-croppedWidth)/2

		return image.Rect(cropLeft, bounds.Min.Y, cropLeft+croppedWidth, bounds.Max.Y)
	}

	croppedHeight := int(float64(sourceWidth) / targetRatio)
	cropTop := bounds.Min.Y + (sourceHeight-croppedHeight)/2

	return image.Rect(bounds.Min.X, cropTop, bounds.Max.X, cropTop+croppedHeight)
}
