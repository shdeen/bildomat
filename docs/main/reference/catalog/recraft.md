# Recraft Models

Provider ID: `recraft`. Credential: `api-keys.recraft` in the configuration file, or else the `RECRAFT_API_KEY` environment variable. Default model: `recraftv4_1`. Provider documentation: <https://www.recraft.ai/docs/api-reference>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat's adjustments; provider requirements can also apply.

## Models

- [`recraft/recraftv4_1`](#recraftv4_1) — image.
- [`recraft/recraftv4_1_utility`](#recraftv4_1_utility) — image.
- [`recraft/recraftv4`](#recraftv4) — image.
- [`recraft/recraftv4_1_pro`](#recraftv4_1_pro) — image.
- [`recraft/recraftv4_1_utility_pro`](#recraftv4_1_utility_pro) — image.
- [`recraft/recraftv4_pro`](#recraftv4_pro) — image.
- [`recraft/recraftv3`](#recraftv3) — image.
- [`recraft/recraftv2`](#recraftv2) — image.
- [`recraft/recraftv4_1_vector`](#recraftv4_1_vector) — image.
- [`recraft/recraftv4_1_pro_vector`](#recraftv4_1_pro_vector) — image.
- [`recraft/recraftv4_1_utility_vector`](#recraftv4_1_utility_vector) — image.
- [`recraft/recraftv4_1_utility_pro_vector`](#recraftv4_1_utility_pro_vector) — image.
- [`recraft/recraftv4_vector`](#recraftv4_vector) — image.
- [`recraft/recraftv4_pro_vector`](#recraftv4_pro_vector) — image.
- [`recraft/recraftv3_vector`](#recraftv3_vector) — image.
- [`recraft/recraftv2_vector`](#recraftv2_vector) — image.
- [`recraft/recraftv4_styles`](#recraftv4_styles) — image.
- [`recraft/recraftv4_styles_pro`](#recraftv4_styles_pro) — image.
- [`recraft/recraftv4_styles_vector`](#recraftv4_styles_vector) — image.
- [`recraft/recraftv4_styles_pro_vector`](#recraftv4_styles_pro_vector) — image.

## recraftv4_1

Recraft V4.1. Output: image.

Full key: `recraft/recraftv4_1`. Aliases: `recraft-v4.1`, `recraft-4.1`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `1024x1024`, `1536x768`, `768x1536`, `1280x832`, `832x1280`, `1216x896`, `896x1216`, `1152x896`, `896x1152`, `832x1344`, `1280x896`, `896x1280`, `1344x768`, `768x1344` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_1_utility

Recraft V4.1 Utility. Output: image.

Full key: `recraft/recraftv4_1_utility`. Aliases: `recraft-v4.1-utility`, `recraft-4.1-utility`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `1024x1024`, `1536x768`, `768x1536`, `1280x832`, `832x1280`, `1216x896`, `896x1216`, `1152x896`, `896x1152`, `832x1344`, `1280x896`, `896x1280`, `1344x768`, `768x1344` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4

Recraft V4. Output: image.

Full key: `recraft/recraftv4`. Aliases: `recraft-v4`, `recraft-4`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `1024x1024`, `1536x768`, `768x1536`, `1280x832`, `832x1280`, `1216x896`, `896x1216`, `1152x896`, `896x1152`, `832x1344`, `1280x896`, `896x1280`, `1344x768`, `768x1344` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_1_pro

Recraft V4.1 Pro. Output: image.

Full key: `recraft/recraftv4_1_pro`. Aliases: `recraft-v4.1-pro`, `recraft-4.1-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `2048x2048`, `3072x1536`, `1536x3072`, `2560x1664`, `1664x2560`, `2432x1792`, `1792x2432`, `2304x1792`, `1792x2304`, `1664x2688`, `2560x1792`, `1792x2560`, `2688x1536`, `1536x2688` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_1_utility_pro

Recraft V4.1 Utility Pro. Output: image.

Full key: `recraft/recraftv4_1_utility_pro`. Aliases: `recraft-v4.1-utility-pro`, `recraft-4.1-utility-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `2048x2048`, `3072x1536`, `1536x3072`, `2560x1664`, `1664x2560`, `2432x1792`, `1792x2432`, `2304x1792`, `1792x2304`, `1664x2688`, `2560x1792`, `1792x2560`, `2688x1536`, `1536x2688` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_pro

Recraft V4 Pro. Output: image.

Full key: `recraft/recraftv4_pro`. Aliases: `recraft-v4-pro`, `recraft-4-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `2048x2048`, `3072x1536`, `1536x3072`, `2560x1664`, `1664x2560`, `2432x1792`, `1792x2432`, `2304x1792`, `1792x2304`, `1664x2688`, `2560x1792`, `1792x2560`, `2688x1536`, `1536x2688` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv3

Recraft V3. Output: image.

Full key: `recraft/recraftv3`. Aliases: `recraft-v3`, `recraft-3`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `1024x1024`, `2048x1024`, `1024x2048`, `1536x1024`, `1024x1536`, `1365x1024`, `1024x1365`, `1280x1024`, `1024x1280`, `1024x1707`, `1434x1024`, `1024x1434`, `1820x1024`, `1024x1820` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--negative-prompt` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv2

Recraft V2. Output: image.

Full key: `recraft/recraftv2`. Aliases: `recraft-v2`, `recraft-2`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `1024x1024`, `2048x1024`, `1024x2048`, `1536x1024`, `1024x1536`, `1365x1024`, `1024x1365`, `1280x1024`, `1024x1280`, `1024x1707`, `1434x1024`, `1024x1434`, `1820x1024`, `1024x1820` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--negative-prompt` | Optional | No declared value constraint; see the general flag and input rules. |

## recraftv4_1_vector

Recraft V4.1 Vector. Output: image.

Full key: `recraft/recraftv4_1_vector`. Aliases: `recraft-v4.1-vector`, `recraft-4.1-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_1_pro_vector

Recraft V4.1 Pro Vector. Output: image.

Full key: `recraft/recraftv4_1_pro_vector`. Aliases: `recraft-v4.1-pro-vector`, `recraft-4.1-pro-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_1_utility_vector

Recraft V4.1 Utility Vector. Output: image.

Full key: `recraft/recraftv4_1_utility_vector`. Aliases: `recraft-v4.1-utility-vector`, `recraft-4.1-utility-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_1_utility_pro_vector

Recraft V4.1 Utility Pro Vector. Output: image.

Full key: `recraft/recraftv4_1_utility_pro_vector`. Aliases: `recraft-v4.1-utility-pro-vector`, `recraft-4.1-utility-pro-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_vector

Recraft V4 Vector. Output: image.

Full key: `recraft/recraftv4_vector`. Aliases: `recraft-v4-vector`, `recraft-4-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv4_pro_vector

Recraft V4 Pro Vector. Output: image.

Full key: `recraft/recraftv4_pro_vector`. Aliases: `recraft-v4-pro-vector`, `recraft-4-pro-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv3_vector

Recraft V3 Vector. Output: image.

Full key: `recraft/recraftv3_vector`. Aliases: `recraft-v3-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--negative-prompt` | Optional | No declared value constraint; see the general flag and input rules. |
| `--strength` | Optional | Minimum: `0`; Maximum: `1`; Image editing requires --strength. |
| `--input-media` | Optional | Maximum inputs: `1` |

## recraftv2_vector

Recraft V2 Vector. Output: image.

Full key: `recraft/recraftv2_vector`. Aliases: `recraft-v2-vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--negative-prompt` | Optional | No declared value constraint; see the general flag and input rules. |

## recraftv4_styles

Recraft V4 Styles. Output: image.

Full key: `recraft/recraftv4_styles`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `1024x1024`, `1536x768`, `768x1536`, `1280x832`, `832x1280`, `1216x896`, `896x1216`, `1152x896`, `896x1152`, `832x1344`, `1280x896`, `896x1280`, `1344x768`, `768x1344` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--style-id` | Required | Use a Recraft style UUID compatible with this model, not a style name. See [Recraft’s style reference](https://www.recraft.ai/docs/api-reference/styles). |
| `--style-match` | Optional | precise follows the style closely; flexible permits more variation. See [Recraft’s V4 Styles guide](https://www.recraft.ai/docs/recraft-models/recraft-v4-styles). |

## recraftv4_styles_pro

Recraft V4 Styles Pro. Output: image.

Full key: `recraft/recraftv4_styles_pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--size` | Optional | Allowed: `2048x2048`, `3072x1536`, `1536x3072`, `2560x1664`, `1664x2560`, `2432x1792`, `1792x2432`, `2304x1792`, `1792x2304`, `1664x2688`, `2560x1792`, `1792x2560`, `2688x1536`, `1536x2688` |
| `--aspect-ratio` | Optional | Rule: Selects the supported image size closest to the requested aspect ratio. |
| `--resolution` | Optional | Rule: Selects a supported image size using the requested resolution. |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--style-id` | Required | Use a Recraft style UUID compatible with this model, not a style name. See [Recraft’s style reference](https://www.recraft.ai/docs/api-reference/styles). |
| `--style-match` | Optional | precise follows the style closely; flexible permits more variation. See [Recraft’s V4 Styles guide](https://www.recraft.ai/docs/recraft-models/recraft-v4-styles). |

## recraftv4_styles_vector

Recraft V4 Styles Vector. Output: image.

Full key: `recraft/recraftv4_styles_vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--style-id` | Required | Use a Recraft style UUID compatible with this model, not a style name. See [Recraft’s style reference](https://www.recraft.ai/docs/api-reference/styles). |
| `--style-match` | Optional | precise follows the style closely; flexible permits more variation. See [Recraft’s V4 Styles guide](https://www.recraft.ai/docs/recraft-models/recraft-v4-styles). |

## recraftv4_styles_pro_vector

Recraft V4 Styles Pro Vector. Output: image.

Full key: `recraft/recraftv4_styles_pro_vector`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `1:1`, `2:1`, `1:2`, `3:2`, `2:3`, `4:3`, `3:4`, `5:4`, `4:5`, `6:10`, `14:10`, `10:14`, `16:9`, `9:16` |
| `--num-images` | Optional | Minimum: `1`; Maximum: `6` |
| `--seed` | Optional | No declared value constraint; see the general flag and input rules. |
| `--style-id` | Required | Use a Recraft style UUID compatible with this model, not a style name. See [Recraft’s style reference](https://www.recraft.ai/docs/api-reference/styles). |
| `--style-match` | Optional | precise follows the style closely; flexible permits more variation. See [Recraft’s V4 Styles guide](https://www.recraft.ai/docs/recraft-models/recraft-v4-styles). |

Revised 2026-10-01
