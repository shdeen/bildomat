# Set Defaults and Credentials

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
