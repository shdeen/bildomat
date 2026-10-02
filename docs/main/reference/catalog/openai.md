# OpenAI models

Provider ID: `openai`. Credential: `api-keys.openai` in the configuration file, or else the `OPENAI_API_KEY` environment variable. Default model: `gpt-image-2.5-flare`. Provider documentation: <https://developers.openai.com/api/docs/guides/image-generation>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

Every OpenAI model takes images only as `--input-media`; a video source fails the run. See [provider input behavior](../input-media.md#provider-input-behavior).

## Models

- [`openai/gpt-image-2.5-flare`](#gpt-image-25-flare) — image.
- [`openai/gpt-image-2.5-sunburst`](#gpt-image-25-sunburst) — image.
- [`openai/gpt-image-2`](#gpt-image-2) — image.
- [`openai/gpt-image-1.5`](#gpt-image-15) — image.
- [`openai/gpt-image-1`](#gpt-image-1) — image.
- [`openai/chatgpt-image-latest`](#chatgpt-image-latest) — image.
- [`openai/gpt-image-1-mini`](#gpt-image-1-mini) — image.
- [`openai/sora-2`](#sora-2) — video.
- [`openai/sora-2-pro`](#sora-2-pro) — video.

## gpt-image-2.5-flare

GPT Image 2.5 Flare. Output: image.

Full key: `openai/gpt-image-2.5-flare`. Aliases: `gpt`, `gpt-2.5-flare`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `xhigh`, `max`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Maximum longer-to-shorter edge ratio: `3`; Maximum edge in pixels: `3840`; Minimum pixel count: `655360`; Maximum pixel count: `8294400`; Edge increment in pixels: `16`; Preferred derived long edge in pixels: `1536` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## gpt-image-2.5-sunburst

GPT Image 2.5 Sunburst. Output: image.

Full key: `openai/gpt-image-2.5-sunburst`. Aliases: `gpt-2.5-sunburst`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `xhigh`, `max`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Maximum longer-to-shorter edge ratio: `3`; Maximum edge in pixels: `3840`; Minimum pixel count: `655360`; Maximum pixel count: `8294400`; Edge increment in pixels: `16`; Preferred derived long edge in pixels: `1536` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## gpt-image-2

GPT Image 2. Output: image.

Full key: `openai/gpt-image-2`. Aliases: `gpt-2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Maximum longer-to-shorter edge ratio: `3`; Maximum edge in pixels: `3840`; Minimum pixel count: `655360`; Maximum pixel count: `8294400`; Edge increment in pixels: `16`; Preferred derived long edge in pixels: `1536` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## gpt-image-1.5

GPT Image 1.5. Output: image.

Full key: `openai/gpt-image-1.5`. Aliases: `gpt-1.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Allowed: `1024x1024`, `1024x1536`, `1536x1024` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## gpt-image-1

GPT Image 1. Output: image.

Full key: `openai/gpt-image-1`. Aliases: `gpt-1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Allowed: `1024x1024`, `1024x1536`, `1536x1024` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## chatgpt-image-latest

ChatGPT Image. Output: image.

Full key: `openai/chatgpt-image-latest`. Aliases: `chatgpt`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Allowed: `1024x1024`, `1024x1536`, `1536x1024` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## gpt-image-1-mini

GPT Image 1 Mini. Output: image.

Full key: `openai/gpt-image-1-mini`. Aliases: `gpt-1-mini`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--size` | Optional | Allowed: `1024x1024`, `1024x1536`, `1536x1024` |
| `--background` | Optional | Allowed: `transparent`, `opaque`, `auto` |
| `--moderation-level` | Optional | Allowed: `low`, `auto` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## sora-2

Sora 2. Output: video.

Full key: `openai/sora-2`. Aliases: `sora`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--duration` | Optional | Allowed: `4`, `8`, `12`, `16`, `20` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--size` | Optional | Allowed: `1280x720`, `720x1280` |

## sora-2-pro

Sora 2 Pro. Output: video.

Full key: `openai/sora-2-pro`. Aliases: `sora-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--duration` | Optional | Allowed: `4`, `8`, `12`, `16`, `20` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--size` | Optional | Allowed: `1280x720`, `720x1280`, `1792x1024`, `1024x1792`, `1920x1080`, `1080x1920` |

Revised 2026-10-01
