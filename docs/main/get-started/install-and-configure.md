# Install and Configure Bildomat

## Install

### Install Script

macOS and Linux:

```sh
curl -fsSL https://bildomat.com/install.sh | sh
```

Windows, in PowerShell:

```powershell
irm https://bildomat.com/install.ps1 | iex
```

### Homebrew

On macOS or Linux, with [Homebrew](https://brew.sh):

```sh
brew install shdeen/tap/bild
```

### Go

With [Go 1.26.5 or later](https://go.dev/dl/) on macOS, Linux, or Windows:

```sh
go install github.com/shdeen/bildomat/cmd/bild@latest
```

Go usually installs binaries to `~/go/bin`. Make sure that directory is on your `PATH`.

### Binaries

Download the archive for your system from the [Bildomat releases page](https://github.com/shdeen/bildomat/releases). Archives are available for macOS, Linux, and Windows, each on ARM and Intel/AMD processors.

## Add a Provider Key

Create a key with each provider whose models you want to use. A key from one provider is enough to start.

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

An OpenRouter key gives access to models from many vendors through OpenRouter. It does not work with those vendors' direct integrations.

Put each key in the configuration file or in an environment variable. A key in the configuration file takes precedence over the environment variable for the same provider.

## Use the Configuration File

Create `~/.bildomat/config.yml` to set provider keys, a default model, and a default output directory. All settings are optional. On Windows, the file is `%USERPROFILE%\.bildomat\config.yml`.

```yaml
default-model: openai/gpt-image-2
output-dir: ~/Pictures/bild
api-keys:
  google: YOUR_GOOGLE_API_KEY
  openai: YOUR_OPENAI_API_KEY
  xai: YOUR_XAI_API_KEY
  bfl: YOUR_BFL_API_KEY
  sourceful: YOUR_SOURCEFUL_API_KEY
  recraft: YOUR_RECRAFT_API_KEY
  kling: YOUR_KLING_API_KEY
  openrouter: YOUR_OPENROUTER_API_KEY
```

Replace each placeholder with your key for that provider, and omit the providers you do not use. Bildomat reads the file each time it runs a command; a standalone `bild --version` does not read it.

Without `default-model`, a run that omits `--model` uses the default model of the first keyed provider, in the order `bild list` shows providers, and `bild help` names the model in effect under `-m, --model`. Set `default-model` when you want one particular model whichever credentials are present.

## Use an Environment Variable

Set the variable for each provider whose key is not in the configuration file. Omit the lines for providers you do not use.

macOS and Linux:

```sh
export GOOGLE_API_KEY='YOUR_GOOGLE_API_KEY'
export OPENAI_API_KEY='YOUR_OPENAI_API_KEY'
export XAI_API_KEY='YOUR_XAI_API_KEY'
export BFL_API_KEY='YOUR_BFL_API_KEY'
export SOURCEFUL_API_KEY='YOUR_SOURCEFUL_API_KEY'
export RECRAFT_API_KEY='YOUR_RECRAFT_API_KEY'
export KLING_API_KEY='YOUR_KLING_API_KEY'
export OPENROUTER_API_KEY='YOUR_OPENROUTER_API_KEY'
```

To set the variables in every new terminal, add these lines to your shell's startup file, such as `~/.zshrc` or `~/.bashrc`.

Windows, in PowerShell:

```powershell
$env:GOOGLE_API_KEY = 'YOUR_GOOGLE_API_KEY'
$env:OPENAI_API_KEY = 'YOUR_OPENAI_API_KEY'
$env:XAI_API_KEY = 'YOUR_XAI_API_KEY'
$env:BFL_API_KEY = 'YOUR_BFL_API_KEY'
$env:SOURCEFUL_API_KEY = 'YOUR_SOURCEFUL_API_KEY'
$env:RECRAFT_API_KEY = 'YOUR_RECRAFT_API_KEY'
$env:KLING_API_KEY = 'YOUR_KLING_API_KEY'
$env:OPENROUTER_API_KEY = 'YOUR_OPENROUTER_API_KEY'
```

These assignments last for the current session. To set them in every session, add them to your PowerShell profile.

An environment variable supplies the key only when the configuration file has no nonempty key for that provider. To let the variable supply the key, remove or empty that provider's entry in the configuration file.

## Override a Default for One Request

```sh
bild --model xai/grok-imagine-image --output-path ./poster.png "A geometric poster for a coastal railway"
```

The `--model` flag overrides the configured model. Any `--output-path` overrides the configured output directory, and a relative path such as `poster.png` or `./poster.png` is relative to the working directory. The configured directory applies only to runs without `--output-path`.

## Check a Configuration Warning

A missing configuration file is normal and produces no warning. An unreadable or malformed file produces a warning and falls back to unconfigured settings. Unknown setting names produce warnings; recognized settings can still apply. A nonempty key for an unknown provider is ignored with a warning. Bildomat does not check `default-model` when it reads the file; an unknown model name there shows up in `bild help` and fails a run that omits `--model`. Warnings appear with generation, `bild help`, and the catalog commands, but not with `bild --version`.

See [configuration reference](../reference/configuration.md) for all provider identifiers, environment variables, path rules, and precedence.

Revised 2026-10-05
