# Changelog

## [0.0.5] - 2026-09-28

First public release. A `bild` command for generating and editing images and videos from the terminal and for effective agentic use.

### Features

- Generate and edit images and videos from the terminal with one command, `bild`, which makes the request, polls for the result, and saves the generated media to disk.
- Provider-agnostic support for descriptor-class and adapter-class providers; current support for OpenAI, Google, xAI, Black Forest Labs, Recraft, Sourceful, Kling, and OpenRouter.
- Built-in model catalog with three non-API commands, `bild list`, `bild search`, and `bild info`, to list and search for providers and models and provide detailed information about each. No API calls are made for catalog commands, and no API key is needed.
- Shared generation options, i.e. the same flags, such as `--aspect-ratio`, `--resolution`, `--size`, and `--duration`, work across providers. (When a model does not accept a value, Bildomat uses the nearest value that the model accepts, or omits it, and reports the change.)
- Take input media with `-i` to edit an image or guide a result with local files or HTTP(S) URLs in PNG, JPEG, WebP, or MP4 format.
- Allow specifying video keyframes for opening and closing frames with a `first:` or `last:` prefix to input media and, for models that support timed keyframes, with exact number of seconds.
- Output flag `-o`/`--output-path` allows one argument to set directory, filename stem, and desired format. Format is honored if the model supports it; otherwise, save file with the format returned, and report the filename.  
- Ensure safe saving of media; for filenames in use, append a suffix beginning with a 2-digit '02',  and report the path of every file saved.
- Output options for better scriptability and agentic use. Specifying `-j`/`--json` returns a JSON structured result, `--print-filename` prints only saved paths, and `--save-results` writes the result to a file. Exit codes distinguish usage errors from other failures.
- Save thought text from supporting Gemini image models in a Markdown file beside the image by adding `--include-thoughts` flag.
- Install scripts for macOS and Linux (`install.sh`) and Windows (`install.ps1`).
- Configuration file in `~/.bildomat/config.yml` sets a default model, a default output directory, and provider API keys. Keys can also be set in environment variables. YAML keys and env vars are listed on each provider's info page.

[Unreleased]: https://github.com/shdeen/bildomat/compare/v0.0.5...HEAD
[0.0.5]: https://github.com/shdeen/bildomat/releases/tag/v0.0.5
