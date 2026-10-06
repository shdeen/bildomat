# Black Forest Labs Models

Every model that `bild` offers from Black Forest Labs (provider ID `bfl`, API key variable `BFL_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info bfl/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

22 image models, 3 video models.

## Contents

- [FLUX.2 [pro]](#flux2-pro)
- [FLUX.2 [pro] (preview)](#flux2-pro-preview)
- [FLUX.2 [max]](#flux2-max)
- [FLUX.2 [flex]](#flux2-flex)
- [FLUX.2 [klein] 4B](#flux2-klein-4b)
- [FLUX.2 [klein] 9B](#flux2-klein-9b)
- [FLUX.2 [klein] 9B (preview)](#flux2-klein-9b-preview)
- [FLUX1.1 [pro]](#flux11-pro)
- [FLUX.1 [dev]](#flux1-dev)
- [FLUX1.1 [pro] ultra](#flux11-pro-ultra)
- [FLUX.1 Fill [pro]](#flux1-fill-pro)
- [FLUX.1 Kontext [pro]](#flux1-kontext-pro)
- [FLUX.1 Kontext [max]](#flux1-kontext-max)
- [FLUX Outpainting](#flux-outpainting)
- [FLUX.2 Deblur](#flux2-deblur)
- [FLUX.3 Video](#flux3-video)
- [FLUX.1 Expand [pro]](#flux1-expand-pro)
- [FLUX.1 Fill [pro] Finetuned](#flux1-fill-pro-finetuned)
- [FLUX1.1 [pro] Ultra Finetuned](#flux11-pro-ultra-finetuned)
- [FLUX Erase](#flux-erase)
- [FLUX Virtual Try-On V1](#flux-virtual-try-on-v1)
- [FLUX Virtual Try-On V2](#flux-virtual-try-on-v2)
- [FLUX 3 Image](#flux-3-image)
- [FLUX Video Edit](#flux-video-edit)
- [FLUX Video Upscale](#flux-video-upscale)

## FLUX.2 [pro]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [pro] | image | `flux.2-pro` | `bfl/flux-2-pro`

Generate or edit an image with FLUX.2 [pro].

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 8; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--disable-prompt-upsampling` |

## FLUX.2 [pro] (preview)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [pro] (preview) | image | `flux.2-pro-preview` | `bfl/flux-2-pro-preview`

Generate or edit an image with FLUX.2 [pro] (preview).

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 8; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--disable-prompt-upsampling` |

## FLUX.2 [max]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [max] | image | `flux.2-max` | `bfl/flux-2-max`

Generate or edit an image with FLUX.2 [max].

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 8; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--disable-prompt-upsampling` |

## FLUX.2 [flex]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [flex] | image | `flux.2-flex` | `bfl/flux-2-flex`

Generate or edit an image with FLUX.2 [flex].

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 8; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--prompt-upsampling` |
`--guidance-scale` | Allowed range: 1.5 to 10
`--steps` | Allowed range: 1 to 50

## FLUX.2 [klein] 4B

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [klein] 4B | image | `flux.2-klein-4b` | `bfl/flux-2-klein-4b`

Generate or edit an image with FLUX.2 [klein] 4B.

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 4; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5

## FLUX.2 [klein] 9B

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [klein] 9B | image | `flux.2-klein-9b` | `bfl/flux-2-klein-9b`

Generate or edit an image with FLUX.2 [klein] 9B.

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 4; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5

## FLUX.2 [klein] 9B (preview)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 [klein] 9B (preview) | image | `flux.2-klein-9b-preview` | `bfl/flux-2-klein-9b-preview`

Generate or edit an image with FLUX.2 [klein] 9B (preview).

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 4; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5

## FLUX1.1 [pro]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX1.1 [pro] | image | `flux1.1-pro`, `flux.1.1-pro`, `flux-1.1-pro` | `bfl/flux-pro-1.1`

Generate an image with FLUX1.1 [pro].

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: edge: 256-1440, 32-px increments, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |

## FLUX.1 [dev]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.1 [dev] | image | `flux.1-dev` | `bfl/flux-dev`

Generate an image with FLUX.1 [dev].

Option | Constraints
-------|------------
`--size` | Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: edge: 256-1440, 32-px increments, long edge of 1024 when the size is derived from an aspect ratio
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--output-format` | Allowed values: jpeg, png, webp
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |
`--guidance-scale` | Allowed range: 1.5 to 5
`--steps` | Allowed range: 1 to 50

## FLUX1.1 [pro] ultra

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX1.1 [pro] ultra | image | `flux1.1-pro-ultra`, `flux.1.1-pro-ultra`, `flux-1.1-pro-ultra` | `bfl/flux-pro-1.1-ultra`

Generate an image with FLUX1.1 [pro] ultra mode.

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3
`--output-format` | Allowed values: jpeg, png, webp
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |

## FLUX.1 Fill [pro]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.1 Fill [pro] | image | `flux.1-fill-pro`, `flux-1-fill-pro` | `bfl/flux-pro-1.0-fill`

Inpaint an image with FLUX.1 Fill [pro] using an input image.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |
`--guidance-scale` | Allowed range: 1.5 to 100
`--steps` | Allowed range: 15 to 50

## FLUX.1 Kontext [pro]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.1 Kontext [pro] | image | `flux.1-kontext-pro` | `bfl/flux-kontext-pro`

Edit or create an image with FLUX.1 Kontext [pro].

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 4; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |

## FLUX.1 Kontext [max]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.1 Kontext [max] | image | `flux.1-kontext-max`, `flux-1-kontext-max` | `bfl/flux-kontext-max`

Edit or create an image with FLUX.1 Kontext [max].

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 4; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |

## FLUX Outpainting

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX Outpainting | image | `flux-outpainting-v1`, `flux-outpainting` | `bfl/flux-tools/outpainting-v1`

Extend an input image to a larger canvas.

This model does not require a prompt.

Option | Constraints
-------|------------
`--size` | **Required.** Examples: 1024x768, 1024x4096; <W>x<H> - Requirements: minimum edge: 64, maximum total pixels (width x height): 4194304
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--safety-tolerance` | Allowed range: 0 to 5
`--disable-prompt-upsampling` |

## FLUX.2 Deblur

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.2 Deblur | image | `flux-deblur-v1`, `flux-deblur` | `bfl/flux-tools/deblur-v1`

Remove blur from an image.

This model does not require a prompt.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5

## FLUX.3 Video

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.3 Video | video | `flux-3`, `flux.3-video`, `flux-video` | `bfl/flux-3-video`

FLUX.3 Video is a video generation model from Black Forest Labs. It supports text-to-video, image-guided generation with opening and closing keyframes, and video continuation workflows.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: auto, 21:9, 2:1, 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: hd, fhd
`--duration` | Allowed range: 5 to 20
`--input-media` | Repeat maximum: 10; Flux.3 Video special usage: <[keyframe:]media-file>. For each media input to appear a number of seconds into the clip, use an optional keyframe value in seconds (e.g., Specify '3:image.png' for the image to appear 3 seconds into the clip.) Images are keyframes: a numeric time can be set on any image and requires --duration. Alternatively, supply one untimed video to continue it; images and a video cannot be combined. See online docs for details on keyframe usage and supporting models: https://bildomat.com/docs.
`--safety-tolerance` | Allowed range: 0 to 4
`--generate-audio` |

## FLUX.1 Expand [pro]

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.1 Expand [pro] | image | none | `bfl/flux-pro-1.0-expand`

Expand an image by specifying the number of pixels to add on each side.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Supply the image to expand. Set at least one --expand-* margin. See https://api.bfl.ai/openapi.json. Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |
`--guidance-scale` | Allowed range: 1.5 to 100
`--steps` | Allowed range: 15 to 50
`--expand-top` | Allowed range: 0 to 2048
`--expand-bottom` | Allowed range: 0 to 2048
`--expand-left` | Allowed range: 0 to 2048
`--expand-right` | Allowed range: 0 to 2048

## FLUX.1 Fill [pro] Finetuned

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX.1 Fill [pro] Finetuned | image | none | `bfl/flux-pro-1.0-fill-finetuned`

Inpaint an image with an existing fine-tune and an encoded mask or image alpha channel.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Supply the image to inpaint. Use its alpha channel or provide --mask as raw base64 image data. Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |
`--guidance-scale` | Allowed range: 1.5 to 100
`--steps` | Allowed range: 15 to 50
`--finetune-id` | **Required.** Use an existing owned or shared fine-tune ID. See https://api.bfl.ai/openapi.json.
`--finetune-strength` | Allowed range: 0 to 2
`--mask` | Raw base64 mask image data; URLs and local filenames are not loaded. The mask must match the source dimensions. See https://api.bfl.ai/openapi.json.

## FLUX1.1 [pro] Ultra Finetuned

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX1.1 [pro] Ultra Finetuned | image | none | `bfl/flux-pro-1.1-ultra-finetuned`

Generate an image with an existing fine-tune and an optional image prompt.

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3; Ratios from 21:9 through 9:21 are supported.
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | Repeat maximum: 1; Optional reference image for generation. Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--strength` | Allowed range: 0 to 1
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 6
`--prompt-upsampling` |
`--finetune-id` | **Required.** Use an existing owned or shared fine-tune ID. See https://api.bfl.ai/openapi.json.
`--finetune-strength` | Allowed range: 0 to 2

## FLUX Erase

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX Erase | image | none | `bfl/flux-tools/erase-v1`

Remove objects identified by a mask from an input image.

This model does not require a prompt.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Supply the image to erase from. Also supply --mask. Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--mask` | **Required.** Public HTTP(S) URL or raw base64 mask image data. Local filenames are not loaded. See https://api.bfl.ai/openapi.json.
`--mask-dilation` | Allowed range: 0 to 25

## FLUX Virtual Try-On V1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX Virtual Try-On V1 | image | none | `bfl/flux-tools/vto-v1`

Dress the person in an input image using a garment image URL.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Supply the person image. Supply the clothing image with --garment-url. Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--garment-url` | **Required.** Public HTTP(S) URL of the garment image. See https://api.bfl.ai/openapi.json.

## FLUX Virtual Try-On V2

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX Virtual Try-On V2 | image | none | `bfl/flux-tools/vto-v2`

Dress the person in an input image using a garment image URL.

Option | Constraints
-------|------------
`--output-format` | Allowed values: jpeg, png, webp
`--input-media` | **Required.** Repeat maximum: 1; Supply the person image. Supply the clothing image with --garment-url. Takes images only; a video source fails the run, and frame prefixes are removed with a notice.
`--seed` |
`--safety-tolerance` | Allowed range: 0 to 5
`--garment-url` | **Required.** Public HTTP(S) URL of the garment image. See https://api.bfl.ai/openapi.json.

## FLUX 3 Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX 3 Image | image | none | `bfl/flux-3-image`

Generate images from text or edit up to ten reference images.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 21:9, 2:1, 16:9, 3:2, 7:5, 4:3, 5:4, 1:1, 4:5, 3:4, 5:7, 2:3, 9:16, 1:2, 9:21, auto; The API default is auto: a ratio requested in the prompt takes precedence, otherwise the first reference determines the framing. Without a reference, the prompt determines the ratio, falling back to 1:1.
`--resolution` | Allowed values: 768sq, 1k, 1.5k, 2k, 4k; The API default is 1k. These tiers specify pixel area; exact dimensions depend on the aspect ratio.
`--input-media` | Repeat maximum: 10; Accepts up to ten image files or public HTTP(S) URLs in the supplied order. Local images are sent as base64. Excess references are omitted with a notice. Videos fail the run; frame prefixes are removed with a notice.
`--safety-tolerance` | Allowed range: 0 to 4; The API default is 2; 0 applies the strictest moderation.

## FLUX Video Edit

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX Video Edit | video | none | `bfl/flux-tools/video-edit-v1`

Edit one video using a prompt while preserving its duration, resolution, aspect ratio, and audio.

Option | Constraints
-------|------------
`--input-media` | **Required.** Repeat maximum: 1; Supply a local MP4 or public HTTP(S) video URL. A run without a video fails before submission. Image inputs and frame prefixes are rejected. The API requires an edit prompt of 1 to 4096 characters after trimming whitespace. Duration, resolution, aspect ratio, and audio follow the source video; generation controls cannot override them. Output is MP4.
`--safety-tolerance` | Allowed range: 0 to 4; The API default is 2; 0 applies the strictest moderation. Sexual content remains limited to level 3 and hate content to level 2 regardless of this setting.

## FLUX Video Upscale

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
FLUX Video Upscale | video | none | `bfl/flux-tools/video-upscale-v1`

Upscale one video by 1.5 to 3 times, with an optional prompt to guide enhanced detail.

This model does not require a prompt.

Option | Constraints
-------|------------
`--input-media` | **Required.** Repeat maximum: 1; Supply a local MP4 or public HTTP(S) video URL. BFL documents a 50 MB limit for base64 MP4 input and a source limit of 2560 by 1440 pixels (3.7 megapixels per frame). Only the first 20 seconds are upscaled; slightly longer clips are shortened and much longer clips are rejected. A run without a video fails before submission. Image inputs and frame prefixes are rejected. A supplied prompt reaches the API; omitting it requests a neutral upscale. Output is MP4.
`--safety-tolerance` | Allowed range: 0 to 4; The API default is 2; 0 applies the strictest moderation. Sexual content remains limited to level 3 and hate content to level 2 regardless of this setting.
`--upscale-factor` | Allowed range: 1.5 to 3; The API default is 2. Output preserves the source aspect ratio and is capped at 13.75 megapixels per frame, which can reduce the actual scale below the requested factor.
`--creativity` | Allowed range: 0 to 1; 0 preserves the source closely; 1 permits creative detail enhancement. The API default is 1.

Revised 2026-10-06
