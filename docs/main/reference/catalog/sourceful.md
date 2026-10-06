# Sourceful Models

Every model that `bild` offers from Sourceful (provider ID `sourceful`, API key variable `SOURCEFUL_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info sourceful/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

4 image models, 0 video models.

## Contents

- [Riverflow 2.5 Pro](#riverflow-25-pro)
- [Riverflow 2.5 Fast](#riverflow-25-fast)
- [Riverflow 2 Fast](#riverflow-2-fast)
- [Riverflow 2 Pro](#riverflow-2-pro)

## Riverflow 2.5 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Riverflow 2.5 Pro | image | `riverflow`, `riverflow-v2.5-pro`, `riverflow-pro` | `sourceful/riverflow-2.5-pro`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: auto, 21:9, 16:9, 3:2, 4:3, 5:4, 1:1, 4:5, 3:4, 2:3, 9:16
`--resolution` | Allowed values: 1K, 2K, 4K
`--output-format` | Allowed values: webp, png, jpg, jpeg
`--thinking-level` | Allowed values: low, medium, high, xhigh
`--input-media` | Repeat maximum: 10
`--background` | Allowed values: original, transparent
`--prompt-upsampling` |

## Riverflow 2.5 Fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Riverflow 2.5 Fast | image | `riverflow-v2.5-fast` | `sourceful/riverflow-2.5-fast`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: auto, 21:9, 16:9, 3:2, 4:3, 5:4, 1:1, 4:5, 3:4, 2:3, 9:16
`--resolution` | Allowed values: 1K, 2K, 4K
`--output-format` | Allowed values: webp, png, jpg, jpeg
`--thinking-level` | Allowed values: low, medium, high, xhigh
`--input-media` | Repeat maximum: 10
`--background` | Allowed values: original, transparent
`--prompt-upsampling` |

## Riverflow 2 Fast

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Riverflow 2 Fast | image | none | `sourceful/riverflow-2-fast`

Generate images from text or edit reference images using Riverflow 2 Fast.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: auto, 21:9, 16:9, 3:2, 4:3, 5:4, 1:1, 4:5, 3:4, 2:3, 9:16; Use auto to infer the ratio from the prompt or reference image. The direct API documents 1:1 as its default.
`--resolution` | Allowed values: 1K, 2K; Fast supports 1K and 2K only. The shared API schema also lists 4K, but the direct Fast API rejects it; 4K is supported by Pro.
`--input-media` | Repeat maximum: 4; Accepts local images or public HTTPS image URLs, preserving their order. Local images are sent as data URIs; URL inputs remain URLs. Excess references are omitted with a notice. Frame prefixes are removed with a notice. Fast accepts four references, although the shared API schema allows ten. The direct API enforces four, matching Sourceful's model-specific documentation. Sourceful documents a 4.5 MB request-body limit, but Fast accepted approximately 5.5 MB and 22 MB requests on 2026-10-05. The actual maximum is unverified. Bildomat forwards the request and reports any provider refusal; public URLs avoid embedding large image data.
`--prompt-upsampling` | When enabled, the API enhances the prompt using use-case context. The API default is false. Prompts must contain at least two characters; the direct API documentation says text beyond 24000 characters is truncated before processing.

## Riverflow 2 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Riverflow 2 Pro | image | none | `sourceful/riverflow-2-pro`

Generate images from text or edit reference images using Riverflow 2 Pro.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: auto, 21:9, 16:9, 3:2, 4:3, 5:4, 1:1, 4:5, 3:4, 2:3, 9:16; Use auto to infer the ratio from the prompt or reference image. The direct API documents 1:1 as its default.
`--resolution` | Allowed values: 1K, 2K, 4K; Pro supports 1K, 2K, and 4K.
`--input-media` | Repeat maximum: 10; Accepts local images or public HTTPS image URLs, preserving their order. Local images are sent as data URIs; URL inputs remain URLs. Excess references are omitted with a notice. Frame prefixes are removed with a notice. Pro accepts ten references. Sourceful documents a 4.5 MB request-body limit and recommends URLs for large images. Bildomat forwards the request and reports any provider refusal; Pro's actual maximum request size has not been verified.
`--prompt-upsampling` | When enabled, the API enhances the prompt using use-case context. The API default is false. Prompts must contain at least two characters; the direct API documentation says text beyond 24000 characters is truncated before processing.

Revised 2026-10-06
