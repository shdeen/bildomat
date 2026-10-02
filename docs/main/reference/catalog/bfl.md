# Black Forest Labs models

Provider ID: `bfl`. Credential: `api-keys.bfl` in the configuration file, or else the `BFL_API_KEY` environment variable. Default model: `flux-2-pro`. Provider documentation: <https://docs.bfl.ai/>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

Image models that accept `--input-media` take images only; a video source fails the run, and frame prefixes are removed with a notice. See [provider input behavior](../input-media.md#provider-input-behavior).

## Models

- [`bfl/flux-2-pro`](#flux-2-pro) — image.
- [`bfl/flux-2-pro-preview`](#flux-2-pro-preview) — image.
- [`bfl/flux-2-max`](#flux-2-max) — image.
- [`bfl/flux-2-flex`](#flux-2-flex) — image.
- [`bfl/flux-2-klein-4b`](#flux-2-klein-4b) — image.
- [`bfl/flux-2-klein-9b`](#flux-2-klein-9b) — image.
- [`bfl/flux-2-klein-9b-preview`](#flux-2-klein-9b-preview) — image.
- [`bfl/flux-pro-1.1`](#flux-pro-11) — image.
- [`bfl/flux-dev`](#flux-dev) — image.
- [`bfl/flux-pro-1.1-ultra`](#flux-pro-11-ultra) — image.
- [`bfl/flux-pro-1.0-fill`](#flux-pro-10-fill) — image.
- [`bfl/flux-kontext-pro`](#flux-kontext-pro) — image.
- [`bfl/flux-kontext-max`](#flux-kontext-max) — image.
- [`bfl/flux-tools/outpainting-v1`](#flux-toolsoutpainting-v1) — image.
- [`bfl/flux-tools/deblur-v1`](#flux-toolsdeblur-v1) — image.
- [`bfl/flux-3-video`](#flux-3-video) — video.
- [`bfl/flux-pro-1.0-expand`](#flux-pro-10-expand) — image.
- [`bfl/flux-pro-1.0-fill-finetuned`](#flux-pro-10-fill-finetuned) — image.
- [`bfl/flux-pro-1.1-ultra-finetuned`](#flux-pro-11-ultra-finetuned) — image.
- [`bfl/flux-tools/erase-v1`](#flux-toolserase-v1) — image.
- [`bfl/flux-tools/vto-v1`](#flux-toolsvto-v1) — image.
- [`bfl/flux-tools/vto-v2`](#flux-toolsvto-v2) — image.

## flux-2-pro

FLUX.2 [pro]. Output: image.

Full key: `bfl/flux-2-pro`. Aliases: `flux.2-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--disable-prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-2-pro-preview

FLUX.2 [pro] (preview). Output: image.

Full key: `bfl/flux-2-pro-preview`. Aliases: `flux.2-pro-preview`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--disable-prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-2-max

FLUX.2 [max]. Output: image.

Full key: `bfl/flux-2-max`. Aliases: `flux.2-max`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--disable-prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-2-flex

FLUX.2 [flex]. Output: image.

Full key: `bfl/flux-2-flex`. Aliases: `flux.2-flex`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--guidance-scale` | Optional | Minimum: `1.5`; Maximum: `10` |
| `--steps` | Optional | Minimum: `1`; Maximum: `50` |

## flux-2-klein-4b

FLUX.2 [klein] 4B. Output: image.

Full key: `bfl/flux-2-klein-4b`. Aliases: `flux.2-klein-4b`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |

## flux-2-klein-9b

FLUX.2 [klein] 9B. Output: image.

Full key: `bfl/flux-2-klein-9b`. Aliases: `flux.2-klein-9b`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |

## flux-2-klein-9b-preview

FLUX.2 [klein] 9B (preview). Output: image.

Full key: `bfl/flux-2-klein-9b-preview`. Aliases: `flux.2-klein-9b-preview`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `64`; Maximum pixel count: `4194304`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |

## flux-pro-1.1

FLUX1.1 [pro]. Output: image.

Full key: `bfl/flux-pro-1.1`. Aliases: `flux1.1-pro`, `flux.1.1-pro`, `flux-1.1-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `256`; Maximum edge in pixels: `1440`; Edge increment in pixels: `32`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-dev

FLUX.1 [dev]. Output: image.

Full key: `bfl/flux-dev`. Aliases: `flux.1-dev`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--size` | Optional | Minimum edge in pixels: `256`; Maximum edge in pixels: `1440`; Edge increment in pixels: `32`; Preferred derived long edge in pixels: `1024` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--guidance-scale` | Optional | Minimum: `1.5`; Maximum: `5` |
| `--steps` | Optional | Minimum: `1`; Maximum: `50` |

## flux-pro-1.1-ultra

FLUX1.1 [pro] ultra. Output: image.

Full key: `bfl/flux-pro-1.1-ultra`. Aliases: `flux1.1-pro-ultra`, `flux.1.1-pro-ultra`, `flux-1.1-pro-ultra`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-pro-1.0-fill

FLUX.1 Fill [pro]. Output: image.

Full key: `bfl/flux-pro-1.0-fill`. Aliases: `flux.1-fill-pro`, `flux-1-fill-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1` |
| `--steps` | Optional | Minimum: `15`; Maximum: `50` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--guidance-scale` | Optional | Minimum: `1.5`; Maximum: `100` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |

## flux-kontext-pro

FLUX.1 Kontext [pro]. Output: image.

Full key: `bfl/flux-kontext-pro`. Aliases: `flux.1-kontext-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-kontext-max

FLUX.1 Kontext [max]. Output: image.

Full key: `bfl/flux-kontext-max`. Aliases: `flux.1-kontext-max`, `flux-1-kontext-max`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-tools/outpainting-v1

FLUX Outpainting. Output: image.

Full key: `bfl/flux-tools/outpainting-v1`. Aliases: `flux-outpainting-v1`, `flux-outpainting`. This model reads no prompt; run it without one.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Required | Minimum edge in pixels: `64`; Maximum pixel count: `4194304` |
| `--input-media` | Required | Maximum inputs: `1` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--disable-prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |

## flux-tools/deblur-v1

FLUX.2 Deblur. Output: image.

Full key: `bfl/flux-tools/deblur-v1`. Aliases: `flux-deblur-v1`, `flux-deblur`. This model reads no prompt; run it without one.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

## flux-3-video

FLUX.3 Video. Output: video.

Full key: `bfl/flux-3-video`. Aliases: `flux-3`, `flux.3-video`, `flux-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `auto`, `21:9`, `2:1`, `16:9`, `4:3`, `1:1`, `3:4`, `9:16` |
| `--resolution` | Optional | Allowed: `hd`, `fhd` |
| `--duration` | Optional | Minimum: `5`; Maximum: `20` |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `4` |
| `--input-media` | Optional | Maximum inputs: `10`; Images are keyframes: a numeric time can be set on any image and requires `--duration`. Alternatively, supply one untimed video to continue it; images and a video cannot be combined. See [timed keyframes](../input-media.md#bfl-timed-keyframes). |

## flux-pro-1.0-expand

FLUX.1 Expand [pro]. Output: image.

Full key: `bfl/flux-pro-1.0-expand`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply the image to expand. Set at least one --expand-* margin. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--expand-top` | Optional | Minimum: `0`; Maximum: `2048` |
| `--expand-bottom` | Optional | Minimum: `0`; Maximum: `2048` |
| `--expand-left` | Optional | Minimum: `0`; Maximum: `2048` |
| `--expand-right` | Optional | Minimum: `0`; Maximum: `2048` |
| `--steps` | Optional | Minimum: `15`; Maximum: `50` |
| `--guidance-scale` | Optional | Minimum: `1.5`; Maximum: `100` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

## flux-pro-1.0-fill-finetuned

FLUX.1 Fill [pro] Finetuned. Output: image.

Full key: `bfl/flux-pro-1.0-fill-finetuned`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply the image to inpaint. Use its alpha channel or provide --mask as raw base64 image data. |
| `--finetune-id` | Required | Use an existing owned or shared fine-tune ID. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--finetune-strength` | Optional | Minimum: `0`; Maximum: `2` |
| `--mask` | Optional | Raw base64 mask image data; URLs and local filenames are not loaded. The mask must match the source dimensions. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--steps` | Optional | Minimum: `15`; Maximum: `50` |
| `--guidance-scale` | Optional | Minimum: `1.5`; Maximum: `100` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

## flux-pro-1.1-ultra-finetuned

FLUX1.1 [pro] Ultra Finetuned. Output: image.

Full key: `bfl/flux-pro-1.1-ultra-finetuned`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--finetune-id` | Required | Use an existing owned or shared fine-tune ID. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--finetune-strength` | Optional | Minimum: `0`; Maximum: `2` |
| `--aspect-ratio` | Optional | Ratios from 21:9 through 9:21 are supported. |
| `--input-media` | Optional | Maximum inputs: `1`; Optional reference image for generation. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

## flux-tools/erase-v1

FLUX Erase. Output: image.

Full key: `bfl/flux-tools/erase-v1`. This model reads no prompt; run it without one.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply the image to erase from. Also supply --mask. |
| `--mask` | Required | Public HTTP(S) URL or raw base64 mask image data. Local filenames are not loaded. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--mask-dilation` | Optional | Minimum: `0`; Maximum: `25` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

## flux-tools/vto-v1

FLUX Virtual Try-On V1. Output: image.

Full key: `bfl/flux-tools/vto-v1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply the person image. Supply the clothing image with --garment-url. |
| `--garment-url` | Required | Public HTTP(S) URL of the garment image. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

## flux-tools/vto-v2

FLUX Virtual Try-On V2. Output: image.

Full key: `bfl/flux-tools/vto-v2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply the person image. Supply the clothing image with --garment-url. |
| `--garment-url` | Required | Public HTTP(S) URL of the garment image. See [BFL’s API reference](https://api.bfl.ai/openapi.json). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |
| `--output-format` | Optional | Allowed: `jpeg`, `png`, `webp` |

Revised 2026-10-01
