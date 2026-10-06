# OpenRouter Models

Every model that `bild` offers from OpenRouter (provider ID `openrouter`, API key variable `OPENROUTER_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info openrouter/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

56 image models, 30 video models.

## Contents

- [OpenAI: GPT Image 2.5 Flare](#openai-gpt-image-25-flare)
- [OpenAI: GPT Image 2.5 Sunburst](#openai-gpt-image-25-sunburst)
- [Black Forest Labs: FLUX.2 Flex](#black-forest-labs-flux2-flex)
- [Black Forest Labs: FLUX.2 Klein 4B](#black-forest-labs-flux2-klein-4b)
- [Black Forest Labs: FLUX.2 Max](#black-forest-labs-flux2-max)
- [Black Forest Labs: FLUX.2 Pro](#black-forest-labs-flux2-pro)
- [Black Forest Labs: FLUX.3 Video](#black-forest-labs-flux3-video)
- [ByteDance Seed: Seedream 4.5](#bytedance-seed-seedream-45)
- [ByteDance Seed: Seedream 5.0 Pro](#bytedance-seed-seedream-50-pro)
- [ByteDance Seed: Seedream 5.0 Lite](#bytedance-seed-seedream-50-lite)
- [Google: Nano Banana (Gemini 2.5 Flash Image)](#google-nano-banana-gemini-25-flash-image)
- [Google: Nano Banana Pro (Gemini 3 Pro Image)](#google-nano-banana-pro-gemini-3-pro-image)
- [Google: Nano Banana Pro (Gemini 3 Pro Image Preview)](#google-nano-banana-pro-gemini-3-pro-image-preview)
- [Google: Nano Banana 2 (Gemini 3.1 Flash Image)](#google-nano-banana-2-gemini-31-flash-image)
- [Google: Nano Banana 2 (Gemini 3.1 Flash Image Preview)](#google-nano-banana-2-gemini-31-flash-image-preview)
- [Google: Nano Banana 2 Lite (Gemini 3.1 Flash Lite Image)](#google-nano-banana-2-lite-gemini-31-flash-lite-image)
- [Krea: Krea 2 Large](#krea-krea-2-large)
- [Krea: Krea 2 Medium](#krea-krea-2-medium)
- [Krea: Krea 2 Medium Turbo](#krea-krea-2-medium-turbo)
- [Microsoft: MAI-Image-2.5](#microsoft-mai-image-25)
- [Microsoft: MAI-Image-2.5 Pro](#microsoft-mai-image-25-pro)
- [OpenAI: GPT-5 Image](#openai-gpt-5-image)
- [OpenAI: GPT-5 Image Mini](#openai-gpt-5-image-mini)
- [OpenAI: GPT-5.4 Image 2](#openai-gpt-54-image-2)
- [OpenAI: GPT Image 1](#openai-gpt-image-1)
- [OpenAI: GPT Image 1 Mini](#openai-gpt-image-1-mini)
- [OpenAI: GPT Image 2](#openai-gpt-image-2)
- [Qwen: Qwen Image 3](#qwen-qwen-image-3)
- [Qwen: Qwen Image 3 Pro](#qwen-qwen-image-3-pro)
- [Recraft: Recraft V3](#recraft-recraft-v3)
- [Recraft: Recraft V4](#recraft-recraft-v4)
- [Recraft: Recraft V4 Pro](#recraft-recraft-v4-pro)
- [Recraft: Recraft V4 Pro Vector](#recraft-recraft-v4-pro-vector)
- [Recraft: Recraft V4 Styles](#recraft-recraft-v4-styles)
- [Recraft: Recraft V4 Styles Pro](#recraft-recraft-v4-styles-pro)
- [Recraft: Recraft V4 Styles Pro Vector](#recraft-recraft-v4-styles-pro-vector)
- [Recraft: Recraft V4 Styles Vector](#recraft-recraft-v4-styles-vector)
- [Recraft: Recraft V4 Vector](#recraft-recraft-v4-vector)
- [Recraft: Recraft V4.1](#recraft-recraft-v41)
- [Recraft: Recraft V4.1 Pro](#recraft-recraft-v41-pro)
- [Recraft: Recraft V4.1 Pro Vector](#recraft-recraft-v41-pro-vector)
- [Recraft: Recraft V4.1 Utility](#recraft-recraft-v41-utility)
- [Recraft: Recraft V4.1 Utility Pro](#recraft-recraft-v41-utility-pro)
- [Recraft: Recraft V4.1 Vector](#recraft-recraft-v41-vector)
- [Sourceful: Riverflow V2 Fast](#sourceful-riverflow-v2-fast)
- [Sourceful: Riverflow V2 Pro](#sourceful-riverflow-v2-pro)
- [Sourceful: Riverflow V2.5 Fast](#sourceful-riverflow-v25-fast)
- [Sourceful: Riverflow V2.5 Pro](#sourceful-riverflow-v25-pro)
- [xAI: Grok Imagine Image 2.0](#xai-grok-imagine-image-20)
- [SpaceXAI: Grok Imagine Image Quality](#spacexai-grok-imagine-image-quality)
- [Alibaba: HappyHorse 1.0](#alibaba-happyhorse-10)
- [Alibaba: HappyHorse 1.1](#alibaba-happyhorse-11)
- [Alibaba: Wan 2.6](#alibaba-wan-26)
- [Alibaba: Wan 2.7](#alibaba-wan-27)
- [Alibaba: Wan 3.0](#alibaba-wan-30)
- [Alibaba: Wan 3.0 Prime](#alibaba-wan-30-prime)
- [ByteDance: Seedance 1.5 Pro](#bytedance-seedance-15-pro)
- [ByteDance: Seedance 2.0](#bytedance-seedance-20)
- [ByteDance: Seedance 2.0 Fast](#bytedance-seedance-20-fast)
- [ByteDance: Seedance 2.0 Mini](#bytedance-seedance-20-mini)
- [ByteDance: Seedance 2.5](#bytedance-seedance-25)
- [Google: Veo 3.1](#google-veo-31)
- [Google: Veo 3.1 Fast](#google-veo-31-fast)
- [Google: Veo 3.1 Lite](#google-veo-31-lite)
- [Kling: Video v3.0 Pro](#kling-video-v30-pro)
- [Kling: Video v3.0 Standard](#kling-video-v30-standard)
- [Kling: Video O1](#kling-video-o1)
- [MiniMax: Hailuo 2.3](#minimax-hailuo-23)
- [MiniMax: H3](#minimax-h3)
- [OpenAI: Sora 2 Pro](#openai-sora-2-pro)
- [Runway: Aleph 2.0](#runway-aleph-20)
- [Runway: Gen-4.5](#runway-gen-45)
- [SpaceXAI: Grok Imagine Video](#spacexai-grok-imagine-video)
- [SpaceXAI: Grok Imagine Video 1.5](#spacexai-grok-imagine-video-15)
- [Microsoft: MAI Image 2.6](#microsoft-mai-image-26)
- [Microsoft: MAI Image 2.6 Flash](#microsoft-mai-image-26-flash)
- [MiniMax: Hailuo 3 Max](#minimax-hailuo-3-max)
- [Black Forest Labs: FLUX Video Edit](#black-forest-labs-flux-video-edit)
- [Black Forest Labs: FLUX Video Upscale](#black-forest-labs-flux-video-upscale)
- [HeyGen: Avatar IV](#heygen-avatar-iv)
- [Black Forest Labs: FLUX 3 Image](#black-forest-labs-flux-3-image)
- [ByteDance Seed: Seedream 5.0 Flash](#bytedance-seed-seedream-50-flash)
- [InclusionAI: Ming Image 0.1 Design](#inclusionai-ming-image-01-design)
- [InclusionAI: Ming Image 0.1 Design Layer](#inclusionai-ming-image-01-design-layer)
- [Recraft: Recraft V4.1 Flash](#recraft-recraft-v41-flash)
- [HeyGen: HeyGen Video 1](#heygen-heygen-video-1)

## OpenAI: GPT Image 2.5 Flare

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT Image 2.5 Flare | image | none | `openrouter/openai/gpt-image-2.5-flare`

Fast image generation and editing for everyday creative work.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:2, 2:3, 4:3, 3:4, 16:9, 9:16, 21:9, auto
`--quality` | Allowed values: low, medium, high, xhigh, max, auto
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT Image 2.5 Sunburst

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT Image 2.5 Sunburst | image | none | `openrouter/openai/gpt-image-2.5-sunburst`

Generate images and make precise edits from text and image references.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:2, 2:3, 4:3, 3:4, 16:9, 9:16, 21:9, auto
`--quality` | Allowed values: low, medium, high, xhigh, max, auto
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, opaque
`--output-compression` | Allowed range: 0 to 100

## Black Forest Labs: FLUX.2 Flex

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX.2 Flex | image | none | `openrouter/black-forest-labs/flux.2-flex`

FLUX.2 [flex] excels at rendering complex text, typography, and fine details, and supports multi-reference editing in the same unified architecture. Pricing is [per the BFL docs](https://bfl.ai/pricing?category=flux.2).

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 8
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Black Forest Labs: FLUX.2 Klein 4B

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX.2 Klein 4B | image | none | `openrouter/black-forest-labs/flux.2-klein-4b`

FLUX.2 [klein] 4B is the fastest and most cost-effective model in the FLUX.2 family, optimized for high-throughput use cases while maintaining excellent image quality.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 4
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Black Forest Labs: FLUX.2 Max

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX.2 Max | image | none | `openrouter/black-forest-labs/flux.2-max`

FLUX.2 [max] is the new top-tier image model from Black Forest Labs, pushing image quality, prompt understanding, and editing consistency to the highest level yet.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 8
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Black Forest Labs: FLUX.2 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX.2 Pro | image | none | `openrouter/black-forest-labs/flux.2-pro`

A high-end image generation and editing model focused on frontier-level visual quality and reliability. It delivers strong prompt adherence, stable lighting, sharp textures, and consistent character/style reproduction across multi-reference inputs.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 8
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Black Forest Labs: FLUX.3 Video

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX.3 Video | video | none | `openrouter/black-forest-labs/flux-3-video`

FLUX.3 Video is a video generation model from Black Forest Labs. It supports text-to-video, image-guided generation with opening and closing keyframes, and video continuation workflows.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 21:9, 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20
`--generate-audio` |

## ByteDance Seed: Seedream 4.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance Seed: Seedream 4.5 | image | none | `openrouter/bytedance-seed/seedream-4.5`

Seedream 4.5 is the latest in-house image generation model developed by ByteDance. Compared with Seedream 4.0, it delivers comprehensive improvements, especially in editing consistency, including better preservation of subject details.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:2, 2:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 9:19.5, 19.5:9, 9:20, 20:9, 9:21, 21:9, auto
`--resolution` | Allowed values: 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 14
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## ByteDance Seed: Seedream 5.0 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance Seed: Seedream 5.0 Pro | image | none | `openrouter/bytedance-seed/seedream-5-0-pro`

Seedream 5.0 Pro is an image generation and editing model from ByteDance Seed. It is suited for commercial visual-production workflows that require precise editing control, lifelike scenes, and natural rendering.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:2, 2:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 9:19.5, 19.5:9, 9:20, 20:9, 9:21, 21:9, auto
`--resolution` | Allowed values: 1K, 2K
`--quality` |
`--output-format` |
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## ByteDance Seed: Seedream 5.0 Lite

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance Seed: Seedream 5.0 Lite | image | none | `openrouter/bytedance-seed/seedream-5-0-lite`

Seedream 5.0 Lite is an image generation and editing model from ByteDance Seed.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:2, 2:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 9:19.5, 19.5:9, 9:20, 20:9, 9:21, 21:9, auto
`--resolution` | Allowed values: 2K, 4K
`--num-images` | Allowed range: 1 to 4
`--input-media` | Repeat maximum: 14
`--seed` |

## Google: Nano Banana (Gemini 2.5 Flash Image)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Nano Banana (Gemini 2.5 Flash Image) | image | none | `openrouter/google/gemini-2.5-flash-image`

Gemini 2.5 Flash Image, a.k.a. "Nano Banana," is now generally available. It is a state of the art image generation model with contextual understanding.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 3
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Google: Nano Banana Pro (Gemini 3 Pro Image)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Nano Banana Pro (Gemini 3 Pro Image) | image | none | `openrouter/google/gemini-3-pro-image`

Nano Banana Pro is Google’s most advanced image-generation and editing model, built on Gemini 3 Pro. It extends the original Nano Banana with significantly improved multimodal reasoning and real-world grounding.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Google: Nano Banana Pro (Gemini 3 Pro Image Preview)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Nano Banana Pro (Gemini 3 Pro Image Preview) | image | none | `openrouter/google/gemini-3-pro-image-preview`

Nano Banana Pro is Google’s most advanced image-generation and editing model, built on Gemini 3 Pro. It extends the original Nano Banana with significantly improved multimodal reasoning and real-world grounding.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Google: Nano Banana 2 (Gemini 3.1 Flash Image)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Nano Banana 2 (Gemini 3.1 Flash Image) | image | none | `openrouter/google/gemini-3.1-flash-image`

Gemini 3.1 Flash Image, a.k.a. "Nano Banana 2," is Google’s latest state of the art image generation and editing model, delivering Pro-level visual quality at Flash speed.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:4, 1:8, 2:3, 3:2, 3:4, 4:1, 4:3, 4:5, 5:4, 8:1, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 512, 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Google: Nano Banana 2 (Gemini 3.1 Flash Image Preview)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Nano Banana 2 (Gemini 3.1 Flash Image Preview) | image | none | `openrouter/google/gemini-3.1-flash-image-preview`

Gemini 3.1 Flash Image Preview, a.k.a. "Nano Banana 2," is Google’s latest state of the art image generation and editing model, delivering Pro-level visual quality at Flash speed.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:4, 1:8, 2:3, 3:2, 3:4, 4:1, 4:3, 4:5, 5:4, 8:1, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 512, 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Google: Nano Banana 2 Lite (Gemini 3.1 Flash Lite Image)

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Nano Banana 2 Lite (Gemini 3.1 Flash Lite Image) | image | none | `openrouter/google/gemini-3.1-flash-lite-image`

Nano Banana 2 Lite (Gemini 3.1 Flash Lite Image) is Google's fastest, most cost-efficient Gemini image model, built for high-velocity developer pipelines and rapid-fire visual exploration.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:4, 1:8, 2:3, 3:2, 3:4, 4:1, 4:3, 4:5, 5:4, 8:1, 9:16, 16:9, 21:9
`--resolution` | Allowed values: 1K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Krea: Krea 2 Large

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Krea: Krea 2 Large | image | none | `openrouter/krea/krea-2-large`

Krea 2 Large is Krea's high-capability image generation model, more than twice the size of Krea 2 Medium. Its lighter post-training gives images a rawer, more textured quality.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:2, 16:9, 4:5, 2:3, 9:16
`--resolution` | Allowed values: 1K
`--quality` |
`--output-format` |
`--input-media` | Repeat maximum: 1
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Krea: Krea 2 Medium

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Krea: Krea 2 Medium | image | none | `openrouter/krea/krea-2-medium`

Krea 2 Medium is Krea's balanced, cost-efficient image generation model and a practical starting point for a broad range of use cases. Its extensive post-training supports stable, consistent generations.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:2, 16:9, 4:5, 2:3, 9:16
`--resolution` | Allowed values: 1K
`--quality` |
`--output-format` |
`--input-media` | Repeat maximum: 1
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Krea: Krea 2 Medium Turbo

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Krea: Krea 2 Medium Turbo | image | none | `openrouter/krea/krea-2-medium-turbo`

Krea 2 Medium Turbo is a distilled, speed-focused variant of Krea 2 Medium from Krea. It is designed for rapid iteration and graphic design exploration where fast generation is the priority.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:2, 16:9, 4:5, 2:3, 9:16
`--resolution` | Allowed values: 1K
`--quality` |
`--output-format` |
`--input-media` | Repeat maximum: 1
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Microsoft: MAI-Image-2.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Microsoft: MAI-Image-2.5 | image | none | `openrouter/microsoft/mai-image-2.5`

Microsoft's MAI-Image-2.5 is a high-quality image generation model available via Azure AI Foundry. It produces photorealistic and artistic images from text prompts with support for various aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, 3:2, 2:3, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Microsoft: MAI-Image-2.5 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Microsoft: MAI-Image-2.5 Pro | image | none | `openrouter/microsoft/mai-image-2.5-pro`

Microsoft's MAI-Image-2.5 is a high-quality image generation model available via Azure AI Foundry. It produces photorealistic and artistic images from text prompts with support for various aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, 3:2, 2:3, auto
`--resolution` |
`--quality` |
`--output-format` |
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT-5 Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT-5 Image | image | none | `openrouter/openai/gpt-5-image`

[GPT-5](https://openrouter.ai/openai/gpt-5) Image combines OpenAI's GPT-5 model with state-of-the-art image generation capabilities. It offers major improvements in reasoning, code quality, and user experience while incorporating GPT Image 1's superior instruction following.

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--quality` | Allowed values: auto, low, medium, high
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT-5 Image Mini

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT-5 Image Mini | image | none | `openrouter/openai/gpt-5-image-mini`

GPT-5 Image Mini combines OpenAI's advanced language capabilities, powered by [GPT-5 Mini](https://openrouter.ai/openai/gpt-5-mini), with GPT Image 1 Mini for efficient image generation. This natively multimodal model features superior instruction following.

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--quality` | Allowed values: auto, low, medium, high
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT-5.4 Image 2

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT-5.4 Image 2 | image | none | `openrouter/openai/gpt-5.4-image-2`

[GPT-5.4](https://openrouter.ai/openai/gpt-5.4) Image 2 combines OpenAI's GPT-5.4 model with state-of-the-art image generation capabilities from GPT Image 2. It enables rich multimodal workflows, allowing users to seamlessly move between reasoning, coding, and image generation.

Option | Constraints
-------|------------
`--aspect-ratio` | Examples: 16:9, 2:3
`--resolution` |
`--quality` | Allowed values: auto, low, medium, high
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT Image 1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT Image 1 | image | none | `openrouter/openai/gpt-image-1`

OpenAI's GPT Image 1 generates and edits images via the dedicated Images API. Features accurate text rendering, transparent backgrounds, and up to 16 reference images for edits.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:2, 2:3, auto
`--resolution` |
`--quality` | Allowed values: auto, low, medium, high
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT Image 1 Mini

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT Image 1 Mini | image | none | `openrouter/openai/gpt-image-1-mini`

A cost-efficient variant of GPT Image 1 for high-quality image generation at reduced latency and cost via OpenAI's dedicated Images API.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:2, 2:3, auto
`--resolution` |
`--quality` | Allowed values: auto, low, medium, high
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## OpenAI: GPT Image 2

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: GPT Image 2 | image | none | `openrouter/openai/gpt-image-2`

OpenAI's latest image generation model. Supports high-fidelity image generation and editing via the dedicated Images API.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:2, 2:3, 4:3, 3:4, 16:9, 9:16, 21:9, auto
`--resolution` |
`--quality` | Allowed values: auto, low, medium, high
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Qwen: Qwen Image 3

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Qwen: Qwen Image 3 | image | none | `openrouter/qwen/qwen-image-3`

Qwen Image 3 is a unified image generation and editing model from Qwen. It supports precise rendering of text and details as small as 10px, along with a richer world knowledge.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:2, 1:4, 2:1, 2:3, 3:2, 3:4, 4:1, 4:3, 4:5, 5:4, 9:16, 16:9
`--resolution` | Allowed values: 1K, 2K
`--quality` |
`--output-format` |
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 4
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Qwen: Qwen Image 3 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Qwen: Qwen Image 3 Pro | image | none | `openrouter/qwen/qwen-image-3-pro`

Qwen Image 3 Pro is an image generation and editing model from Qwen. It supports precise rendering of text and details as small as 10px, along with richer world knowledge.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:2, 1:4, 2:1, 2:3, 3:2, 3:4, 4:1, 4:3, 4:5, 5:4, 9:16, 16:9
`--resolution` | Allowed values: 1K, 2K
`--quality` |
`--output-format` |
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 4
`--seed` |
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V3

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V3 | image | none | `openrouter/recraft/recraft-v3`

Recraft V3 is an image generation model from Recraft. It supports text and image inputs with image output at ~1K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 | image | none | `openrouter/recraft/recraft-v4`

Recraft V4 is an image generation model from Recraft. It supports text and image inputs with image output at ~1K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Pro | image | none | `openrouter/recraft/recraft-v4-pro`

Recraft V4 Pro is an image generation model from Recraft. It supports text and image inputs with image output at ~2K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4 Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Pro Vector | image | none | `openrouter/recraft/recraft-v4-pro-vector`

Recraft V4 Pro Vector is the vector (SVG) variant of Recraft V4 Pro. It supports text and image inputs and produces vector image output across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4 Styles

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Styles | image | none | `openrouter/recraft/recraft-v4-styles`

Recraft V4 Styles is a style-consistent image generation model from Recraft. Every request requires at least one style reference image.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 16:9, 9:16, auto
`--num-images` | Allowed range: 1 to 6
`--input-media` | **Required.** Repeat maximum: 10

## Recraft: Recraft V4 Styles Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Styles Pro | image | none | `openrouter/recraft/recraft-v4-styles-pro`

Recraft V4 Styles Pro is a style-consistent image generation model from Recraft. Every request requires at least one style reference image.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 16:9, 9:16, auto
`--num-images` | Allowed range: 1 to 6
`--input-media` | **Required.** Repeat maximum: 10

## Recraft: Recraft V4 Styles Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Styles Pro Vector | image | none | `openrouter/recraft/recraft-v4-styles-pro-vector`

Recraft V4 Styles Pro Vector is a style-consistent vector image generation model from Recraft. Every request requires at least one style reference image.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 16:9, 9:16, auto
`--output-format` | Allowed values: svg
`--num-images` | Allowed range: 1 to 6
`--input-media` | **Required.** Repeat maximum: 10

## Recraft: Recraft V4 Styles Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Styles Vector | image | none | `openrouter/recraft/recraft-v4-styles-vector`

Recraft V4 Styles Vector is a style-consistent vector image generation model from Recraft. Every request requires at least one style reference image.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 16:9, 9:16, auto
`--output-format` | Allowed values: svg
`--num-images` | Allowed range: 1 to 6
`--input-media` | **Required.** Repeat maximum: 10

## Recraft: Recraft V4 Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4 Vector | image | none | `openrouter/recraft/recraft-v4-vector`

Recraft V4 Vector is the vector (SVG) variant of Recraft V4. It supports text and image inputs and produces vector image output across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4.1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 | image | none | `openrouter/recraft/recraft-v4.1`

Recraft V4.1 is an image generation model from Recraft tuned for high aesthetics. It supports text and image inputs with image output at ~1K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4.1 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 Pro | image | none | `openrouter/recraft/recraft-v4.1-pro`

Recraft V4.1 Pro is an image generation model from Recraft tuned for high aesthetics. It supports text and image inputs with image output at ~2K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4.1 Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 Pro Vector | image | none | `openrouter/recraft/recraft-v4.1-pro-vector`

Recraft V4.1 Pro Vector is the vector (SVG) variant of Recraft V4.1 Pro, tuned for high aesthetics. It supports text and image inputs and produces higher-resolution SVG image output across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4.1 Utility

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 Utility | image | none | `openrouter/recraft/recraft-v4.1-utility`

Recraft V4.1 Utility is a general-purpose image generation model from Recraft. It supports text and image inputs with image output at ~1K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4.1 Utility Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 Utility Pro | image | none | `openrouter/recraft/recraft-v4.1-utility-pro`

Recraft V4.1 Utility Pro is a general-purpose image generation model from Recraft. It supports text and image inputs with image output at ~2K resolution across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Recraft: Recraft V4.1 Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 Vector | image | none | `openrouter/recraft/recraft-v4.1-vector`

Recraft V4.1 Vector is the vector (SVG) variant of Recraft V4.1, tuned for high aesthetics. It supports text and image inputs and produces SVG image output across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--resolution` |
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Sourceful: Riverflow V2 Fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Sourceful: Riverflow V2 Fast | image | none | `openrouter/sourceful/riverflow-v2-fast`

Riverflow V2 Fast is the fastest variant of Sourceful's Riverflow 2.0 lineup, best for production deployments and latency-critical workflows. The Riverflow 2.0 series represents SOTA performance on image generation.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` | Allowed values: 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 4
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Sourceful: Riverflow V2 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Sourceful: Riverflow V2 Pro | image | none | `openrouter/sourceful/riverflow-v2-pro`

Riverflow V2 Pro is the most powerful variant of Sourceful's Riverflow 2.0 lineup, best for top-tier control and perfect text rendering. The Riverflow 2.0 series represents SOTA performance on image generation.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` | Allowed values: 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 10
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Sourceful: Riverflow V2.5 Fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Sourceful: Riverflow V2.5 Fast | image | none | `openrouter/sourceful/riverflow-v2.5-fast`

Riverflow V2.5 Fast is the speed-optimized variant of Sourceful's Riverflow 2.5 lineup, best for production deployments and latency-critical workflows. The Riverflow 2.5 series is a unified text-to-image and image-to-image family across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` | Allowed values: 1K, 2K
`--quality` |
`--output-format` | Allowed values: jpeg
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 4
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Sourceful: Riverflow V2.5 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Sourceful: Riverflow V2.5 Pro | image | none | `openrouter/sourceful/riverflow-v2.5-pro`

Riverflow V2.5 Pro is the most powerful variant of Sourceful's Riverflow 2.5 lineup, best for top-tier control and quality-sensitive outputs. The Riverflow 2.5 series is a unified text-to-image and image-to-image family across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 3:2, 2:3, 16:9, 9:16, 21:9, auto
`--resolution` | Allowed values: 1K, 2K, 4K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 10
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## xAI: Grok Imagine Image 2.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
xAI: Grok Imagine Image 2.0 | image | none | `openrouter/x-ai/grok-imagine-image-2.0`

Grok Imagine Image 2.0 is an image generation and editing model from xAI. It is suited for creating images from text prompts and editing images from references, with low and high fidelity options across multiple aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:4, 4:3, 9:16, 16:9, 2:3, 3:2, 9:19.5, 19.5:9, 9:20, 20:9, 1:2, 2:1, auto
`--resolution` | Allowed values: 1K, 2K
`--quality` | Allowed values: low, medium
`--output-format` |
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 3
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## SpaceXAI: Grok Imagine Image Quality

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
SpaceXAI: Grok Imagine Image Quality | image | none | `openrouter/x-ai/grok-imagine-image-quality`

Grok Imagine Image Quality is SpaceXAI's fast, high-fidelity image generation and editing model. It accepts text prompts and optional reference images, producing photorealistic outputs at 1K or 2K across a range of aspect ratios.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:4, 4:3, 9:16, 16:9, 2:3, 3:2, 9:19.5, 19.5:9, 9:20, 20:9, 1:2, 2:1, auto
`--resolution` | Allowed values: 1K, 2K
`--quality` |
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 3
`--background` | Allowed values: auto, transparent, opaque
`--output-compression` | Allowed range: 0 to 100

## Alibaba: HappyHorse 1.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Alibaba: HappyHorse 1.0 | video | none | `openrouter/alibaba/happyhorse-1.0`

HappyHorse 1.0 is a video generation model from Alibaba. It generates short videos from a text prompt, a single starting image, or a set of reference images.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 21:9, 9:21
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |

## Alibaba: HappyHorse 1.1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Alibaba: HappyHorse 1.1 | video | none | `openrouter/alibaba/happyhorse-1.1`

HappyHorse 1.1 is a video generation model from Alibaba. It generates short videos from a text prompt, a single starting image, or a set of reference images.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 21:9, 9:21
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |

## Alibaba: Wan 2.6

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Alibaba: Wan 2.6 | video | none | `openrouter/alibaba/wan-2.6`

Alibaba's most advanced video generation model, supporting over 10 visual creation capabilities in a unified system. Wan 2.6 generates 1080p video at 24fps from text, images, reference videos, or audio.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 5, 10
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Alibaba: Wan 2.7

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Alibaba: Wan 2.7 | video | none | `openrouter/alibaba/wan-2.7`

Wan 2.7 is a video generation model from Alibaba. It supports text-to-video, image-to-video with first and last frame control, and reference-to-video, where multiple reference images guide the style and content.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 2, 3, 4, 5, 6, 7, 8, 9, 10
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Alibaba: Wan 3.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Alibaba: Wan 3.0 | video | none | `openrouter/alibaba/wan-3.0`

Wan 3.0 is a video generation model from Alibaba for text-to-video, image-to-video, and reference-guided video generation. It produces 480p, 720p, or 1080p video with durations from 2 to 30 seconds.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: 480p, 720p, 1080p
`--duration` | Allowed range: 2 to 30
`--input-media` | Repeat maximum: 1; Special usage: <[first:]media-file>. To use the image as the opening frame, add the 'first:' prefix (for example, first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Alibaba: Wan 3.0 Prime

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Alibaba: Wan 3.0 Prime | video | none | `openrouter/alibaba/wan-3.0-prime`

Wan 3.0 Prime is a fast-mode variant of Wan 3.0 from Alibaba. It supports text-to-video and first-frame image-to-video generation.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: 480p, 720p, 1080p
`--duration` | Allowed range: 2 to 30
`--input-media` | Repeat maximum: 1; Special usage: <[first:]media-file>. To use the image as the opening frame, add the 'first:' prefix (for example, first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## ByteDance: Seedance 1.5 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance: Seedance 1.5 Pro | video | none | `openrouter/bytedance/seedance-1-5-pro`

ByteDance's next-generation audio-visual generation model with a 4.5B parameter Dual-Branch Diffusion Transformer architecture. Seedance 1.5 Pro generates video and audio simultaneously in a single unified pass — eliminating the timing issues between audio and video.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:4, 9:16, 9:21, 4:3, 16:9, 21:9
`--resolution` | Allowed values: 480p, 720p, 1080p
`--duration` | Allowed values: 4, 5, 6, 7, 8, 9, 10, 11, 12
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## ByteDance: Seedance 2.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance: Seedance 2.0 | video | none | `openrouter/bytedance/seedance-2.0`

Seedance 2.0 is a video generation model from ByteDance. It supports text-to-video, image-to-video with first and last frame control, and multimodal reference-to-video. It is particularly strong at preserving character consistency.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:4, 9:16, 4:3, 16:9, 21:9, 9:21
`--resolution` | Allowed values: 480p, 720p, 1080p, 4K
`--duration` | Allowed values: 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## ByteDance: Seedance 2.0 Fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance: Seedance 2.0 Fast | video | none | `openrouter/bytedance/seedance-2.0-fast`

Seedance 2.0 Fast is a video generation model from ByteDance. It supports text-to-video, image-to-video with first and last frame control, and multimodal reference-to-video. It prioritizes generation speed and lower cost.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:4, 9:16, 4:3, 16:9, 21:9, 9:21
`--resolution` | Allowed values: 480p, 720p
`--duration` | Allowed values: 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## ByteDance: Seedance 2.0 Mini

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance: Seedance 2.0 Mini | video | none | `openrouter/bytedance/seedance-2.0-mini`

Seedance 2.0 Mini is a video generation model from ByteDance. It supports text-to-video, image-to-video with first and last frame control, and multimodal reference-to-video with image, video, and audio inputs.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 3:4, 9:16, 4:3, 16:9, 21:9, 9:21
`--resolution` | Allowed values: 480p, 720p
`--duration` | Allowed values: 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## ByteDance: Seedance 2.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance: Seedance 2.5 | video | none | `openrouter/bytedance/seedance-2.5`

Seedance 2.5 is a video generation model from ByteDance. It is suited for long-form storytelling, multimodal reference-based generation, video editing, and video extension. It supports first-frame and first-and-last-frame control.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 4:3, 1:1, 3:4, 9:16, 21:9
`--resolution` | Allowed values: 480p, 720p
`--duration` | Allowed values: 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Google: Veo 3.1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Veo 3.1 | video | none | `openrouter/google/veo-3.1`

Google's state-of-the-art video generation model, built for maximum visual fidelity in final production cuts. Veo 3.1 generates high-quality 1080p video from text or image prompts with native synchronized audio —...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p, 4K
`--duration` | Allowed values: 4, 6, 8
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Google: Veo 3.1 Fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Veo 3.1 Fast | video | none | `openrouter/google/veo-3.1-fast`

Google's mid-tier video generation model balancing speed and quality. Veo 3.1 Fast generates high-quality video from text or image prompts with native synchronized audio, offering faster turnaround than Veo 3.1...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p, 4K
`--duration` | Allowed values: 4, 6, 8
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Google: Veo 3.1 Lite

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Google: Veo 3.1 Lite | video | none | `openrouter/google/veo-3.1-lite`

Google's most cost-effective video generation model, designed for high-volume applications and rapid iteration. Veo 3.1 Lite generates 720p and 1080p video from text or image prompts with native synchronized audio...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 4, 6, 8
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |
`--generate-audio` |

## Kling: Video v3.0 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling: Video v3.0 Pro | video | none | `openrouter/kwaivgi/kling-v3.0-pro`

Kling v3.0 Pro is Kuaishou's premium video generation model, offering higher visual quality than the Standard tier. It supports text-to-video and image-to-video workflows, with first-frame and last-frame control for precise...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p
`--duration` | Allowed values: 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--generate-audio` |

## Kling: Video v3.0 Standard

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling: Video v3.0 Standard | video | none | `openrouter/kwaivgi/kling-v3.0-std`

Kling v3.0 Standard is a video generation model from Kuaishou. It supports text-to-video and image-to-video workflows, with first-frame and last-frame control for guided scene composition. Clips range from 3 to...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p
`--duration` | Allowed values: 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--generate-audio` |

## Kling: Video O1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Kling: Video O1 | video | none | `openrouter/kwaivgi/kling-video-o1`

Kling Video O1 is a video generation model from Kuaishou. It supports text and image inputs with video output, enabling text-to-video and image-to-video workflows. It is suited for cinematic content...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p
`--duration` | Allowed values: 5, 10
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--generate-audio` |

## MiniMax: Hailuo 2.3

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
MiniMax: Hailuo 2.3 | video | none | `openrouter/minimax/hailuo-2.3`

Hailuo 2.3 is a video generation model from MiniMax. It accepts text prompts and reference images as input and generates video output, supporting both text-to-video and image-to-video workflows. It is...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9
`--resolution` | Allowed values: 1080p
`--duration` | Allowed values: 6, 10
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.

## MiniMax: H3

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
MiniMax: H3 | video | none | `openrouter/minimax/hailuo-3`

MiniMax H3 is a lightweight, open-weights video generation model from MiniMax. It is designed for precise multimodal editing and controlled content generation, including instruction-guided edits, text and brand rendering, and...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 21:9, 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: 2K
`--duration` | Allowed values: 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--generate-audio` |

## OpenAI: Sora 2 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
OpenAI: Sora 2 Pro | video | none | `openrouter/openai/sora-2-pro`

OpenAI's flagship video generation model, delivering production-quality video with physics-accurate motion, synchronized audio, and world-state persistence across shots. Sora 2 Pro follows intricate multi-shot instructions while maintaining consistent spatial relationships...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p, 1080p
`--duration` | Allowed values: 4, 8, 12, 16, 20
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--generate-audio` |

## Runway: Aleph 2.0

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Runway: Aleph 2.0 | video | none | `openrouter/runway/aleph-2`

Runway Aleph 2.0 is an in-context video editing model from Runway. It applies text instructions to existing footage.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 4:3, 3:2, 1:1, 2:3, 3:4, 9:16, 21:9
`--input-media` | **Required.** Repeat maximum: 1
`--seed` |

## Runway: Gen-4.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Runway: Gen-4.5 | video | none | `openrouter/runway/gen-4.5`

Runway Gen-4.5 is a video generation model from Runway for text-to-video and image-to-video workflows. It is designed for cinematic scene creation with strong motion quality, visual fidelity, and prompt adherence....

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16
`--resolution` | Allowed values: 720p
`--duration` | Allowed values: 2, 3, 4, 5, 6, 7, 8, 9, 10
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.
`--seed` |

## SpaceXAI: Grok Imagine Video

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
SpaceXAI: Grok Imagine Video | video | none | `openrouter/x-ai/grok-imagine-video`

Grok Imagine Video is SpaceXAI's fast, text-, image-, and reference-conditioned video generation model. It produces short videos (1–15 seconds, 24 fps) at 480p or 720p across seven aspect ratios -...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 3:2, 2:3
`--resolution` | Allowed values: 480p, 720p
`--duration` | Allowed values: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.

## SpaceXAI: Grok Imagine Video 1.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
SpaceXAI: Grok Imagine Video 1.5 | video | none | `openrouter/x-ai/grok-imagine-video-1.5`

Grok Imagine Video 1.5 is a video generation model from SpaceXAI. It creates videos from text prompts, with an optional starting image to guide the scene. It can direct subject...

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1, 4:3, 3:4, 3:2, 2:3
`--resolution` | Allowed values: 480p, 720p, 1080p
`--duration` | Allowed values: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15
`--input-media` | Special usage: <[first:\|last:]media-file>. For the media to be used for the opening or closing frame, add a 'first:' or 'last:' prefix (e.g., first:image.png). See online docs for details on specifying frames for input media: https://bildomat.com/docs.

## Microsoft: MAI Image 2.6

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Microsoft: MAI Image 2.6 | image | none | `openrouter/microsoft/mai-image-2.6`

Generate or edit an image using up to five image references.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, 3:2, 2:3, auto
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 5

## Microsoft: MAI Image 2.6 Flash

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Microsoft: MAI Image 2.6 Flash | image | none | `openrouter/microsoft/mai-image-2.6-flash`

Generate or edit an image using up to five image references.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, 3:2, 2:3, auto
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 5

## MiniMax: Hailuo 3 Max

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
MiniMax: Hailuo 3 Max | video | none | `openrouter/minimax/hailuo-3-max`

Generate video from text or opening and closing frames.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 21:9, 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: 768p, 480p
`--duration` | Allowed range: 5 to 15
`--input-media` | Repeat maximum: 2; Prefix images with first: or last: to select the opening or closing frame. See https://openrouter.ai/docs/guides/overview/multimodal/video-generation.
`--watermark` |

## Black Forest Labs: FLUX Video Edit

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX Video Edit | video | none | `openrouter/black-forest-labs/flux-video-edit`

Edit an input video using a text prompt.

Option | Constraints
-------|------------
`--input-media` | **Required.** Repeat maximum: 1; Supply one public HTTPS video URL. Local video files are rejected by this route. See https://openrouter.ai/docs/guides/overview/multimodal/video-generation.
`--safety-tolerance` | Allowed range: 0 to 5

## Black Forest Labs: FLUX Video Upscale

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX Video Upscale | video | none | `openrouter/black-forest-labs/flux-video-upscale`

Increase an input video resolution.

Option | Constraints
-------|------------
`--input-media` | **Required.** Repeat maximum: 1; Supply one public HTTPS video URL. Local video files are rejected by this route. See https://openrouter.ai/docs/guides/overview/multimodal/video-generation.
`--safety-tolerance` | Allowed range: 0 to 5
`--upscale-factor` | Allowed range: 1.5 to 3
`--creativity` | Allowed values: 0, 1

## HeyGen: Avatar IV

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
HeyGen: Avatar IV | video | none | `openrouter/heygen/avatar-iv`

Animate a portrait and speak the prompt using a selected provider voice.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 16:9, 9:16, 1:1
`--resolution` | Allowed values: 720p, 1080p
`--input-media` | **Required.** Repeat maximum: 1; Supply one portrait image. The prompt is spoken using --voice-id. Audio input is not supported.
`--voice-id` | **Required.** Examples: 16a09e4706f74997ba4ed05ea11470f6, 6be73833ef9a4eb0aeee399b8fe9d62b; The example IDs have been verified with this endpoint. Find other voice IDs at https://developers.heygen.com/reference/list-voices.
`--motion-prompt` |
`--expressiveness` | Examples: low, medium, high
`--image-fit` | Examples: contain, cover
`--remove-background` |
`--voice-speed` | Allowed range: 0.5 to 1.5
`--voice-pitch` | Pitch adjustment from -50 to +50 semitones.
`--voice-volume` | Allowed range: 0 to 1
`--voice-locale` | Examples: en-US, en-GB

## Black Forest Labs: FLUX 3 Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Black Forest Labs: FLUX 3 Image | image | none | `openrouter/black-forest-labs/flux-3-image`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 21:9, 2:1, 16:9, 3:2, 7:5, 4:3, 5:4, 1:1, 4:5, 3:4, 5:7, 2:3, 9:16, 1:2, 9:21, auto
`--resolution` | Allowed values: 768, 1K, 1.5K, 2K, 4K
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 10
`--safety-tolerance` | Allowed range: 0 to 4

## ByteDance Seed: Seedream 5.0 Flash

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ByteDance Seed: Seedream 5.0 Flash | image | none | `openrouter/bytedance-seed/seedream-5-0-flash`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 1:2, 2:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 9:19.5, 19.5:9, 9:20, 20:9, 9:21, 21:9, auto
`--resolution` | Allowed values: 1K, 2K
`--num-images` | Allowed range: 1 to 1
`--input-media` | Repeat maximum: 14
`--seed` |

## InclusionAI: Ming Image 0.1 Design

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
InclusionAI: Ming Image 0.1 Design | image | none | `openrouter/inclusionai/ming-image-0.1-design`

Option | Constraints
-------|------------
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 1

## InclusionAI: Ming Image 0.1 Design Layer

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
InclusionAI: Ming Image 0.1 Design Layer | image | none | `openrouter/inclusionai/ming-image-0.1-design-layer`

Generate separate transparent layer images from a prompt and one reference image.

Option | Constraints
-------|------------
`--output-format` | Allowed values: png, webp
`--num-images` | Allowed range: 1 to 1
`--input-media` | **Required.** Repeat maximum: 1; Supply one reference image. Each returned layer is saved as a separate image.

## Recraft: Recraft V4.1 Flash

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft: Recraft V4.1 Flash | image | none | `openrouter/recraft/recraft-v4.1-flash`

Generate raster images from a text prompt.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 4:3, 3:4, 16:9, 9:16, auto
`--num-images` | Allowed range: 1 to 6

## HeyGen: HeyGen Video 1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
HeyGen: HeyGen Video 1 | video | none | `openrouter/heygen/heygen-video-1`

Generate video with audio from text, an opening image, or image references.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 21:9, 16:9, 4:3, 1:1, 3:4, 9:16
`--resolution` | Allowed values: 480p, 768p
`--duration` | Allowed range: 5 to 15
`--input-media` | Repeat maximum: 9; Use up to nine image references, or one opening image with first:image.png. Do not combine an opening image with references. Closing frames, video references, and audio references are not supported by this configuration.
`--seed` |

Revised 2026-10-06
