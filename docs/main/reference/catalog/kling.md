# Kling models

Provider ID: `kling`. Credential: `api-keys.kling` in the configuration file, or else the `KLING_API_KEY` environment variable. Default model: `kling-v3`. Provider documentation: <https://kling.ai/document-api>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

Every Kling model takes images only as `--input-media`. Bildomat rejects a video source before submission: a local MP4, or a URL identified as video. Image models remove frame prefixes with a notice. On `kling-3.0`, `kling-3.0-turbo`, `kling-2.6`, and `kling-2.5-turbo`, `first:` and `last:` select the opening and closing frames, numeric times map to those positions, and unprefixed images fill the remaining positions in order. On `kling-3.0-omni` and `kling-o1`, `first:`, `last:`, and numeric times select the opening and closing frames in the same way, and unprefixed images are references. See [opening and closing frames](../input-media.md#opening-and-closing-frames) and [provider input behavior](../input-media.md#provider-input-behavior).

## Models

- [`kling/kling-v3`](#kling-v3) — image.
- [`kling/kling-v3-omni`](#kling-v3-omni) — image.
- [`kling/kling-image-o1`](#kling-image-o1) — image.
- [`kling/kling-v2-1`](#kling-v2-1) — image.
- [`kling/kling-3.0`](#kling-30) — video.
- [`kling/kling-3.0-turbo`](#kling-30-turbo) — video.
- [`kling/kling-3.0-omni`](#kling-30-omni) — video.
- [`kling/kling-o1`](#kling-o1) — video.
- [`kling/kling-2.6`](#kling-26) — video.
- [`kling/kling-2.5-turbo`](#kling-25-turbo) — video.

## kling-v3

Kling Image 3.0. Output: image.

Full key: `kling/kling-v3`. Aliases: `kling-3`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `21:9` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `9` |
| `--negative-prompt` | Optional | Kling refuses --negative-prompt when --input-media is supplied. |
| `--input-media` | Optional | Maximum inputs: `1` |

## kling-v3-omni

Kling Image 3.0 Omni. Output: image.

Full key: `kling/kling-v3-omni`. Family: `omni`. Aliases: `kling-3-omni`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1k`, `2k`, `4k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `9` |
| `--input-media` | Optional | Maximum inputs: `10` |

## kling-image-o1

Kling Image O1. Output: image.

Full key: `kling/kling-image-o1`. Family: `omni`. Aliases: `kling-1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `9` |
| `--input-media` | Optional | Maximum inputs: `10` |

## kling-v2-1

Kling Image 2.1. Output: image.

Full key: `kling/kling-v2-1`. Aliases: `kling-2.1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `21:9` |
| `--resolution` | Optional | Allowed: `1k`, `2k` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `9` |
| `--negative-prompt` | Optional | Kling refuses --negative-prompt when --input-media is supplied. |
| `--input-media` | Optional | Maximum inputs: `1` |

## kling-3.0

Kling 3.0. Output: video.

Full key: `kling/kling-3.0`. Aliases: `kling-3-video`, `kling-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--resolution` | Optional | Allowed: `720p`, `1080p`, `4k` |
| `--duration` | Optional | Minimum: `3`; Maximum: `15` |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `2` |

## kling-3.0-turbo

Kling 3.0 Turbo. Output: video.

Full key: `kling/kling-3.0-turbo`. Aliases: `kling-3-turbo-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Minimum: `3`; Maximum: `15` |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--input-media` | Optional | Maximum inputs: `1` |

## kling-3.0-omni

Kling 3.0 Omni. Output: video.

Full key: `kling/kling-3.0-omni`. Family: `omni`. Aliases: `kling-3-omni-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--resolution` | Optional | Allowed: `720p`, `1080p`, `4k` |
| `--duration` | Optional | Minimum: `3`; Maximum: `15` |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `7` |

## kling-o1

Kling O1. Output: video.

Full key: `kling/kling-o1`. Family: `omni`. Aliases: `kling-o1-video`, `kling-1-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Minimum: `3`; Maximum: `10` |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--input-media` | Optional | Maximum inputs: `7` |

## kling-2.6

Kling 2.6. Output: video.

Full key: `kling/kling-2.6`. Aliases: `kling-2.6-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `5`, `10` |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--generate-audio` | Optional | Kling 2.6 requires 1080p for generated audio and frame input. |
| `--input-media` | Optional | Maximum inputs: `2` |

## kling-2.5-turbo

Kling 2.5 Turbo. Output: video.

Full key: `kling/kling-2.5-turbo`. Aliases: `kling-2.5-turbo-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `5`, `10` |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--input-media` | Optional | Maximum inputs: `1` |

Revised 2026-10-01
