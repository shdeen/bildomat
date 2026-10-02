# Sourceful models

Provider ID: `sourceful`. Credential: `api-keys.sourceful` in the configuration file, or else the `SOURCEFUL_API_KEY` environment variable. Default model: `riverflow-2.5-fast`. Provider documentation: <https://www.riverflow.ai/app/platform-api/docs>.

[All providers](../providers-and-models.md) · [Flag types and shorthands](../generation-flags.md) · [Input and frame rules](../input-media.md)

Options default to unset unless supplied or derived. An unlisted option is unsupported by that model in Bildomat. Constraints below govern Bildomat’s adjustments; provider requirements can also apply.

## Models

- [`sourceful/riverflow-2.5-pro`](#riverflow-25-pro) — image.
- [`sourceful/riverflow-2.5-fast`](#riverflow-25-fast) — image.

## riverflow-2.5-pro

Riverflow 2.5 Pro. Output: image.

Full key: `sourceful/riverflow-2.5-pro`. Aliases: `riverflow`, `riverflow-v2.5-pro`, `riverflow-pro`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `auto`, `21:9`, `16:9`, `3:2`, `4:3`, `5:4`, `1:1`, `4:5`, `3:4`, `2:3`, `9:16` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--output-format` | Optional | Allowed: `webp`, `png`, `jpg`, `jpeg` |
| `--background` | Optional | Allowed: `original`, `transparent` |
| `--thinking-level` | Optional | Allowed: `low`, `medium`, `high`, `xhigh` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `10` |

## riverflow-2.5-fast

Riverflow 2.5 Fast. Output: image.

Full key: `sourceful/riverflow-2.5-fast`. Aliases: `riverflow-v2.5-fast`.

| Option | Requirement | Values and constraints |
| --- | --- | --- |
| `--aspect-ratio` | Optional | Allowed: `auto`, `21:9`, `16:9`, `3:2`, `4:3`, `5:4`, `1:1`, `4:5`, `3:4`, `2:3`, `9:16` |
| `--resolution` | Optional | Allowed: `1K`, `2K`, `4K` |
| `--output-format` | Optional | Allowed: `webp`, `png`, `jpg`, `jpeg` |
| `--background` | Optional | Allowed: `original`, `transparent` |
| `--thinking-level` | Optional | Allowed: `low`, `medium`, `high`, `xhigh` |
| `--prompt-upsampling` | Optional | No declared value constraint; see the general flag and input rules. |
| `--input-media` | Optional | Maximum inputs: `10` |

Revised 2026-10-01
