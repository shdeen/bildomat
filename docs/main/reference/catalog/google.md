# Google Models

Every model that `bild` offers from Google (provider ID `google`, key variable `GOOGLE_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info google/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

4 image models, 5 video models.

## Contents

- [Veo 3.1](#veo-31)
- [Veo 3.1 fast](#veo-31-fast)
- [Veo 3.1 lite](#veo-31-lite)
- [Gemini 3 Pro Image](#gemini-3-pro-image)
- [Gemini 3.1 Flash Image](#gemini-31-flash-image)
- [Gemini 3.1 Flash Lite Image](#gemini-31-flash-lite-image)
- [Gemini 2.5 Flash Preview Image](#gemini-25-flash-preview-image)
- [Gemini Omni 1.1 Flash](#gemini-omni-11-flash)
- [Gemini Omni Flash Preview](#gemini-omni-flash-preview)

## Veo 3.1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Veo 3.1 | video | `veo`, `veo-3.1` | `google/veo-3.1-generate-preview`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p, 4k
`--duration` | Allowed values: 4, 6, 8; Rule: forced to 8 when reference images are supplied, when extending a video, or when the resolution is 1080p or 4k
`--input-media` | Repeat maximum: 3; Veo 3.1 special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` | 
`--person-generation` | Google accepts allow_all for text-to-video and video extension, and only allow_adult for image-to-video, interpolation, and reference-image requests; in the EU, the UK, Switzerland, and the MENA regions only allow_adult is accepted.
`--negative-prompt` | 

## Veo 3.1 fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Veo 3.1 fast | video | `veo-3.1-fast` | `google/veo-3.1-fast-generate-preview`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p, 4k
`--duration` | Allowed values: 4, 6, 8; Rule: forced to 8 when reference images are supplied, when extending a video, or when the resolution is 1080p or 4k
`--input-media` | Repeat maximum: 3; Veo 3.1 family special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` | 
`--person-generation` | Google accepts allow_all for text-to-video and video extension, and only allow_adult for image-to-video, interpolation, and reference-image requests; in the EU, the UK, Switzerland, and the MENA regions only allow_adult is accepted.
`--negative-prompt` | 

## Veo 3.1 lite

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Veo 3.1 lite | video | `veo-3.1-lite` | `google/veo-3.1-lite-generate-preview`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 4, 6, 8; Rule: forced to 8 when reference images are supplied or the resolution is 1080p
`--input-media` | Repeat maximum: 1; Veo 3.1 family special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` | 
`--person-generation` | Google accepts allow_all for text-to-video and only allow_adult for image-to-video, interpolation, and reference-image requests; in the EU, the UK, Switzerland, and the MENA regions only allow_adult is accepted.

## Gemini 3 Pro Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Gemini 3 Pro Image | image | `gemini-3-pro` | `google/gemini-3-pro-image`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 1K, 2K, 4K
`--thinking-level` | Allowed values: low, high
`--include-thoughts` | 
`--input-media` | Repeat maximum: 14

## Gemini 3.1 Flash Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Gemini 3.1 Flash Image | image | `gemini`, `gemini-3.1`, `gemini-3.1-flash`, `gemini-flash`, `nano-banana` | `google/gemini-3.1-flash-image`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:4, 1:8, 2:3, 3:2, 3:4, 4:1, 4:3, 4:5, 5:4, 8:1, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 512, 1K, 2K, 4K
`--thinking-level` | Allowed values: minimal, high
`--include-thoughts` | 
`--input-media` | Repeat maximum: 14

## Gemini 3.1 Flash Lite Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Gemini 3.1 Flash Lite Image | image | `gemini-3.1-lite`, `gemini-3.1-flash-lite` | `google/gemini-3.1-flash-lite-image`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 1K
`--thinking-level` | Allowed values: low, high
`--include-thoughts` | 
`--input-media` | Repeat maximum: 14

## Gemini 2.5 Flash Preview Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Gemini 2.5 Flash Preview Image | image | `gemini-2.5`, `gemini-2.5-flash` | `google/gemini-2.5-flash-image`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
`--input-media` | 

## Gemini Omni 1.1 Flash

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Gemini Omni 1.1 Flash | video | none | `google/gemini-omni-1.1-flash`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 360p, 720p, 1080p, 4k
`--input-media` | 

## Gemini Omni Flash Preview

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Gemini Omni Flash Preview | video | `gemini-omni`, `gemini-omni-flash` | `google/gemini-omni-flash-preview`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--input-media` | 

Revised 2026-10-05
