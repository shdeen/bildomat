# Providers and models

The catalog below describes Bildomat’s supported model identifiers and options. An option not listed for a model is omitted with a notice when supplied. “No declared constraint” means Bildomat leaves that check to the provider; it does not promise unlimited provider support.

Use the installed executable to check its catalog:

```sh
bild list --providers
bild list --models
bild info google --json
```

| Provider | ID | Image models | Video models | Credential variable | Default model |
| --- | --- | --- | --- | --- | --- |
| [Black Forest Labs](catalog/bfl.md) | `bfl` | 22 | 3 | `BFL_API_KEY` | `flux-2-pro` |
| [Google](catalog/google.md) | `google` | 5 | 5 | `GOOGLE_API_KEY` | `gemini-3.1-flash-image` |
| [Kling](catalog/kling.md) | `kling` | 4 | 6 | `KLING_API_KEY` | `kling-v3` |
| [OpenAI](catalog/openai.md) | `openai` | 7 | 2 | `OPENAI_API_KEY` | `gpt-image-2.5-flare` |
| [Recraft](catalog/recraft.md) | `recraft` | 21 | 0 | `RECRAFT_API_KEY` | `recraftv4_1` |
| [Sourceful](catalog/sourceful.md) | `sourceful` | 4 | 0 | `SOURCEFUL_API_KEY` | `riverflow-2.5-fast` |
| [xAI](catalog/xai.md) | `xai` | 4 | 3 | `XAI_API_KEY` | `grok-imagine-image` |
| [OpenRouter](catalog/openrouter.md) | `openrouter` | 57 | 30 | `OPENROUTER_API_KEY` | `google/gemini-3.1-flash-image` |
| **173 models** | | **124** | **49** | | |

A run without `--model`, and without `default-model` in the configuration file, uses the default model of the first provider in this table whose credential is available. The table's order is the listing order of `bild list`: alphabetical by provider name, with the aggregator last. Prefix the default model with its provider ID to name it explicitly.

This set covers 173 models across eight providers: 124 image models and 49 video models. OpenRouter is an aggregator: in a full key such as `openrouter/google/veo-3.1`, the vendor name does not switch to that vendor’s direct service. Without the `openrouter/` prefix, a vendor name that is also a Bildomat provider ID can select the direct service instead; see [model specifiers](model-specifiers.md). Use the full key for stable selection and inspect the exact integration’s options.

Nearly every model needs a prompt. A few do not require one, and a run on one of those needs no prompt argument; `bild info MODEL` marks such a model, and so does its entry in these pages.

Model option values are unset unless supplied or derived; provider defaults apply to omitted values. A “required” mark means that the model needs the option. Bildomat does not check for it before submission, so a request without it reaches the provider, which can reject it. The exception is the input video of the direct BFL video tools, FLUX Video Edit and FLUX Video Upscale: a run on either one without a video fails before submission. Conditional requirements appear alongside the affected option.

See [flag types and shorthands](generation-flags.md), [adjustment rules](parameter-adjustment.md), and [input-media behavior](input-media.md) for rules shared across the catalog.

Revised 2026-10-07
