# Recraft Models

Every model that `bild` offers from Recraft (provider ID `recraft`, API key variable `RECRAFT_API_KEY`), with every option each model accepts and the constraints the model declares. An option the provider requires opens with **Required.**; every other option is optional. `bild info recraft/<model>` prints the same facts for the binary you run. What the constraints mean is described in [Parameter adjustment](../parameter-adjustment.md), the flags in [Generation flags](../generation-flags.md), and how to name a model in [Model specifiers](../model-specifiers.md).

21 image models, 0 video models.

## Contents

- [Recraft V4.1](#recraft-v41)
- [Recraft V4.1 Utility](#recraft-v41-utility)
- [Recraft V4](#recraft-v4)
- [Recraft V4.1 Pro](#recraft-v41-pro)
- [Recraft V4.1 Utility Pro](#recraft-v41-utility-pro)
- [Recraft V4 Pro](#recraft-v4-pro)
- [Recraft V3](#recraft-v3)
- [Recraft V2](#recraft-v2)
- [Recraft V4.1 Vector](#recraft-v41-vector)
- [Recraft V4.1 Pro Vector](#recraft-v41-pro-vector)
- [Recraft V4.1 Utility Vector](#recraft-v41-utility-vector)
- [Recraft V4.1 Utility Pro Vector](#recraft-v41-utility-pro-vector)
- [Recraft V4 Vector](#recraft-v4-vector)
- [Recraft V4 Pro Vector](#recraft-v4-pro-vector)
- [Recraft V3 Vector](#recraft-v3-vector)
- [Recraft V2 Vector](#recraft-v2-vector)
- [Recraft V4 Styles](#recraft-v4-styles)
- [Recraft V4 Styles Pro](#recraft-v4-styles-pro)
- [Recraft V4 Styles Vector](#recraft-v4-styles-vector)
- [Recraft V4 Styles Pro Vector](#recraft-v4-styles-pro-vector)
- [Recraft V4.1 Flash](#recraft-v41-flash)

## Recraft V4.1

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 | image | `recraft-v4.1`, `recraft-4.1` | `recraft/recraftv4_1`

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1536x768, 768x1536, 1280x832, 832x1280, 1216x896, 896x1216, 1152x896, 896x1152, 832x1344, 1280x896, 896x1280, 1344x768, 768x1344
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4.1 Utility

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Utility | image | `recraft-v4.1-utility`, `recraft-4.1-utility` | `recraft/recraftv4_1_utility`

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1536x768, 768x1536, 1280x832, 832x1280, 1216x896, 896x1216, 1152x896, 896x1152, 832x1344, 1280x896, 896x1280, 1344x768, 768x1344
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 | image | `recraft-v4`, `recraft-4` | `recraft/recraftv4`

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1536x768, 768x1536, 1280x832, 832x1280, 1216x896, 896x1216, 1152x896, 896x1152, 832x1344, 1280x896, 896x1280, 1344x768, 768x1344
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4.1 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Pro | image | `recraft-v4.1-pro`, `recraft-4.1-pro` | `recraft/recraftv4_1_pro`

Option | Constraints
-------|------------
`--size` | Allowed values: 2048x2048, 3072x1536, 1536x3072, 2560x1664, 1664x2560, 2432x1792, 1792x2432, 2304x1792, 1792x2304, 1664x2688, 2560x1792, 1792x2560, 2688x1536, 1536x2688
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4.1 Utility Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Utility Pro | image | `recraft-v4.1-utility-pro`, `recraft-4.1-utility-pro` | `recraft/recraftv4_1_utility_pro`

Option | Constraints
-------|------------
`--size` | Allowed values: 2048x2048, 3072x1536, 1536x3072, 2560x1664, 1664x2560, 2432x1792, 1792x2432, 2304x1792, 1792x2304, 1664x2688, 2560x1792, 1792x2560, 2688x1536, 1536x2688
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4 Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Pro | image | `recraft-v4-pro`, `recraft-4-pro` | `recraft/recraftv4_pro`

Option | Constraints
-------|------------
`--size` | Allowed values: 2048x2048, 3072x1536, 1536x3072, 2560x1664, 1664x2560, 2432x1792, 1792x2432, 2304x1792, 1792x2304, 1664x2688, 2560x1792, 1792x2560, 2688x1536, 1536x2688
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V3

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V3 | image | `recraft-v3`, `recraft-3` | `recraft/recraftv3`

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 2048x1024, 1024x2048, 1536x1024, 1024x1536, 1365x1024, 1024x1365, 1280x1024, 1024x1280, 1024x1707, 1434x1024, 1024x1434, 1820x1024, 1024x1820
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |
`--negative-prompt` |

## Recraft V2

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V2 | image | `recraft-v2`, `recraft-2` | `recraft/recraftv2`

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 2048x1024, 1024x2048, 1536x1024, 1024x1536, 1365x1024, 1024x1365, 1280x1024, 1024x1280, 1024x1707, 1434x1024, 1024x1434, 1820x1024, 1024x1820
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--seed` |
`--negative-prompt` |

## Recraft V4.1 Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Vector | image | `recraft-v4.1-vector`, `recraft-4.1-vector` | `recraft/recraftv4_1_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4.1 Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Pro Vector | image | `recraft-v4.1-pro-vector`, `recraft-4.1-pro-vector` | `recraft/recraftv4_1_pro_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4.1 Utility Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Utility Vector | image | `recraft-v4.1-utility-vector`, `recraft-4.1-utility-vector` | `recraft/recraftv4_1_utility_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4.1 Utility Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Utility Pro Vector | image | `recraft-v4.1-utility-pro-vector`, `recraft-4.1-utility-pro-vector` | `recraft/recraftv4_1_utility_pro_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4 Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Vector | image | `recraft-v4-vector`, `recraft-4-vector` | `recraft/recraftv4_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V4 Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Pro Vector | image | `recraft-v4-pro-vector`, `recraft-4-pro-vector` | `recraft/recraftv4_pro_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |

## Recraft V3 Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V3 Vector | image | `recraft-v3-vector` | `recraft/recraftv3_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--input-media` | Repeat maximum: 1
`--strength` | Allowed range: 0 to 1; Image editing requires --strength.
`--seed` |
`--negative-prompt` |

## Recraft V2 Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V2 Vector | image | `recraft-v2-vector` | `recraft/recraftv2_vector`

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--seed` |
`--negative-prompt` |

## Recraft V4 Styles

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Styles | image | none | `recraft/recraftv4_styles`

Generate images using an existing Recraft style ID.

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1536x768, 768x1536, 1280x832, 832x1280, 1216x896, 896x1216, 1152x896, 896x1152, 832x1344, 1280x896, 896x1280, 1344x768, 768x1344
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--seed` |
`--style-id` | **Required.** Use a Recraft style UUID compatible with this model, not a style name. See https://www.recraft.ai/docs/api-reference/styles.
`--style-match` | Examples: precise, flexible; precise follows the style closely; flexible permits more variation. See https://www.recraft.ai/docs/recraft-models/recraft-v4-styles.

## Recraft V4 Styles Pro

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Styles Pro | image | none | `recraft/recraftv4_styles_pro`

Generate images using an existing Recraft style ID.

Option | Constraints
-------|------------
`--size` | Allowed values: 2048x2048, 3072x1536, 1536x3072, 2560x1664, 1664x2560, 2432x1792, 1792x2432, 2304x1792, 1792x2304, 1664x2688, 2560x1792, 1792x2560, 2688x1536, 1536x2688
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--num-images` | Allowed range: 1 to 6
`--seed` |
`--style-id` | **Required.** Use a Recraft style UUID compatible with this model, not a style name. See https://www.recraft.ai/docs/api-reference/styles.
`--style-match` | Examples: precise, flexible; precise follows the style closely; flexible permits more variation. See https://www.recraft.ai/docs/recraft-models/recraft-v4-styles.

## Recraft V4 Styles Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Styles Vector | image | none | `recraft/recraftv4_styles_vector`

Generate images using an existing Recraft style ID.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--seed` |
`--style-id` | **Required.** Use a Recraft style UUID compatible with this model, not a style name. See https://www.recraft.ai/docs/api-reference/styles.
`--style-match` | Examples: precise, flexible; precise follows the style closely; flexible permits more variation. See https://www.recraft.ai/docs/recraft-models/recraft-v4-styles.

## Recraft V4 Styles Pro Vector

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4 Styles Pro Vector | image | none | `recraft/recraftv4_styles_pro_vector`

Generate images using an existing Recraft style ID.

Option | Constraints
-------|------------
`--aspect-ratio` | Allowed values: 1:1, 2:1, 1:2, 3:2, 2:3, 4:3, 3:4, 5:4, 4:5, 6:10, 14:10, 10:14, 16:9, 9:16
`--num-images` | Allowed range: 1 to 6
`--seed` |
`--style-id` | **Required.** Use a Recraft style UUID compatible with this model, not a style name. See https://www.recraft.ai/docs/api-reference/styles.
`--style-match` | Examples: precise, flexible; precise follows the style closely; flexible permits more variation. See https://www.recraft.ai/docs/recraft-models/recraft-v4-styles.

## Recraft V4.1 Flash

Name | Medium | Aliases | Full ID
-----|--------|---------|--------
Recraft V4.1 Flash | image | `recraftv4_1_flash_raster` | `recraft/recraftv4_1_flash`

Generate raster images from a text prompt.

Option | Constraints
-------|------------
`--size` | Allowed values: 1024x1024, 1536x768, 768x1536, 1280x832, 832x1280, 1216x896, 896x1216, 1152x896, 896x1152, 832x1344, 1280x896, 896x1280, 1344x768, 768x1344
`--aspect-ratio` | Examples: 16:9, 2:3; Rule: Selects the supported image size closest to the requested aspect ratio.
`--resolution` | Rule: Selects a supported image size using the requested resolution.
`--output-format` | Allowed values: png, webp
`--num-images` | Allowed range: 1 to 6
`--seed` |

Revised 2026-10-06
