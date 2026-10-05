# Kling Models

Every model that `bild` offers from Kling (provider ID `kling`, key variable `KLING_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info kling/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

4 image models, 6 video models.

## Contents

- [Kling Image 3.0](#kling-image-30)
- [Kling Image 3.0 Omni](#kling-image-30-omni)
- [Kling Image O1](#kling-image-o1)
- [Kling Image 2.1](#kling-image-21)
- [Kling 3.0](#kling-30)
- [Kling 3.0 Turbo](#kling-30-turbo)
- [Kling 3.0 Omni](#kling-30-omni)
- [Kling O1](#kling-o1)
- [Kling 2.6](#kling-26)
- [Kling 2.5 Turbo](#kling-25-turbo)

## Kling Image 3.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling Image 3.0 | image | `kling-3` | `kling/kling-v3`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 3:2, 2:3, 21:9
`--resolution` | Allowed values: 1k, 2k
`--num-images` | Allowed range: 1 to 9
`--input-media` | Repeat maximum: 1
`--negative-prompt` | Kling refuses --negative-prompt when --input-media is supplied.

## Kling Image 3.0 Omni

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling Image 3.0 Omni | image | `kling-3-omni` | `kling/kling-v3-omni`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 3:2, 2:3, 21:9, auto
`--resolution` | Allowed values: 1k, 2k, 4k
`--num-images` | Allowed range: 1 to 9
`--input-media` | Repeat maximum: 10

## Kling Image O1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling Image O1 | image | `kling-1` | `kling/kling-image-o1`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 3:2, 2:3, 21:9, auto
`--resolution` | Allowed values: 1k, 2k
`--num-images` | Allowed range: 1 to 9
`--input-media` | Repeat maximum: 10

## Kling Image 2.1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling Image 2.1 | image | `kling-2.1` | `kling/kling-v2-1`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 3:2, 2:3, 21:9
`--resolution` | Allowed values: 1k, 2k
`--num-images` | Allowed range: 1 to 9
`--input-media` | Repeat maximum: 1
`--negative-prompt` | Kling refuses --negative-prompt when --input-media is supplied.

## Kling 3.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling 3.0 | video | `kling-3-video`, `kling-video` | `kling/kling-3.0`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p, 4k
`--duration` | Allowed range: 3 to 15
`--input-media` | Repeat maximum: 2
`--generate-audio` | 

## Kling 3.0 Turbo

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling 3.0 Turbo | video | `kling-3-turbo-video` | `kling/kling-3.0-turbo`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed range: 3 to 15
`--input-media` | Repeat maximum: 1

## Kling 3.0 Omni

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling 3.0 Omni | video | `kling-3-omni-video` | `kling/kling-3.0-omni`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p, 4k
`--duration` | Allowed range: 3 to 15
`--input-media` | Repeat maximum: 7
`--generate-audio` | 

## Kling O1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling O1 | video | `kling-o1-video`, `kling-1-video` | `kling/kling-o1`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed range: 3 to 10
`--input-media` | Repeat maximum: 7

## Kling 2.6

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling 2.6 | video | `kling-2.6-video` | `kling/kling-2.6`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 5, 10
`--input-media` | Repeat maximum: 2
`--generate-audio` | Kling 2.6 requires 1080p for generated audio and frame input.

## Kling 2.5 Turbo

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling 2.5 Turbo | video | `kling-2.5-turbo-video` | `kling/kling-2.5-turbo`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 5, 10
`--input-media` | Repeat maximum: 1

Revised 2026-10-05
