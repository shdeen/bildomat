# Sourceful Models

Every model that `bild` offers from Sourceful (provider ID `sourceful`, key variable `SOURCEFUL_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info sourceful/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

2 image models, 0 video models.

## Contents

- [Riverflow 2.5 Pro](#riverflow-25-pro)
- [Riverflow 2.5 Fast](#riverflow-25-fast)

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

Revised 2026-10-05
