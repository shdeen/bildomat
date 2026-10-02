# Bildomat Documentation

Bildomat (`bild`) generates and edits images and videos from the terminal. Use the same command options across providers, supply local files or URLs as references, and save the results where a script or agent can use them.

Generation requires an installed `bild` executable, network access, and credentials for the selected provider. Catalog commands require no API key and make no generation requests.

## Start here

[Generate and edit your first image](tutorials/first-image.md) follows one image from a prompt to a saved file and an edited version.

```sh
bild --model gemini --output-path ./paper-city.png "A paper city photographed in soft window light"
```

Generation options can come before or after the prompt. Quote a prompt containing spaces. Check [configuration and credentials](reference/configuration.md) for provider keys and default settings.

## How-to Guides

- [Find a model](how-to/find-a-model.md).
- [Choose image dimensions and format](how-to/control-size-and-format.md).
- [Edit an image or use references](how-to/use-input-media.md).
- [Generate and extend video](how-to/generate-video.md).
- [Control video with keyframes](how-to/use-keyframes.md).
- [Choose where files are saved](how-to/save-output-where-you-want.md).
- [Process images in scripts](how-to/use-bild-in-scripts.md).
- [Use Bildomat with an agent](how-to/use-bild-with-agents.md).
- [Save a model’s thoughts](how-to/keep-model-thoughts.md).
- [Set defaults and credentials](how-to/configure-bild.md).

## Reference

- [Commands and parsing](reference/commands.md).
- [Generation flags](reference/generation-flags.md).
- [Model names and aliases](reference/model-specifiers.md).
- [Parameter adjustment](reference/parameter-adjustment.md).
- [Input media and frame rules](reference/input-media.md).
- [Output files and reports](reference/output-files.md).
- [JSON output](reference/json-output.md).
- [Exit codes and errors](reference/exit-codes-and-errors.md).
- [Configuration, environment, and network limits](reference/configuration.md).
- [Providers and models](reference/providers-and-models.md).

## Explanation

[One command across different models](explanation/one-command-many-models.md) explains what stays consistent when you change providers, what the selected model controls, and how adjustments affect automation.

Revised 2026-10-01
