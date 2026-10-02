# xAI models

Provider ID: `xai`. Credential: `api-keys.xai` in the configuration file, or else the `XAI_API_KEY` environment variable. Default model: `grok-imagine-image`. Provider documentation: <https://docs.x.ai/developers/model-capabilities/images/generation>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

Every xAI model, including the video models, takes images only as `--input-media`; a video source fails the run. See [provider input behavior](../input-media.md#provider-input-behavior).

## Models

- [`xai/grok-imagine-image-quality`](#grok-imagine-image-quality) — image.
- [`xai/grok-imagine-image-quality-latest`](#grok-imagine-image-quality-latest) — image.
- [`xai/grok-imagine-image`](#grok-imagine-image) — image.
- [`xai/grok-imagine-image-2.0`](#grok-imagine-image-20) — image.
- [`xai/grok-imagine-video`](#grok-imagine-video) — video.
- [`xai/grok-imagine-video-1.5`](#grok-imagine-video-15) — video.

## grok-imagine-image-quality

Grok Imagine Image Quality. Output: image.

Full key: `xai/grok-imagine-image-quality`. Aliases: `grok`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `2:1`, `1:2`, `19.5:9`, `9:19.5`, `20:9`, `9:20`, `auto` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--input-media` | Optional | Maximum inputs: `3` |

## grok-imagine-image-quality-latest

Grok Imagine Image Quality Latest. Output: image.

Full key: `xai/grok-imagine-image-quality-latest`. Aliases: `grok-latest`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `2:1`, `1:2`, `19.5:9`, `9:19.5`, `20:9`, `9:20`, `auto` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--input-media` | Optional | Maximum inputs: `3` |

## grok-imagine-image

Grok Imagine Image. Output: image.

Full key: `xai/grok-imagine-image`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `2:1`, `1:2`, `19.5:9`, `9:19.5`, `20:9`, `9:20`, `auto` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--input-media` | Optional | Maximum inputs: `3` |

## grok-imagine-image-2.0

Grok Imagine Image 2.0. Output: image.

Full key: `xai/grok-imagine-image-2.0`. Aliases: `grok-2.0`, `grok-2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3`, `2:1`, `1:2`, `19.5:9`, `9:19.5`, `20:9`, `9:20`, `auto` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--quality` | Optional | Allowed: `low`, `medium` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--input-media` | Optional | Maximum inputs: `3` |

## grok-imagine-video

Grok Imagine Video. Output: video.

Full key: `xai/grok-imagine-video`. Aliases: `grok-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3` |
| `--resolution` | Optional | Allowed: `480p`, `720p` |
| `--duration` | Optional | Minimum: `1`; Maximum: `15` |
| `--input-media` | Optional | Maximum inputs: `1` |

## grok-imagine-video-1.5

Grok Imagine Video 1.5. Output: video.

Full key: `xai/grok-imagine-video-1.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `3:2`, `2:3` |
| `--resolution` | Optional | Allowed: `480p`, `720p`, `1080p` |
| `--duration` | Optional | Minimum: `1`; Maximum: `15` |
| `--input-media` | Optional | Maximum inputs: `1` |

Revised 2026-10-01
