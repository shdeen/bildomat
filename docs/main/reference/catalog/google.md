# Google models

Provider ID: `google`. Credential: `api-keys.google` in the configuration file, or else the `GOOGLE_API_KEY` environment variable. Default model: `gemini-3.1-flash-image`. Provider documentation: <https://ai.google.dev/gemini-api/docs/image-generation>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

## Models

- [`google/veo-3.1-generate-preview`](#veo-31-generate-preview) — video.
- [`google/veo-3.1-fast-generate-preview`](#veo-31-fast-generate-preview) — video.
- [`google/veo-3.1-lite-generate-preview`](#veo-31-lite-generate-preview) — video.
- [`google/gemini-3-pro-image`](#gemini-3-pro-image) — image.
- [`google/gemini-3.1-flash-image`](#gemini-31-flash-image) — image.
- [`google/gemini-3.1-flash-lite-image`](#gemini-31-flash-lite-image) — image.
- [`google/gemini-2.5-flash-image`](#gemini-25-flash-image) — image.
- [`google/gemini-omni-1.1-flash`](#gemini-omni-11-flash) — video.
- [`google/gemini-omni-flash-preview`](#gemini-omni-flash-preview) — video.

## veo-3.1-generate-preview

Veo 3.1. Output: video.

Full key: `google/veo-3.1-generate-preview`. Family: `veo`. Aliases: `veo`, `veo-3.1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p`, `4k` |
| `--duration` | Optional | Allowed: `4`, `6`, `8`; Rule: forced to 8 with any input media or resolution 1080p or 4k |
| `--input-media` | Optional | Maximum inputs: `3`; Use `first:` for an opening image and `last:` for a closing image. A closing image requires an opening image. Unprefixed images are references, not opening frames. Alternatively, supply one video to extend it; images and a video cannot be combined. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--person-generation` | Optional | Google accepts allow_all for text-to-video and video extension, and only allow_adult for image-to-video, interpolation, and reference-image requests; in the EU, the UK, Switzerland, and the MENA regions only allow_adult is accepted. See [Google’s video guidance](https://ai.google.dev/gemini-api/docs/veo). |
| `--negative-prompt` | Optional | No declared value constraint; see the general flag and input rules. |

## veo-3.1-fast-generate-preview

Veo 3.1 fast. Output: video.

Full key: `google/veo-3.1-fast-generate-preview`. Family: `veo`. Aliases: `veo-3.1-fast`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p`, `4k` |
| `--duration` | Optional | Allowed: `4`, `6`, `8`; Rule: forced to 8 with any input media or resolution 1080p or 4k |
| `--input-media` | Optional | Maximum inputs: `3`; Use `first:` for an opening image and `last:` for a closing image. A closing image requires an opening image. Unprefixed images are references, not opening frames. Alternatively, supply one video to extend it; images and a video cannot be combined. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--person-generation` | Optional | Google accepts allow_all for text-to-video and video extension, and only allow_adult for image-to-video, interpolation, and reference-image requests; in the EU, the UK, Switzerland, and the MENA regions only allow_adult is accepted. See [Google’s video guidance](https://ai.google.dev/gemini-api/docs/veo). |
| `--negative-prompt` | Optional | No declared value constraint; see the general flag and input rules. |

## veo-3.1-lite-generate-preview

Veo 3.1 lite. Output: video.

Full key: `google/veo-3.1-lite-generate-preview`. Family: `veo`. Aliases: `veo-3.1-lite`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `4`, `6`, `8`; Rule: forced to 8 with any input media or resolution 1080p |
| `--input-media` | Optional | Maximum inputs: `1`. An unprefixed image or `first:` selects the opening frame. A closing image requires an opening image, so the one-input cap prevents using `last:` successfully. Bildomat can submit one video for extension instead; the provider determines whether Lite accepts that operation. See [frame and input rules](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--person-generation` | Optional | Use `allow_all` for text-to-video where permitted, or `allow_adult` for image-based requests. In the EU, UK, Switzerland, and MENA regions, only `allow_adult` is accepted. See [Google’s video guidance](https://ai.google.dev/gemini-api/docs/veo). |

## gemini-3-pro-image

Gemini 3 Pro Image. Output: image.

Full key: `google/gemini-3-pro-image`. Aliases: `gemini-3-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--thinking-level` | Optional | Allowed: `low`, `high` |
| `--include-thoughts` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `14` |

## gemini-3.1-flash-image

Gemini 3.1 Flash Image. Output: image.

Full key: `google/gemini-3.1-flash-image`. Aliases: `gemini`, `gemini-3.1`, `gemini-3.1-flash`, `gemini-flash`, `nano-banana`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:4`, `1:8`, `2:3`, `3:2`, `3:4`, `4:1`, `4:3`, `4:5`, `5:4`, `8:1`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `512`, `1K`, `2K`, `4K` |
| `--thinking-level` | Optional | Allowed: `minimal`, `high` |
| `--include-thoughts` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `14` |

## gemini-3.1-flash-lite-image

Gemini 3.1 Flash Lite Image. Output: image.

Full key: `google/gemini-3.1-flash-lite-image`. Aliases: `gemini-3.1-lite`, `gemini-3.1-flash-lite`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `1K` |
| `--thinking-level` | Optional | Allowed: `low`, `high` |
| `--include-thoughts` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `14` |

## gemini-2.5-flash-image

Gemini 2.5 Flash Preview Image. Output: image.

Full key: `google/gemini-2.5-flash-image`. Aliases: `gemini-2.5`, `gemini-2.5-flash`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |
| `--input-media` | Optional | No declared value constraint; see the general flag and input rules. |

## gemini-omni-1.1-flash

Gemini Omni 1.1 Flash. Output: video.

Full key: `google/gemini-omni-1.1-flash`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `360p`, `720p`, `1080p`, `4k` |
| `--input-media` | Optional | No declared value constraint; see the general flag and input rules. |

## gemini-omni-flash-preview

Gemini Omni Flash Preview. Output: video.

Full key: `google/gemini-omni-flash-preview`. Aliases: `gemini-omni`, `gemini-omni-flash`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--input-media` | Optional | No declared value constraint; see the general flag and input rules. |

Revised 2026-10-01
