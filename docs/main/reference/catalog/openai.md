# OpenAI Models

Every model that `bild` offers from OpenAI (provider ID `openai`, key variable `OPENAI_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info openai/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

7 image models, 2 video models.

## Contents

- [GPT Image 2.5 Flare](#gpt-image-25-flare)
- [GPT Image 2.5 Sunburst](#gpt-image-25-sunburst)
- [GPT Image 2](#gpt-image-2)
- [GPT Image 1.5](#gpt-image-15)
- [GPT Image 1](#gpt-image-1)
- [ChatGPT Image](#chatgpt-image)
- [GPT Image 1 Mini](#gpt-image-1-mini)
- [Sora 2](#sora-2)
- [Sora 2 Pro](#sora-2-pro)

## GPT Image 2.5 Flare

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
GPT Image 2.5 Flare | image | `gpt`, `gpt-2.5-flare` | `openai/gpt-image-2.5-flare`

Fast image generation and editing for everyday creative work.

Option | Constraints
-------|------------
`--size` | <W>x<H> - Requirements: aspect ratio between 1:3 and 3:1, maximum edge: 3840, total pixels (width x height): 655360-8294400, 16-px increments, long edge of 1536 when the size is derived from an aspect ratio
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, xhigh, max, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## GPT Image 2.5 Sunburst

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
GPT Image 2.5 Sunburst | image | `gpt-2.5-sunburst` | `openai/gpt-image-2.5-sunburst`

Generate images and make precise edits from text and image references.

Option | Constraints
-------|------------
`--size` | <W>x<H> - Requirements: aspect ratio between 1:3 and 3:1, maximum edge: 3840, total pixels (width x height): 655360-8294400, 16-px increments, long edge of 1536 when the size is derived from an aspect ratio
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, xhigh, max, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## GPT Image 2

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
GPT Image 2 | image | `gpt-2` | `openai/gpt-image-2`

State-of-the-art image generation model.

Option | Constraints
-------|------------
`--size` | <W>x<H> - Requirements: aspect ratio between 1:3 and 3:1, maximum edge: 3840, total pixels (width x height): 655360-8294400, 16-px increments, long edge of 1536 when the size is derived from an aspect ratio
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## GPT Image 1.5

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
GPT Image 1.5 | image | `gpt-1.5` | `openai/gpt-image-1.5`

Previous image generation model, with better instruction following and adherence to prompts.

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1024x1536, 1536x1024
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## GPT Image 1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
GPT Image 1 | image | `gpt-1` | `openai/gpt-image-1`

Generate or edit images with GPT Image 1.

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1024x1536, 1536x1024
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## ChatGPT Image

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
ChatGPT Image | image | `chatgpt` | `openai/chatgpt-image-latest`

Previous image model used in ChatGPT.

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1024x1536, 1536x1024
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## GPT Image 1 Mini

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
GPT Image 1 Mini | image | `gpt-1-mini` | `openai/gpt-image-1-mini`

A cost-efficient version of GPT Image 1.

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1024x1536, 1536x1024
`--aspect-ratio` | 
`--resolution` | 
`--quality` | Allowed values: low, medium, high, auto
`--output-format` | Allowed values: png, jpeg, webp
`--num-images` | Allowed range: 1 to 10
`--input-media` | Repeat maximum: 16
`--background` | Allowed values: transparent, opaque, auto
`--output-compression` | Allowed range: 0 to 100
`--moderation-level` | Allowed values: low, auto

## Sora 2

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Sora 2 | video | `sora` | `openai/sora-2`

Flagship video generation with synced audio, creating richly detailed, dynamic clips from natural language or images.

Option | Constraints
-------|------------
`--size` | Allowed values: 1280x720, 720x1280
`--aspect-ratio` | 
`--resolution` | 
`--duration` | Allowed values: 4, 8, 12, 16, 20
`--input-media` | Repeat maximum: 1

## Sora 2 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Sora 2 Pro | video | `sora-pro` | `openai/sora-2-pro`

State-of-the-art, most advanced media generation model, generating videos with synced audio.

Option | Constraints
-------|------------
`--size` | Allowed values: 1280x720, 720x1280, 1792x1024, 1024x1792, 1920x1080, 1080x1920
`--aspect-ratio` | 
`--resolution` | 
`--duration` | Allowed values: 4, 8, 12, 16, 20
`--input-media` | Repeat maximum: 1

Revised 2026-10-02
