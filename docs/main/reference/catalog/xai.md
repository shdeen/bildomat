# xAI Models

Every model that `bild` offers from xAI (provider ID `xai`, API key variable `XAI_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info xai/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

4 image models, 3 video models.

## Contents

- [Grok Imagine Image Quality](#grok-imagine-image-quality)
- [Grok Imagine Image Quality Latest](#grok-imagine-image-quality-latest)
- [Grok Imagine Image](#grok-imagine-image)
- [Grok Imagine Image 2.0](#grok-imagine-image-20)
- [Grok Imagine Video](#grok-imagine-video)
- [Grok Imagine Video 1.5](#grok-imagine-video-15)
- [Grok Imagine Video 1.5 Lite](#grok-imagine-video-15-lite)

## Grok Imagine Image Quality

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Image Quality | image | `grok` | `xai/grok-imagine-image-quality`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 2:1, 1:2, 19.5:9, 9:19.5, 20:9, 9:20, auto
`--resolution` | Allowed values: 1k, 2k
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 3; Takes images only; a video source fails the run.

## Grok Imagine Image Quality Latest

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Image Quality Latest | image | `grok-latest` | `xai/grok-imagine-image-quality-latest`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 2:1, 1:2, 19.5:9, 9:19.5, 20:9, 9:20, auto
`--resolution` | Allowed values: 1k, 2k
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 3; Takes images only; a video source fails the run.

## Grok Imagine Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Image | image | none | `xai/grok-imagine-image`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 2:1, 1:2, 19.5:9, 9:19.5, 20:9, 9:20, auto
`--resolution` | Allowed values: 1k, 2k
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 3; Takes images only; a video source fails the run.

## Grok Imagine Image 2.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Image 2.0 | image | `grok-2.0`, `grok-2` | `xai/grok-imagine-image-2.0`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3, 2:1, 1:2, 19.5:9, 9:19.5, 20:9, 9:20, auto
`--resolution` | Allowed values: 1k, 2k
`--quality` | Allowed values: low, medium
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 3; Takes images only; a video source fails the run.

## Grok Imagine Video

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Video | video | `grok-video` | `xai/grok-imagine-video`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3
`--resolution` | Allowed values: 480p, 720p
`--duration` | Allowed range: 1 to 15
`--input-media` | Repeat maximum: 1; Takes images only; a video source fails the run.

## Grok Imagine Video 1.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Video 1.5 | video | none | `xai/grok-imagine-video-1.5`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3
`--resolution` | Allowed values: 480p, 720p, 1080p
`--duration` | Allowed range: 1 to 15
`--input-media` | Repeat maximum: 1; Takes images only; a video source fails the run.

## Grok Imagine Video 1.5 Lite

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Grok Imagine Video 1.5 Lite | video | none | `xai/grok-imagine-video-1.5-lite`

Generate video from a text prompt or one reference image.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 16:9, 9:16, 4:3, 3:4, 3:2, 2:3
`--resolution` | Allowed values: 480p, 720p, 1080p
`--duration` | Allowed range: 1 to 15
`--input-media` | Repeat maximum: 1; Supply one image for image-to-video generation. Takes images only; a video source fails the run.

Revised 2026-10-06
