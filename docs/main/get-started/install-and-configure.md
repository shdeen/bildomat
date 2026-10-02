# Install and Configure Bildomat

## Install

macOS and Linux:

```sh
curl -fsSL https://raw.githubusercontent.com/shdeen/bildomat/main/scripts/install/install.sh | sh
```

Windows, in PowerShell:

```powershell
irm https://raw.githubusercontent.com/shdeen/bildomat/main/scripts/install/install.ps1 | iex
```

### Go

```sh
go install github.com/shdeen/bildomat/cmd/bild@latest
```

### Binaries

- [macOS, Apple silicon](https://github.com/shdeen/bildomat/releases/download/v0.0.5/bild_0.0.5_darwin_arm64.tar.gz)
- [macOS, Intel](https://github.com/shdeen/bildomat/releases/download/v0.0.5/bild_0.0.5_darwin_amd64.tar.gz)
- [Linux, ARM](https://github.com/shdeen/bildomat/releases/download/v0.0.5/bild_0.0.5_linux_arm64.tar.gz)
- [Linux, Intel/AMD](https://github.com/shdeen/bildomat/releases/download/v0.0.5/bild_0.0.5_linux_amd64.tar.gz)
- [Windows, ARM](https://github.com/shdeen/bildomat/releases/download/v0.0.5/bild_0.0.5_windows_arm64.zip)
- [Windows, Intel/AMD](https://github.com/shdeen/bildomat/releases/download/v0.0.5/bild_0.0.5_windows_amd64.zip)

## Add a Provider Key

Create a key with the provider whose models you want to use:

| Provider | Where to create a key | Configuration key | Environment variable |
| --- | --- | --- | --- |
| Google | [Google AI Studio](https://aistudio.google.com/apikey) | `google` | `GOOGLE_API_KEY` |
| OpenAI | [API keys](https://platform.openai.com/api-keys) | `openai` | `OPENAI_API_KEY` |
| xAI | [API keys](https://console.x.ai/team/default/api-keys) | `xai` | `XAI_API_KEY` |
| Black Forest Labs | [Dashboard](https://dashboard.bfl.ai/), under the project's API keys | `bfl` | `BFL_API_KEY` |
| Sourceful | [API keys](https://www.riverflow.ai/app/team-settings/api) | `sourceful` | `SOURCEFUL_API_KEY` |
| Recraft | [Profile, API](https://app.recraft.ai/profile/api) | `recraft` | `RECRAFT_API_KEY` |
| Kling | [API keys](https://kling.ai/dev/api-key) | `kling` | `KLING_API_KEY` |
| OpenRouter | [API keys](https://openrouter.ai/settings/keys) | `openrouter` | `OPENROUTER_API_KEY` |

Put the key in the configuration file or in an environment variable.

## Use the Configuration File

Create `~/.bildomat/config.yml` to choose a default model, default output directory, and provider keys. All settings are optional.

```yaml
default-model: google/gemini-3.1-flash-image
output-dir: ~/Pictures/bild
api-keys:
  google: YOUR_GOOGLE_API_KEY
  openai: YOUR_OPENAI_API_KEY
```

Replace placeholder keys with credentials for the providers you use. Omit providers you do not use. Bildomat reads the file each time it runs a command; a standalone `bild --version` does not read it.

Without `default-model`, a run that omits `--model` uses the default model of the first keyed provider, in the order `bild list` shows providers, and `bild help` names the model in effect under `-m, --model`. Set `default-model` when you want one particular model whichever credentials are present.

## Override a Default for One Request

```sh
bild --model openai/gpt-image-2 --output-path ./poster.png "A geometric poster for a coastal railway"
```

The model flag overrides the configured model. Any `--output-path` overrides the configured output directory, and a relative path such as `poster.png` or `./poster.png` is relative to the working directory. The configured directory applies only to runs without `--output-path`.

## Use an Environment Variable

```sh
export OPENAI_API_KEY='YOUR_API_KEY'
```

This works when there is no nonempty OpenAI key in the configuration file. A configured key wins over the environment variable. Remove or empty that configuration entry if you want the environment variable to supply the credential.

## Check a Configuration Warning

A missing configuration file is normal and produces no warning. An unreadable or malformed file produces a warning and falls back to unconfigured settings. Unknown setting names produce warnings; recognized settings can still apply. A nonempty key for an unknown provider is ignored with a warning. Bildomat does not check `default-model` when it reads the file; an unknown model name there shows up in `bild help` and fails a run that omits `--model`. Warnings appear with generation, `bild help`, and the catalog commands, but not with `bild --version`.

See [configuration reference](../reference/configuration.md) for all provider identifiers, environment variables, path rules, and precedence.

Revised 2026-10-01
