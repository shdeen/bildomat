# OpenRouter models

Provider ID: `openrouter`. Credential: `api-keys.openrouter` in the configuration file, or else the `OPENROUTER_API_KEY` environment variable. Default model: `google/gemini-3.1-flash-image`. Provider documentation: <https://openrouter.ai/docs>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

## Models

- [`openrouter/openai/gpt-image-2.5-flare`](#openaigpt-image-25-flare) — image.
- [`openrouter/openai/gpt-image-2.5-sunburst`](#openaigpt-image-25-sunburst) — image.
- [`openrouter/black-forest-labs/flux.2-flex`](#black-forest-labsflux2-flex) — image.
- [`openrouter/black-forest-labs/flux.2-klein-4b`](#black-forest-labsflux2-klein-4b) — image.
- [`openrouter/black-forest-labs/flux.2-max`](#black-forest-labsflux2-max) — image.
- [`openrouter/black-forest-labs/flux.2-pro`](#black-forest-labsflux2-pro) — image.
- [`openrouter/black-forest-labs/flux-3-video`](#black-forest-labsflux-3-video) — video.
- [`openrouter/bytedance-seed/seedream-4.5`](#bytedance-seedseedream-45) — image.
- [`openrouter/bytedance-seed/seedream-5-0-pro`](#bytedance-seedseedream-5-0-pro) — image.
- [`openrouter/bytedance-seed/seedream-5-0-lite`](#bytedance-seedseedream-5-0-lite) — image.
- [`openrouter/google/gemini-2.5-flash-image`](#googlegemini-25-flash-image) — image.
- [`openrouter/google/gemini-3-pro-image`](#googlegemini-3-pro-image) — image.
- [`openrouter/google/gemini-3-pro-image-preview`](#googlegemini-3-pro-image-preview) — image.
- [`openrouter/google/gemini-3.1-flash-image`](#googlegemini-31-flash-image) — image.
- [`openrouter/google/gemini-3.1-flash-image-preview`](#googlegemini-31-flash-image-preview) — image.
- [`openrouter/google/gemini-3.1-flash-lite-image`](#googlegemini-31-flash-lite-image) — image.
- [`openrouter/krea/krea-2-large`](#kreakrea-2-large) — image.
- [`openrouter/krea/krea-2-medium`](#kreakrea-2-medium) — image.
- [`openrouter/krea/krea-2-medium-turbo`](#kreakrea-2-medium-turbo) — image.
- [`openrouter/microsoft/mai-image-2.5`](#microsoftmai-image-25) — image.
- [`openrouter/microsoft/mai-image-2.5-pro`](#microsoftmai-image-25-pro) — image.
- [`openrouter/openai/gpt-5-image`](#openaigpt-5-image) — image.
- [`openrouter/openai/gpt-5-image-mini`](#openaigpt-5-image-mini) — image.
- [`openrouter/openai/gpt-5.4-image-2`](#openaigpt-54-image-2) — image.
- [`openrouter/openai/gpt-image-1`](#openaigpt-image-1) — image.
- [`openrouter/openai/gpt-image-1-mini`](#openaigpt-image-1-mini) — image.
- [`openrouter/openai/gpt-image-2`](#openaigpt-image-2) — image.
- [`openrouter/qwen/qwen-image-3`](#qwenqwen-image-3) — image.
- [`openrouter/qwen/qwen-image-3-pro`](#qwenqwen-image-3-pro) — image.
- [`openrouter/recraft/recraft-v3`](#recraftrecraft-v3) — image.
- [`openrouter/recraft/recraft-v4`](#recraftrecraft-v4) — image.
- [`openrouter/recraft/recraft-v4-pro`](#recraftrecraft-v4-pro) — image.
- [`openrouter/recraft/recraft-v4-pro-vector`](#recraftrecraft-v4-pro-vector) — image.
- [`openrouter/recraft/recraft-v4-styles`](#recraftrecraft-v4-styles) — image.
- [`openrouter/recraft/recraft-v4-styles-pro`](#recraftrecraft-v4-styles-pro) — image.
- [`openrouter/recraft/recraft-v4-styles-pro-vector`](#recraftrecraft-v4-styles-pro-vector) — image.
- [`openrouter/recraft/recraft-v4-styles-vector`](#recraftrecraft-v4-styles-vector) — image.
- [`openrouter/recraft/recraft-v4-vector`](#recraftrecraft-v4-vector) — image.
- [`openrouter/recraft/recraft-v4.1`](#recraftrecraft-v41) — image.
- [`openrouter/recraft/recraft-v4.1-pro`](#recraftrecraft-v41-pro) — image.
- [`openrouter/recraft/recraft-v4.1-pro-vector`](#recraftrecraft-v41-pro-vector) — image.
- [`openrouter/recraft/recraft-v4.1-utility`](#recraftrecraft-v41-utility) — image.
- [`openrouter/recraft/recraft-v4.1-utility-pro`](#recraftrecraft-v41-utility-pro) — image.
- [`openrouter/recraft/recraft-v4.1-vector`](#recraftrecraft-v41-vector) — image.
- [`openrouter/sourceful/riverflow-v2-fast`](#sourcefulriverflow-v2-fast) — image.
- [`openrouter/sourceful/riverflow-v2-pro`](#sourcefulriverflow-v2-pro) — image.
- [`openrouter/sourceful/riverflow-v2.5-fast`](#sourcefulriverflow-v25-fast) — image.
- [`openrouter/sourceful/riverflow-v2.5-pro`](#sourcefulriverflow-v25-pro) — image.
- [`openrouter/x-ai/grok-imagine-image-2.0`](#x-aigrok-imagine-image-20) — image.
- [`openrouter/x-ai/grok-imagine-image-quality`](#x-aigrok-imagine-image-quality) — image.
- [`openrouter/alibaba/happyhorse-1.0`](#alibabahappyhorse-10) — video.
- [`openrouter/alibaba/happyhorse-1.1`](#alibabahappyhorse-11) — video.
- [`openrouter/alibaba/wan-2.6`](#alibabawan-26) — video.
- [`openrouter/alibaba/wan-2.7`](#alibabawan-27) — video.
- [`openrouter/alibaba/wan-3.0`](#alibabawan-30) — video.
- [`openrouter/alibaba/wan-3.0-prime`](#alibabawan-30-prime) — video.
- [`openrouter/bytedance/seedance-1-5-pro`](#bytedanceseedance-1-5-pro) — video.
- [`openrouter/bytedance/seedance-2.0`](#bytedanceseedance-20) — video.
- [`openrouter/bytedance/seedance-2.0-fast`](#bytedanceseedance-20-fast) — video.
- [`openrouter/bytedance/seedance-2.0-mini`](#bytedanceseedance-20-mini) — video.
- [`openrouter/bytedance/seedance-2.5`](#bytedanceseedance-25) — video.
- [`openrouter/google/veo-3.1`](#googleveo-31) — video.
- [`openrouter/google/veo-3.1-fast`](#googleveo-31-fast) — video.
- [`openrouter/google/veo-3.1-lite`](#googleveo-31-lite) — video.
- [`openrouter/kwaivgi/kling-v3.0-pro`](#kwaivgikling-v30-pro) — video.
- [`openrouter/kwaivgi/kling-v3.0-std`](#kwaivgikling-v30-std) — video.
- [`openrouter/kwaivgi/kling-video-o1`](#kwaivgikling-video-o1) — video.
- [`openrouter/minimax/hailuo-2.3`](#minimaxhailuo-23) — video.
- [`openrouter/minimax/hailuo-3`](#minimaxhailuo-3) — video.
- [`openrouter/openai/sora-2-pro`](#openaisora-2-pro) — video.
- [`openrouter/runway/aleph-2`](#runwayaleph-2) — video.
- [`openrouter/runway/gen-4.5`](#runwaygen-45) — video.
- [`openrouter/x-ai/grok-imagine-video`](#x-aigrok-imagine-video) — video.
- [`openrouter/x-ai/grok-imagine-video-1.5`](#x-aigrok-imagine-video-15) — video.
- [`openrouter/microsoft/mai-image-2.6`](#microsoftmai-image-26) — image.
- [`openrouter/microsoft/mai-image-2.6-flash`](#microsoftmai-image-26-flash) — image.
- [`openrouter/minimax/hailuo-3-max`](#minimaxhailuo-3-max) — video.
- [`openrouter/black-forest-labs/flux-video-edit`](#black-forest-labsflux-video-edit) — video.
- [`openrouter/black-forest-labs/flux-video-upscale`](#black-forest-labsflux-video-upscale) — video.
- [`openrouter/heygen/avatar-iv`](#heygenavatar-iv) — video.

## openai/gpt-image-2.5-flare

OpenAI: GPT Image 2.5 Flare. Output: image.

Full key: `openrouter/openai/gpt-image-2.5-flare`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:2`, `2:3`, `4:3`, `3:4`, `16:9`, `9:16`, `21:9`, `auto` |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `xhigh`, `max`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-image-2.5-sunburst

OpenAI: GPT Image 2.5 Sunburst. Output: image.

Full key: `openrouter/openai/gpt-image-2.5-sunburst`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:2`, `2:3`, `4:3`, `3:4`, `16:9`, `9:16`, `21:9`, `auto` |
| `--quality` | Optional | Allowed: `low`, `medium`, `high`, `xhigh`, `max`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## black-forest-labs/flux.2-flex

Black Forest Labs: FLUX.2 Flex. Output: image.

Full key: `openrouter/black-forest-labs/flux.2-flex`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## black-forest-labs/flux.2-klein-4b

Black Forest Labs: FLUX.2 Klein 4B. Output: image.

Full key: `openrouter/black-forest-labs/flux.2-klein-4b`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## black-forest-labs/flux.2-max

Black Forest Labs: FLUX.2 Max. Output: image.

Full key: `openrouter/black-forest-labs/flux.2-max`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## black-forest-labs/flux.2-pro

Black Forest Labs: FLUX.2 Pro. Output: image.

Full key: `openrouter/black-forest-labs/flux.2-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg` |
| `--input-media` | Optional | Maximum inputs: `8` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## black-forest-labs/flux-3-video

Black Forest Labs: FLUX.3 Video. Output: video.

Full key: `openrouter/black-forest-labs/flux-3-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `21:9`, `16:9`, `4:3`, `1:1`, `3:4`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15`, `16`, `17`, `18`, `19`, `20` |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance-seed/seedream-4.5

ByteDance Seed: Seedream 4.5. Output: image.

Full key: `openrouter/bytedance-seed/seedream-4.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:2`, `2:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `9:19.5`, `19.5:9`, `9:20`, `20:9`, `9:21`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance-seed/seedream-5-0-pro

ByteDance Seed: Seedream 5.0 Pro. Output: image.

Full key: `openrouter/bytedance-seed/seedream-5-0-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:2`, `2:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `9:19.5`, `19.5:9`, `9:20`, `20:9`, `9:21`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance-seed/seedream-5-0-lite

ByteDance Seed: Seedream 5.0 Lite. Output: image.

Full key: `openrouter/bytedance-seed/seedream-5-0-lite`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:2`, `2:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `9:19.5`, `19.5:9`, `9:20`, `20:9`, `9:21`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `2K`, `4K` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `4` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## google/gemini-2.5-flash-image

Google: Nano Banana (Gemini 2.5 Flash Image). Output: image.

Full key: `openrouter/google/gemini-2.5-flash-image`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `3` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## google/gemini-3-pro-image

Google: Nano Banana Pro (Gemini 3 Pro Image). Output: image.

Full key: `openrouter/google/gemini-3-pro-image`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## google/gemini-3-pro-image-preview

Google: Nano Banana Pro (Gemini 3 Pro Image Preview). Output: image.

Full key: `openrouter/google/gemini-3-pro-image-preview`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## google/gemini-3.1-flash-image

Google: Nano Banana 2 (Gemini 3.1 Flash Image). Output: image.

Full key: `openrouter/google/gemini-3.1-flash-image`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:4`, `1:8`, `2:3`, `3:2`, `3:4`, `4:1`, `4:3`, `4:5`, `5:4`, `8:1`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `512`, `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## google/gemini-3.1-flash-image-preview

Google: Nano Banana 2 (Gemini 3.1 Flash Image Preview). Output: image.

Full key: `openrouter/google/gemini-3.1-flash-image-preview`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:4`, `1:8`, `2:3`, `3:2`, `3:4`, `4:1`, `4:3`, `4:5`, `5:4`, `8:1`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `512`, `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## google/gemini-3.1-flash-lite-image

Google: Nano Banana 2 Lite (Gemini 3.1 Flash Lite Image). Output: image.

Full key: `openrouter/google/gemini-3.1-flash-lite-image`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:4`, `1:8`, `2:3`, `3:2`, `3:4`, `4:1`, `4:3`, `4:5`, `5:4`, `8:1`, `9:16`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `1K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `14` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## krea/krea-2-large

Krea: Krea 2 Large. Output: image.

Full key: `openrouter/krea/krea-2-large`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:2`, `16:9`, `4:5`, `2:3`, `9:16` |
| `--resolution` | Optional | Allowed: `1K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## krea/krea-2-medium

Krea: Krea 2 Medium. Output: image.

Full key: `openrouter/krea/krea-2-medium`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:2`, `16:9`, `4:5`, `2:3`, `9:16` |
| `--resolution` | Optional | Allowed: `1K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## krea/krea-2-medium-turbo

Krea: Krea 2 Medium Turbo. Output: image.

Full key: `openrouter/krea/krea-2-medium-turbo`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:2`, `16:9`, `4:5`, `2:3`, `9:16` |
| `--resolution` | Optional | Allowed: `1K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## microsoft/mai-image-2.5

Microsoft: MAI-Image-2.5. Output: image.

Full key: `openrouter/microsoft/mai-image-2.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `3:2`, `2:3`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## microsoft/mai-image-2.5-pro

Microsoft: MAI-Image-2.5 Pro. Output: image.

Full key: `openrouter/microsoft/mai-image-2.5-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `3:2`, `2:3`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-5-image

OpenAI: GPT-5 Image. Output: image.

Full key: `openrouter/openai/gpt-5-image`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `auto`, `low`, `medium`, `high` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-5-image-mini

OpenAI: GPT-5 Image Mini. Output: image.

Full key: `openrouter/openai/gpt-5-image-mini`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `auto`, `low`, `medium`, `high` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-5.4-image-2

OpenAI: GPT-5.4 Image 2. Output: image.

Full key: `openrouter/openai/gpt-5.4-image-2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | No declared value constraint; see the general flag and input rules. |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `auto`, `low`, `medium`, `high` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-image-1

OpenAI: GPT Image 1. Output: image.

Full key: `openrouter/openai/gpt-image-1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:2`, `2:3`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `auto`, `low`, `medium`, `high` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-image-1-mini

OpenAI: GPT Image 1 Mini. Output: image.

Full key: `openrouter/openai/gpt-image-1-mini`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:2`, `2:3`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `auto`, `low`, `medium`, `high` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## openai/gpt-image-2

OpenAI: GPT Image 2. Output: image.

Full key: `openrouter/openai/gpt-image-2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:2`, `2:3`, `4:3`, `3:4`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | Allowed: `auto`, `low`, `medium`, `high` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `10` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `16` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## qwen/qwen-image-3

Qwen: Qwen Image 3. Output: image.

Full key: `openrouter/qwen/qwen-image-3`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:2`, `1:4`, `2:1`, `2:3`, `3:2`, `3:4`, `4:1`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9` |
| `--resolution` | Optional | Allowed: `1K`, `2K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## qwen/qwen-image-3-pro

Qwen: Qwen Image 3 Pro. Output: image.

Full key: `openrouter/qwen/qwen-image-3-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `1:2`, `1:4`, `2:1`, `2:3`, `3:2`, `3:4`, `4:1`, `4:3`, `4:5`, `5:4`, `9:16`, `16:9` |
| `--resolution` | Optional | Allowed: `1K`, `2K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## recraft/recraft-v3

Recraft: Recraft V3. Output: image.

Full key: `openrouter/recraft/recraft-v3`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4

Recraft: Recraft V4. Output: image.

Full key: `openrouter/recraft/recraft-v4`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4-pro

Recraft: Recraft V4 Pro. Output: image.

Full key: `openrouter/recraft/recraft-v4-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4-pro-vector

Recraft: Recraft V4 Pro Vector. Output: image.

Full key: `openrouter/recraft/recraft-v4-pro-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4-styles

Recraft: Recraft V4 Styles. Output: image.

Full key: `openrouter/recraft/recraft-v4-styles`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `16:9`, `9:16`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--input-media` | Required | Maximum inputs: `10` |

## recraft/recraft-v4-styles-pro

Recraft: Recraft V4 Styles Pro. Output: image.

Full key: `openrouter/recraft/recraft-v4-styles-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `16:9`, `9:16`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--input-media` | Required | Maximum inputs: `10` |

## recraft/recraft-v4-styles-pro-vector

Recraft: Recraft V4 Styles Pro Vector. Output: image.

Full key: `openrouter/recraft/recraft-v4-styles-pro-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `16:9`, `9:16`, `auto` |
| `--output-format` | Optional | Allowed: `svg` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--input-media` | Required | Maximum inputs: `10` |

## recraft/recraft-v4-styles-vector

Recraft: Recraft V4 Styles Vector. Output: image.

Full key: `openrouter/recraft/recraft-v4-styles-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `16:9`, `9:16`, `auto` |
| `--output-format` | Optional | Allowed: `svg` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--input-media` | Required | Maximum inputs: `10` |

## recraft/recraft-v4-vector

Recraft: Recraft V4 Vector. Output: image.

Full key: `openrouter/recraft/recraft-v4-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4.1

Recraft: Recraft V4.1. Output: image.

Full key: `openrouter/recraft/recraft-v4.1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4.1-pro

Recraft: Recraft V4.1 Pro. Output: image.

Full key: `openrouter/recraft/recraft-v4.1-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4.1-pro-vector

Recraft: Recraft V4.1 Pro Vector. Output: image.

Full key: `openrouter/recraft/recraft-v4.1-pro-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4.1-utility

Recraft: Recraft V4.1 Utility. Output: image.

Full key: `openrouter/recraft/recraft-v4.1-utility`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4.1-utility-pro

Recraft: Recraft V4.1 Utility Pro. Output: image.

Full key: `openrouter/recraft/recraft-v4.1-utility-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## recraft/recraft-v4.1-vector

Recraft: Recraft V4.1 Vector. Output: image.

Full key: `openrouter/recraft/recraft-v4.1-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `auto` |
| `--resolution` | Optional | No declared value constraint; see the general flag and input rules. |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `1` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## sourceful/riverflow-v2-fast

Sourceful: Riverflow V2 Fast. Output: image.

Full key: `openrouter/sourceful/riverflow-v2-fast`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## sourceful/riverflow-v2-pro

Sourceful: Riverflow V2 Pro. Output: image.

Full key: `openrouter/sourceful/riverflow-v2-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `10` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## sourceful/riverflow-v2.5-fast

Sourceful: Riverflow V2.5 Fast. Output: image.

Full key: `openrouter/sourceful/riverflow-v2.5-fast`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `jpeg` |
| `--input-media` | Optional | Maximum inputs: `4` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## sourceful/riverflow-v2.5-pro

Sourceful: Riverflow V2.5 Pro. Output: image.

Full key: `openrouter/sourceful/riverflow-v2.5-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `3:2`, `2:3`, `16:9`, `9:16`, `21:9`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `10` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## x-ai/grok-imagine-image-2.0

xAI: Grok Imagine Image 2.0. Output: image.

Full key: `openrouter/x-ai/grok-imagine-image-2.0`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:4`, `4:3`, `9:16`, `16:9`, `2:3`, `3:2`, `9:19.5`, `19.5:9`, `9:20`, `20:9`, `1:2`, `2:1`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K` |
| `--quality` | Optional | Allowed: `low`, `medium` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `3` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## x-ai/grok-imagine-image-quality

SpaceXAI: Grok Imagine Image Quality. Output: image.

Full key: `openrouter/x-ai/grok-imagine-image-quality`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:4`, `4:3`, `9:16`, `16:9`, `2:3`, `3:2`, `9:19.5`, `19.5:9`, `9:20`, `20:9`, `1:2`, `2:1`, `auto` |
| `--resolution` | Optional | Allowed: `1K`, `2K` |
| `--quality` | Optional | No declared value constraint; see the general flag and input rules. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--output-format` | Optional | Allowed: `png`, `jpeg`, `webp` |
| `--input-media` | Optional | Maximum inputs: `3` |
| `--background` | Optional | Allowed: `auto`, `transparent`, `opaque` |
| `--output-compression` | Optional | Minimum: `0`; Maximum: `100` |

## alibaba/happyhorse-1.0

Alibaba: HappyHorse 1.0. Output: video.

Full key: `openrouter/alibaba/happyhorse-1.0`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `21:9`, `9:21` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## alibaba/happyhorse-1.1

Alibaba: HappyHorse 1.1. Output: video.

Full key: `openrouter/alibaba/happyhorse-1.1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `21:9`, `9:21` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## alibaba/wan-2.6

Alibaba: Wan 2.6. Output: video.

Full key: `openrouter/alibaba/wan-2.6`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `5`, `10` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## alibaba/wan-2.7

Alibaba: Wan 2.7. Output: video.

Full key: `openrouter/alibaba/wan-2.7`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## alibaba/wan-3.0

Alibaba: Wan 3.0. Output: video.

Full key: `openrouter/alibaba/wan-3.0`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `4:3`, `1:1`, `3:4`, `9:16` |
| `--resolution` | Optional | Allowed: `480p`, `720p`, `1080p` |
| `--duration` | Optional | Minimum: `2`; Maximum: `30` |
| `--input-media` | Optional | Maximum inputs: `1`; Use `first:` for an opening image only. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## alibaba/wan-3.0-prime

Alibaba: Wan 3.0 Prime. Output: video.

Full key: `openrouter/alibaba/wan-3.0-prime`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `4:3`, `1:1`, `3:4`, `9:16` |
| `--resolution` | Optional | Allowed: `480p`, `720p`, `1080p` |
| `--duration` | Optional | Minimum: `2`; Maximum: `30` |
| `--input-media` | Optional | Maximum inputs: `1`; Use `first:` for an opening image only. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance/seedance-1-5-pro

ByteDance: Seedance 1.5 Pro. Output: video.

Full key: `openrouter/bytedance/seedance-1-5-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:4`, `9:16`, `9:21`, `4:3`, `16:9`, `21:9` |
| `--resolution` | Optional | Allowed: `480p`, `720p`, `1080p` |
| `--duration` | Optional | Allowed: `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance/seedance-2.0

ByteDance: Seedance 2.0. Output: video.

Full key: `openrouter/bytedance/seedance-2.0`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:4`, `9:16`, `4:3`, `16:9`, `21:9`, `9:21` |
| `--resolution` | Optional | Allowed: `480p`, `720p`, `1080p`, `4K` |
| `--duration` | Optional | Allowed: `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance/seedance-2.0-fast

ByteDance: Seedance 2.0 Fast. Output: video.

Full key: `openrouter/bytedance/seedance-2.0-fast`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:4`, `9:16`, `4:3`, `16:9`, `21:9`, `9:21` |
| `--resolution` | Optional | Allowed: `480p`, `720p` |
| `--duration` | Optional | Allowed: `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance/seedance-2.0-mini

ByteDance: Seedance 2.0 Mini. Output: video.

Full key: `openrouter/bytedance/seedance-2.0-mini`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `3:4`, `9:16`, `4:3`, `16:9`, `21:9`, `9:21` |
| `--resolution` | Optional | Allowed: `480p`, `720p` |
| `--duration` | Optional | Allowed: `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## bytedance/seedance-2.5

ByteDance: Seedance 2.5. Output: video.

Full key: `openrouter/bytedance/seedance-2.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `4:3`, `1:1`, `3:4`, `9:16`, `21:9` |
| `--resolution` | Optional | Allowed: `480p`, `720p` |
| `--duration` | Optional | Allowed: `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15`, `16`, `17`, `18`, `19`, `20`, `21`, `22`, `23`, `24`, `25`, `26`, `27`, `28`, `29`, `30` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## google/veo-3.1

Google: Veo 3.1. Output: video.

Full key: `openrouter/google/veo-3.1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p`, `4K` |
| `--duration` | Optional | Allowed: `4`, `6`, `8` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## google/veo-3.1-fast

Google: Veo 3.1 Fast. Output: video.

Full key: `openrouter/google/veo-3.1-fast`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p`, `4K` |
| `--duration` | Optional | Allowed: `4`, `6`, `8` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## google/veo-3.1-lite

Google: Veo 3.1 Lite. Output: video.

Full key: `openrouter/google/veo-3.1-lite`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `4`, `6`, `8` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## kwaivgi/kling-v3.0-pro

Kling: Video v3.0 Pro. Output: video.

Full key: `openrouter/kwaivgi/kling-v3.0-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--resolution` | Optional | Allowed: `720p` |
| `--duration` | Optional | Allowed: `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## kwaivgi/kling-v3.0-std

Kling: Video v3.0 Standard. Output: video.

Full key: `openrouter/kwaivgi/kling-v3.0-std`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--resolution` | Optional | Allowed: `720p` |
| `--duration` | Optional | Allowed: `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## kwaivgi/kling-video-o1

Kling: Video O1. Output: video.

Full key: `openrouter/kwaivgi/kling-video-o1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--resolution` | Optional | Allowed: `720p` |
| `--duration` | Optional | Allowed: `5`, `10` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## minimax/hailuo-2.3

MiniMax: Hailuo 2.3. Output: video.

Full key: `openrouter/minimax/hailuo-2.3`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9` |
| `--resolution` | Optional | Allowed: `1080p` |
| `--duration` | Optional | Allowed: `6`, `10` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |

## minimax/hailuo-3

MiniMax: H3. Output: video.

Full key: `openrouter/minimax/hailuo-3`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `21:9`, `16:9`, `4:3`, `1:1`, `3:4`, `9:16` |
| `--resolution` | Optional | Allowed: `2K` |
| `--duration` | Optional | Allowed: `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## openai/sora-2-pro

OpenAI: Sora 2 Pro. Output: video.

Full key: `openrouter/openai/sora-2-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--duration` | Optional | Allowed: `4`, `8`, `12`, `16`, `20` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--generate-audio` | Optional | No declared value constraint; see the general flag and input rules. |

## runway/aleph-2

Runway: Aleph 2.0. Output: video.

Full key: `openrouter/runway/aleph-2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `4:3`, `3:2`, `1:1`, `2:3`, `3:4`, `9:16`, `21:9` |
| `--input-media` | Required | Maximum inputs: `1` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## runway/gen-4.5

Runway: Gen-4.5. Output: video.

Full key: `openrouter/runway/gen-4.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16` |
| `--resolution` | Optional | Allowed: `720p` |
| `--duration` | Optional | Allowed: `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |

## x-ai/grok-imagine-video

SpaceXAI: Grok Imagine Video. Output: video.

Full key: `openrouter/x-ai/grok-imagine-video`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `3:2`, `2:3` |
| `--resolution` | Optional | Allowed: `480p`, `720p` |
| `--duration` | Optional | Allowed: `1`, `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |

## x-ai/grok-imagine-video-1.5

SpaceXAI: Grok Imagine Video 1.5. Output: video.

Full key: `openrouter/x-ai/grok-imagine-video-1.5`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1`, `4:3`, `3:4`, `3:2`, `2:3` |
| `--resolution` | Optional | Allowed: `480p`, `720p`, `1080p` |
| `--duration` | Optional | Allowed: `1`, `2`, `3`, `4`, `5`, `6`, `7`, `8`, `9`, `10`, `11`, `12`, `13`, `14`, `15` |
| `--input-media` | Optional | Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |

## microsoft/mai-image-2.6

Microsoft: MAI Image 2.6. Output: image.

Full key: `openrouter/microsoft/mai-image-2.6`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `3:2`, `2:3`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--input-media` | Optional | Maximum inputs: `5` |

## microsoft/mai-image-2.6-flash

Microsoft: MAI Image 2.6 Flash. Output: image.

Full key: `openrouter/microsoft/mai-image-2.6-flash`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `4:3`, `3:4`, `16:9`, `9:16`, `3:2`, `2:3`, `auto` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `1` |
| `--input-media` | Optional | Maximum inputs: `5` |

## minimax/hailuo-3-max

MiniMax: Hailuo 3 Max. Output: video.

Full key: `openrouter/minimax/hailuo-3-max`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `21:9`, `16:9`, `4:3`, `1:1`, `3:4`, `9:16` |
| `--resolution` | Optional | Allowed: `768p`, `480p` |
| `--duration` | Optional | Minimum: `5`; Maximum: `15` |
| `--input-media` | Optional | Maximum inputs: `2`; Use `first:` for an opening image and `last:` for a closing image. See [frame rules and model limitations](../input-media.md#opening-and-closing-frames). |
| `--watermark` | Optional | No declared value constraint; see the general flag and input rules. |

## black-forest-labs/flux-video-edit

Black Forest Labs: FLUX Video Edit. Output: video.

Full key: `openrouter/black-forest-labs/flux-video-edit`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply one public HTTPS video URL. Local video files are rejected by this route. See [OpenRouter’s video generation guide](https://openrouter.ai/docs/guides/overview/multimodal/video-generation). |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |

## black-forest-labs/flux-video-upscale

Black Forest Labs: FLUX Video Upscale. Output: video.

Full key: `openrouter/black-forest-labs/flux-video-upscale`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply one public HTTPS video URL. Local video files are rejected by this route. See [OpenRouter’s video generation guide](https://openrouter.ai/docs/guides/overview/multimodal/video-generation). |
| `--upscale-factor` | Optional | Minimum: `1.5`; Maximum: `3` |
| `--creativity` | Optional | Allowed: `0`, `1` |
| `--safety-tolerance` | Optional | Minimum: `0`; Maximum: `5` |

## heygen/avatar-iv

HeyGen: Avatar IV. Output: video.

Full key: `openrouter/heygen/avatar-iv`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--input-media` | Required | Maximum inputs: `1`; Supply one portrait image. The prompt is spoken using --voice-id. Audio input is not supported. |
| `--voice-id` | Required | Examples: `16a09e4706f74997ba4ed05ea11470f6`, `6be73833ef9a4eb0aeee399b8fe9d62b`. The example IDs have been verified with this endpoint. Find other voice IDs in [HeyGen’s voice list reference](https://developers.heygen.com/reference/list-voices). |
| `--aspect-ratio` | Optional | Allowed: `16:9`, `9:16`, `1:1` |
| `--resolution` | Optional | Allowed: `720p`, `1080p` |
| `--motion-prompt` | Optional | No declared value constraint; see the general flag and input rules. |
| `--expressiveness` | Optional | Examples: `low`, `medium`, `high`. |
| `--image-fit` | Optional | Examples: `contain`, `cover`. |
| `--remove-background` | Optional | No declared value constraint; see the general flag and input rules. |
| `--voice-speed` | Optional | Minimum: `0.5`; Maximum: `1.5` |
| `--voice-pitch` | Optional | Pitch adjustment from -50 to +50 semitones. |
| `--voice-volume` | Optional | Minimum: `0`; Maximum: `1` |
| `--voice-locale` | Optional | Examples: `en-US`, `en-GB`. |

Revised 2026-10-01
